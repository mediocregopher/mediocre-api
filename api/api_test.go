package api

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	. "testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI(t *T) {
	secret := []byte("wubalubadubdub!")
	s := http.NewServeMux()
	a := NewAPI(s)
	a.Secret = secret

	a.SetHandlerFlags("/foo", NoAPITokenRequired)
	s.HandleFunc("/foo", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "foo")
	})

	s.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "bar")
	})

	r, err := http.NewRequest("GET", "/foo", nil)
	require.Nil(t, err)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "foo\n", w.Body.String())

	r, err = http.NewRequest("GET", "/bar", nil)
	require.Nil(t, err)
	w = httptest.NewRecorder()
	a.ServeHTTP(w, r)
	assert.Equal(t, 400, w.Code)
	assert.Equal(t, APITokenMissing+"\n", w.Body.String())

	r, err = http.NewRequest("GET", "/bar", nil)
	require.Nil(t, err)
	r.Header.Set("X-API-TOKEN", "something awful")
	w = httptest.NewRecorder()
	a.ServeHTTP(w, r)
	assert.Equal(t, 400, w.Code)
	assert.Equal(t, APITokenInvalid+"\n", w.Body.String())

	tok := a.NewAPIToken()

	r, err = http.NewRequest("GET", "/bar", nil)
	require.Nil(t, err)
	r.Header.Set("X-API-TOKEN", tok)
	w = httptest.NewRecorder()
	a.ServeHTTP(w, r)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "bar\n", w.Body.String())

}

func TestRequestSignData(t *T) {
	r, err := http.NewRequest("GET", "http://localhost/bar?this=that", bytes.NewBuffer([]byte("hi there")))
	require.Nil(t, err)

	b := requestSignData(r, false)
	assert.Equal(t, "GEThttp://localhost/bar?this=that", string(b))

	b = requestSignData(r, true)
	assert.Equal(t, "GEThttp://localhost/bar?this=thathi there", string(b))

	// Make sure that even after checking the POST data we can still read the
	// body (requestSignData is supposed to re-buffer it and replace the
	// original buffer)
	body, err := ioutil.ReadAll(r.Body)
	require.Nil(t, err)
	assert.Equal(t, "hi there", string(body))
}
