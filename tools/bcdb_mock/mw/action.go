package mw

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

// Response interface
type Response interface {
	Status() int
	Err() error
}

// Action interface
type Action func(r *http.Request) (interface{}, Response)

// AsHandlerFunc is a helper wrapper to make implementing actions easier
func AsHandlerFunc(a Action) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		object, result := a(r)
		w.Header().Set("Content-Type", "application/json")
		if result == nil {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(result.Status())
			if err := result.Err(); err != nil {
				log.Error().Msgf("%s", err.Error())
				object = struct {
					Error string `json:"error"`
				}{
					Error: err.Error(),
				}
			}
		}

		if err := json.NewEncoder(w).Encode(object); err != nil {
			log.Error().Err(err).Msg("failed to encode return object")
		}
	}
}

type genericResponse struct {
	status int
	err    error
}

func (r genericResponse) Status() int {
	return r.status
}

func (r genericResponse) Err() error {
	return r.err
}

// Created return a created response
func Created() Response {
	return genericResponse{status: http.StatusCreated}
}

// Error generic error response
func Error(err error, code ...int) Response {
	status := http.StatusInternalServerError
	if len(code) > 0 {
		status = code[0]
	}

	if err == nil {
		err = fmt.Errorf("no message")
	}

	return genericResponse{status: status, err: err}
}

// BadRequest result
func BadRequest(err error) Response {
	return Error(err, http.StatusBadRequest)
}

// NotFound response
func NotFound(err error) Response {
	return Error(err, http.StatusNotFound)
}

// Conflict response
func Conflict(err error) Response {
	return Error(err, http.StatusConflict)
}

// UnAuthorized response
func UnAuthorized(err error) Response {
	return Error(err, http.StatusUnauthorized)
}
