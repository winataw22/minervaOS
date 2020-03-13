package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/threefoldtech/zos/pkg/schema"
)

func handle(in io.Reader, pkg, dir string) error {
	sch, err := schema.New(in)
	if err != nil {
		return err
	}

	w, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s.go", pkg)))
	if err != nil {
		return err
	}

	return schema.GenerateGolang(w, pkg, sch)
}

func main() {
	var (
		pkg string
		dir string
		in  string
	)

	flag.StringVar(&pkg, "pkg", "schema", "package name in generated go code")
	flag.StringVar(&dir, "dir", ".", "directory to output files to")
	flag.StringVar(&in, "in", "", "directory with schema files. process all files in the directory. Otherwise process all input files given as extra args")

	flag.Parse()
	files := flag.Args()
	if len(in) != 0 {
		fs, err := ioutil.ReadDir(in)
		if err != nil {
			log.Fatalf("failed to list files in director(%s): %s", in, err)
		}
		for _, f := range fs {
			if f.IsDir() {
				continue
			}
			if !strings.HasSuffix(f.Name(), ".toml") {
				continue
			}

			files = append(files, filepath.Join(in, f.Name()))
		}
	}

	var readers []io.Reader

	for _, name := range files {
		fd, err := os.Open(name)
		if err != nil {
			log.Fatalf("failed to open file (%s) for reading; %s", name, err)
		}

		readers = append(readers, fd)
	}

	if err := handle(io.MultiReader(readers...), pkg, dir); err != nil {
		log.Fatalf("error while generating schema: %s", err)
	}
}
