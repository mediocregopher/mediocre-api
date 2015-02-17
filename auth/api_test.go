package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	. "testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testAPI, testMux = func() (*API, *http.ServeMux) {
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

	a.SetHandlerFlags("/baz", RequireUserAuthPost)
	s.HandleFunc("/baz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			fmt.Fprintln(w, a.GetUser(r))
		} else {
			fmt.Fprintln(w, "baz")
		}
	})

	return a, s
}()

func req(t *T, method, endpnt, apiTok, userTok string) (int, string) {
	r, err := http.NewRequest(method, endpnt, nil)
	require.Nil(t, err)
	if apiTok != "" {
		r.Header.Set("X-API-TOKEN", apiTok)
	}
	if userTok != "" {
		r.Header.Set("X-USER-TOKEN", userTok)
	}
	w := httptest.NewRecorder()
	testAPI.ServeHTTP(w, r)
	return w.Code, w.Body.String()
}

func TestNoAPIToken(t *T) {
	code, body := req(t, "GET", "/foo", "", "")
	assert.Equal(t, 200, code)
	assert.Equal(t, "foo\n", body)
}

func TestAPIToken(t *T) {
	apiTok := testAPI.NewAPIToken()

	code, body := req(t, "GET", "/bar", "", "")
	assert.Equal(t, 400, code)
	assert.Equal(t, APITokenMissing+"\n", body)

	code, body = req(t, "GET", "/bar", "blah blah blah", "")
	assert.Equal(t, 400, code)
	assert.Equal(t, APITokenInvalid+"\n", body)

	code, body = req(t, "GET", "/bar", apiTok, "")
	assert.Equal(t, 200, code)
	assert.Equal(t, "bar\n", body)
}

func TestUserToken(t *T) {
	username := "morty"
	apiTok := testAPI.NewAPIToken()
	userTok := testAPI.NewUserToken(username)

	code, body := req(t, "GET", "/baz", apiTok, "")
	assert.Equal(t, 200, code)
	assert.Equal(t, "baz\n", body)

	code, body = req(t, "POST", "/baz", apiTok, "")
	assert.Equal(t, 400, code)
	assert.Equal(t, UserTokenMissing+"\n", body)

	code, body = req(t, "POST", "/baz", apiTok, "blah blah blah")
	assert.Equal(t, 400, code)
	assert.Equal(t, UserTokenInvalid+"\n", body)

	code, body = req(t, "POST", "/baz", apiTok, userTok)
	assert.Equal(t, 200, code)
	assert.Equal(t, username+"\n", body)
}
