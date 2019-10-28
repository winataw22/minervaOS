package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func httpError(w http.ResponseWriter, err error, code int) {
	b, err := json.Marshal(struct {
		Error error
	}{
		Error: err,
	})
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	fmt.Fprintln(w, b)
}
