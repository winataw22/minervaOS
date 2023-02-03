package mw

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
)

// Response interface
type Response interface {
	Status() int
	Err() error

	// header getter
	Header() http.Header
	// header setter
	WithHeader(k, v string) Response
}

// Action interface
type Action func(r *http.Request) (interface{}, Response)

// AsHandlerFunc is a helper wrapper to make implementing actions easier
func AsHandlerFunc(a Action) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			_, _ = io.ReadAll(r.Body)
			_ = r.Body.Close()
		}()

		object, result := a(r)

		w.Header().Set("Content-Type", "application/json")

		if result == nil {
			w.WriteHeader(http.StatusOK)
		} else {

			h := result.Header()
			for k := range h {
				for _, v := range h.Values(k) {
					w.Header().Add(k, v)
				}
			}

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
	header http.Header
}

func (r genericResponse) Status() int {
	return r.status
}

func (r genericResponse) Err() error {
	return r.err
}

func (r genericResponse) Header() http.Header {
	if r.header == nil {
		r.header = http.Header{}
	}
	return r.header
}

func (r genericResponse) WithHeader(k, v string) Response {
	if r.header == nil {
		r.header = http.Header{}
	}

	r.header.Add(k, v)
	return r
}

// Created return a created response
func Created() Response {
	return genericResponse{status: http.StatusCreated}
}

// Ok return a ok response
func Ok() Response {
	return genericResponse{status: http.StatusOK}
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

// PaymentRequired result
func PaymentRequired(err error) Response {
	return Error(err, http.StatusPaymentRequired)
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

// Forbidden response
func Forbidden(err error) Response {
	return Error(err, http.StatusForbidden)
}

// NoContent response
func NoContent() Response {
	return genericResponse{status: http.StatusNoContent}
}

// Accepted response
func Accepted() Response {
	return genericResponse{status: http.StatusAccepted}
}

// Unavailable returned when server is too busy
func Unavailable(err error) Response {
	return Error(err, http.StatusServiceUnavailable)
}
