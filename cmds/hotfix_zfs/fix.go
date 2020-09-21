package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cenkalti/backoff/v3"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/zos/pkg/app"
)

func download(url, output string) error {
	log.Info().Str("file", output).Str("url", url).Msg("downloading file")
	bf := backoff.NewExponentialBackOff()

	err := backoff.Retry(func() error {
		log.Info().Str("url", url).Msg("requesting url")
		response, err := http.Get(url)
		if err != nil {
			return err
		}

		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("invalid status code")
		}

		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			return err
		}

		defer file.Close()

		_, err = io.Copy(file, response.Body)
		return err
	}, bf)

	return err
}

func copyFile(dst, src string) error {
	log.Info().Str("source", src).Str("destination", dst).Msg("copy file")

	var (
		isNew  = false
		dstOld string
	)

	if _, err := os.Stat(dst); os.IsNotExist(err) {
		// case where this is a new file
		// we just need to copy from flist to root
		isNew = true
	}

	if !isNew {
		dstOld = dst + ".old"
		if err := os.Rename(dst, dstOld); err != nil {
			return err
		}
	}

	fSrc, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fSrc.Close()

	stat, err := fSrc.Stat()
	if err != nil {
		return err
	}

	fDst, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_SYNC, stat.Mode().Perm())
	if err != nil {
		return err
	}
	defer fDst.Close()

	if _, err = io.Copy(fDst, fSrc); err != nil {
		return err
	}

	if !isNew {
		return os.Remove(dstOld)
	}
	return nil
}

func main() {
	app.Initialize()
	target := "/bin"

	for name, url := range map[string]string{
		"g8ufs": "https://download.grid.tf/g8ufs",
	} {
		src := filepath.Join(os.TempDir(), name)
		if err := download(url, src); err != nil {
			log.Err(err).Str("name", name).Msg("failed to download binary")
			continue
		}

		dst := filepath.Join(target, name)
		copyFile(dst, src)
	}
}
