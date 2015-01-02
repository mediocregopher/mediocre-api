package api

import (
	"fmt"
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
