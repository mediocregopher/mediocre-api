// Package commontest holds methods which are helpful when writing tests within
// mediocre-api
package commontest

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"testing"

	"github.com/mediocregopher/mediocre-api/common"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// APIStarterKit returns an initialized *API and a Cmder which can be used as
// generic entities for testing
func APIStarterKit() util.Cmder {
	p, err := pool.New("tcp", "localhost:6379", 10)
	if err != nil {
		panic(err)
	}
	return p
}

// RandStr returns a string of random alphanumeric characters
func RandStr() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

//RandEmail returns a string which could plausibly be an email (but definitely
//isn't a real one)
func RandEmail() string {
	s := RandStr()
	return fmt.Sprintf("%s@%s.com", s[:4], s[4:8])
}

// Req is a method which makes testing http requests easier. It makes the
// request with the given method and body against the given endpoint, returning
// the return status code and body
func Req(
	t *testing.T, mux http.Handler, method, endpoint, body string,
) (
	int, string,
) {
	r, err := http.NewRequest(method, endpoint, bytes.NewBufferString(body))
	require.Nil(t, err, "\n%s", string(debug.Stack()))
	r.RemoteAddr = "1.1.1.1:50000"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w.Code, w.Body.String()
}

// AssertReq uses the stretchr/assert package to assert that the result of
// calling Req with the given arguments has a 200 response code and the given
// expectedBody
func AssertReq(
	t *testing.T, mux http.Handler, method, endpoint, body, expectedBody string,
) {
	code, body := Req(t, mux, method, endpoint, body)
	assert.Equal(t, 200, code, "\n%s", string(debug.Stack()))
	assert.Equal(t, expectedBody, body, "\n%s", string(debug.Stack()))
}

// AssertReqRaw uses the stretchr/assert package to assert that the result of
// executing the given *http.Request returns a 200 response code and the given
// expectedBody
func AssertReqRaw(
	t *testing.T, mux http.Handler, r *http.Request, expectedBody string,
) {
	r.RemoteAddr = "1.1.1.1:50000"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code, "\n%s", string(debug.Stack()))
	assert.Equal(t, expectedBody, w.Body.String(), "\n%s", string(debug.Stack()))
}

// AssertReqJSON uses the stretchr/assert package to assert that the result of
// calling Req with the given arguments has a 200 response and a body which is
// unmarshaled into dst successfully.
func AssertReqJSON(
	t *testing.T, mux http.Handler, method, endpoint, body string,
	dst interface{},
) {
	code, body := Req(t, mux, method, endpoint, body)
	assert.Equal(t, 200, code, "\n%s", string(debug.Stack()))

	err := json.Unmarshal([]byte(body), dst)
	require.Nil(t, err, "\n%s", string(debug.Stack()))
}

// AssertReqRawJSON uses the stretchr/assert package to assert that the result
// of executing the given *http.Request returns a 200 response and a body which
// is unmarshaled into dst successfully.
func AssertReqRawJSON(
	t *testing.T, mux http.Handler, r *http.Request, dst interface{},
) {
	r.RemoteAddr = "1.1.1.1:50000"
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code, "\n%s", string(debug.Stack()))
	err := json.Unmarshal([]byte(w.Body.String()), dst)
	require.Nil(t, err, "\n%s", string(debug.Stack()))
}

// AssertReqErr uses the stretchr/assert package to assert that the result of
// calling Req with the given arguments has a response code equal to the given
// ExpectedErr's and a body the same as well
func AssertReqErr(
	t *testing.T, mux http.Handler, method, endpoint, body string,
	err common.ExpectedErr,
) {
	code, body := Req(t, mux, method, endpoint, body)
	assert.Equal(t, err.Code, code, "\n%s", string(debug.Stack()))
	assert.Equal(t, err.Err+"\n", body, "\n%s", string(debug.Stack()))
}
