package main

import (
	"fmt"
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
