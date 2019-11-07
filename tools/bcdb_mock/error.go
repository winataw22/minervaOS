package main

import (
	"encoding/json"
	"net/http"
)

func httpError(w http.ResponseWriter, err error, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(struct {
		Error string
	}{
		Error: err.Error(),
	})
}
