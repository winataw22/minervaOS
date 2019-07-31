package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/threefoldtech/zosv2/modules/schema"
)

func handle(file, pkg, dir string) error {
	fd, err := os.Open(file)
	if err != nil {
		return err
	}

	defer fd.Close()

	sch, err := schema.New(fd)
	if err != nil {
		return err
	}
	out := path.Base(fd.Name())
	w, err := os.Create(path.Join(dir, fmt.Sprintf("%s.go", out)))
	if err != nil {
		return err
	}

	return schema.GenerateGolang(w, pkg, sch)
}

func main() {
	var (
		pkg string
		dir string
	)

	flag.StringVar(&pkg, "pkg", "schema", "package name in generated go code")
	flag.StringVar(&dir, "dir", ".", "directory to output files to")

	flag.Parse()

	for _, input := range flag.Args() {
		if err := handle(input, pkg, dir); err != nil {
			fmt.Fprintf(os.Stderr, "Error while generating schema: %s\n", err)
			os.Exit(1)
		}
	}
}
