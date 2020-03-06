package main

import (
	"fmt"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/storage"
)

func main() {
	s, err := storage.New()
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}

	kind := pkg.SSDDevice
	total, err := s.Total(kind)
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}

	fmt.Printf("SSD: %v\n", total)

	kind = pkg.HDDDevice
	total, err = s.Total(kind)
	if err != nil {
		panic(fmt.Sprintf("%v", err))
	}

	fmt.Printf("HDD: %v\n", total)
}
