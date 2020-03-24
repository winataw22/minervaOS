package main

import (
	"encoding/json"
	"io"
	"os"
)

func writeWorkload(name string, workload interface{}) (err error) {
	var w io.Writer = os.Stdout
	if name != "" {
		w, err = os.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
		if err != nil {
			return err
		}
	}

	return json.NewEncoder(w).Encode(workload)
}
