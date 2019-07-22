package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tcnksm/go-input"
)

func confirm(s string) (string, error) {
	ans, err := input.DefaultUI().Ask(s, &input.Options{
		Default:     "Y",
		Required:    true,
		Loop:        true,
		HideDefault: true,
		ValidateFunc: func(s string) error {
			s = strings.ToLower(s)
			if s != "y" && s != "n" {
				return fmt.Errorf("input must be y or n")
			}
			return nil
		},
	})
	if err != nil {
		return "", err
	}
	return strings.ToLower(ans), nil
}

func output(name string, workload interface{}) (err error) {
	var w io.Writer = os.Stdout
	if name != "" {
		w, err = os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
		if err != nil {
			return err
		}
	}

	return json.NewEncoder(w).Encode(workload)
}
