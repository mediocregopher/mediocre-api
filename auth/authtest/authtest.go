// Package authtest conains a few methods which are useful for testing APIs
// which use the auth wrapper
package authtest

import (
	"bytes"
	"net/http"
	"net/http/httptest"

	"github.com/mediocregopher/mediocre-api/auth"
)

// Req performs a request on the given API/mux instances, returning the return
// http code and body. If user is not empty a user token will be automatically
// created
func Req(
	a *auth.API, mux http.Handler, method, endpoint, user, body string,
) (
	int, string,
) {
	r, err := http.NewRequest(method, endpoint, bytes.NewBufferString(body))
	if err != nil {
		panic(err)
	}
	r.RemoteAddr = "2.2.2.2:50000"

	r.Header.Set(auth.APITokenHeader, a.NewAPIToken())
	if user != "" {
		r.Header.Set(auth.UserTokenHeader, a.NewUserToken(user))
	}

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w.Code, w.Body.String()
}
