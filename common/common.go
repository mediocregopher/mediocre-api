// Package common is used to hold things that multiple packags within the
// mediocre-api toolkit will need to use or reference
package common

import (
	"fmt"
	"log"
	"net/http"
)

// ExpectedErr is an implementation of the error interface which will be used to
// indicate that the error being returned is an expected one and can sent back
// to the client
type ExpectedErr struct {
	Code int
	Err  string
}

// ExpectedErrf returns an ExpectedErr with a formatted message
func ExpectedErrf(code int, s string, args ...interface{}) ExpectedErr {
	return ExpectedErr{
		Code: code,
		Err:  fmt.Sprintf(s, args...),
	}
}

// Error implements the error interface
func (e ExpectedErr) Error() string {
	return e.Err
}

// HTTPError will attempt to cast the given error to an ExpectedErr. If it's
// able to it will write that error and its response code back to the
// http.ResponseWriter. Otherwise it will log the error and send back a 500
// unknown server-side error. If err is nil it will do nothing.
func HTTPError(w http.ResponseWriter, r *http.Request, err error) {
	if eerr, ok := err.(ExpectedErr); ok {
		http.Error(w, eerr.Error(), eerr.Code)
	} else if err != nil {
		log.Printf("%s %s -> %s", r.Method, r.URL, err)
		http.Error(w, "unknown server-side error", 500)
	}
}
