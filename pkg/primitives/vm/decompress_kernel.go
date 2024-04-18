package vm

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"debug/elf"
	"fmt"
	"io"
	"os"
	"strings"

	// "github.com/cyberdelia/lzo" --> it requires liblzo2-dev to be installed

	"github.com/hashicorp/go-multierror"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
	"github.com/rs/zerolog/log"
	"github.com/ulikunitz/xz/lzma"
	"github.com/xi2/xz"
)

func isValidELFKernel(KernelImagePath string) error {
	f, err := elf.Open(KernelImagePath)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

type algoOptions struct {
	decompressFunc func(kernelStream io.Reader) (io.Reader, error)
	headerBytes    []byte
	name           string
}

func decompressData(data []byte, tmpFile *os.File, o algoOptions) error {
	var headerIndex int
	var errs error

	headerCount := bytes.Count(data, o.headerBytes)
	if headerCount == 0 {
		return fmt.Errorf("%s: couldn't find the compression algorithm header", o.name)
	}

	for i := 0; i < headerCount; i++ {
		headerIndex += bytes.Index(data[headerIndex:], o.headerBytes)

		headerData := data[headerIndex:]

		// to ignore the current header in the next iteration
		headerIndex += len(o.headerBytes)

		r, err := o.decompressFunc(bytes.NewBuffer(headerData))
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		// close zstd reader after write
		if reader, ok := r.(*zstd.Decoder); ok {
			defer reader.Close()
		}

		_, err = io.Copy(tmpFile, r)
		if err != nil && err != bzip2.StructuralError("bad magic value in continuation file") && err != zstd.ErrMagicMismatch {
			if o.name == "zstd" {
				err = fmt.Errorf("%s: %v", o.name, err)
			}
			errs = multierror.Append(errs, err)
			continue
		}

		if err = isValidELFKernel(tmpFile.Name()); err == nil {
			return nil
		}

		errs = multierror.Append(errs, err)
	}

	return errs
}

func gUnzip(kernelStream io.Reader) (io.Reader, error) {
	r, err := gzip.NewReader(kernelStream)
	if err != nil {
		return nil, err
	}

	r.Multistream(false)
	// TODO: how it read after closing !!!!
	defer r.Close()

	return r, nil
}

func unXZ(kernelStream io.Reader) (io.Reader, error) {
	r, err := xz.NewReader(kernelStream, 0)
	if err != nil {
		return nil, err
	}

	r.Multistream(false)
	return r, nil
}

func bUnzip2(kernelStream io.Reader) (io.Reader, error) {
	return bzip2.NewReader(kernelStream), nil
}

func unlzma(kernelStream io.Reader) (io.Reader, error) {
	return lzma.NewReader(kernelStream)
}

func lZop(kernelStream io.Reader) (io.Reader, error) {
	return nil, fmt.Errorf("lzo: algorithm is not supported yet")
}

func lZ4(kernelStream io.Reader) (io.Reader, error) {
	return lz4.NewReader(kernelStream), nil
}

func unZstd(kernelStream io.Reader) (io.Reader, error) {
	r, err := zstd.NewReader(kernelStream)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func tryDecompressKernel(kernelImagePath string) error {
	if len(strings.TrimSpace(kernelImagePath)) == 0 {
		return fmt.Errorf("kernel image is required")
	}

	// if kernel is already an elf (uncompressed)
	if err := isValidELFKernel(kernelImagePath); err == nil {
		log.Debug().Msg("kernel is decompressed")
		return nil
	}

	log.Debug().Msg("kernel is compressed, trying to decompress kernel")

	// Prepare temp files:
	tmpFile, err := os.CreateTemp("", "vmlinux-")
	if err != nil {
		return err
	}

	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	kernelData, err := os.ReadFile(kernelImagePath)
	if err != nil {
		return err
	}

	algos := []algoOptions{
		{
			decompressFunc: gUnzip,
			headerBytes:    []byte("\037\213\010"),
			name:           "gunzip",
		},
		{
			decompressFunc: unXZ,
			headerBytes:    []byte("\3757zXZ\000"),
			name:           "unxz",
		},
		{
			decompressFunc: bUnzip2,
			headerBytes:    []byte("BZh"),
			name:           "bunzip2",
		},
		{
			decompressFunc: unlzma,
			headerBytes:    []byte("\135\000\000\000"),
			name:           "unlzma",
		},
		{
			decompressFunc: lZop,
			headerBytes:    []byte("\211\114\132"),
			name:           "lzop",
		},
		{
			decompressFunc: lZ4,
			headerBytes:    []byte("\002!L\030"),
			name:           "lz4",
		},
		{
			decompressFunc: unZstd,
			headerBytes:    []byte("(\265/\375"),
			name:           "zstd",
		},
	}

	var errs error

	for _, algo := range algos {
		if err = decompressData(kernelData, tmpFile, algo); err == nil {
			if _, err = tmpFile.Seek(0, io.SeekStart); err != nil {
				return err
			}

			f, err := os.Create(kernelImagePath)
			if err != nil {
				return err
			}

			defer f.Close()

			if _, err := io.Copy(f, tmpFile); err != nil {
				return err
			}

			log.Debug().Str("algorithm", algo.name).Msg("kernel is decompressed")
			return nil
		}

		errs = multierror.Append(errs, err)
	}

	return errs
}
