// Package apihelper contains some methods which make working with the http
// package a bit nicer
package apihelper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mediocregopher/mediocre-api/common"
	"github.com/mediocregopher/mediocre-api/pickyjson"
)

// ErrUnlessMethod checks that the given request is using one of the given
// HTTP methods. If it is not than an error is sent back to the client and true
// is returned
func ErrUnlessMethod(
	w http.ResponseWriter, r *http.Request, methods ...string,
) bool {
	for i := range methods {
		if r.Method == methods[i] {
			return false
		}
	}
	http.Error(w, "invalid method", 400)
	return false
}

// Prepare takes in a request and its response, and performs the following
// checks/enhancements:
//
// * If any methods are given, ensure the request is using one of them.
// Otherwise send error to client and return false
//
// * Replace r.Body with a MaxBytesReader which will stop the reading at the
// given bodySizeLimit
//
// * If params isn't nil attempt to json.Unmarshal the request body into it. If
// that fails an error is sent to the client and false is returned
//
func Prepare(
	w http.ResponseWriter, r *http.Request, params interface{},
	bodySizeLimit int64,
) bool {
	r.Body = http.MaxBytesReader(w, r.Body, bodySizeLimit)
	if params != nil {
		if err := json.NewDecoder(r.Body).Decode(params); err != nil {
			http.Error(w, err.Error(), 400)
			return false
		}
		if err := pickyjson.CheckRequired(params); err != nil {
			common.HTTPError(w, r, err)
			return false
		}
		if err := pickyjson.CheckRequired(&params); err != nil {
			common.HTTPError(w, r, err)
			return false
		}
	}

	return true
}

// JSONSuccess json encodes the given return value and writes that to the given
// io.Writer (presumably an http.ResponseWriter)
func JSONSuccess(w io.Writer, i interface{}) {
	json.NewEncoder(w).Encode(i)
	fmt.Fprintf(w, "\n")
}
