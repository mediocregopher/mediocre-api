package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	. "testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testAPI, testMux = func() (*API, *http.ServeMux) {
	s := http.NewServeMux()
	a := NewAPI()
	a.Secret = []byte("wubalubadubdub!")

	s.Handle("/foo", a.WrapHandlerFunc(
		NoAPITokenRequired,
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "foo")
		},
	))

	s.Handle("/bar", a.WrapHandlerFunc(
		Default,
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "bar")
		},
	))

	s.Handle("/baz", a.WrapHandlerFunc(
		RequireUserAuthPost,
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				fmt.Fprintln(w, a.GetUser(r))
			} else {
				fmt.Fprintln(w, "baz")
			}
		},
	))

	return a, s
}()

func req(
	t *T, api http.Handler, method, endpnt, apiTok, userTok string,
) (
	int, string,
) {
	r, err := http.NewRequest(method, endpnt, nil)
	require.Nil(t, err)
	r.RemoteAddr = "1.1.1.1:50000"
	if apiTok != "" {
		r.Header.Set(APITokenHeader, apiTok)
	}
	if userTok != "" {
		r.Header.Set(UserTokenHeader, userTok)
	}
	w := httptest.NewRecorder()
	api.ServeHTTP(w, r)
	return w.Code, w.Body.String()
}

func TestNoAPIToken(t *T) {
	code, body := req(t, testMux, "GET", "/foo", "", "")
	assert.Equal(t, 200, code)
	assert.Equal(t, "foo\n", body)
}

func TestAPIToken(t *T) {
	apiTok := testAPI.NewAPIToken()

	code, body := req(t, testMux, "GET", "/bar", "", "")
	assert.Equal(t, 400, code)
	assert.Equal(t, APITokenMissing+"\n", body)

	code, body = req(t, testMux, "GET", "/bar", "blah blah blah", "")
	assert.Equal(t, 400, code)
	assert.Equal(t, APITokenInvalid+"\n", body)

	code, body = req(t, testMux, "GET", "/bar", apiTok, "")
	assert.Equal(t, 200, code)
	assert.Equal(t, "bar\n", body)
}

func TestUserToken(t *T) {
	username := "morty"
	apiTok := testAPI.NewAPIToken()
	userTok := testAPI.NewUserToken(username)

	code, body := req(t, testMux, "GET", "/baz", apiTok, "")
	assert.Equal(t, 200, code)
	assert.Equal(t, "baz\n", body)

	code, body = req(t, testMux, "POST", "/baz", apiTok, "")
	assert.Equal(t, 400, code)
	assert.Equal(t, UserTokenMissing+"\n", body)

	code, body = req(t, testMux, "POST", "/baz", apiTok, "blah blah blah")
	assert.Equal(t, 400, code)
	assert.Equal(t, UserTokenInvalid+"\n", body)

	code, body = req(t, testMux, "POST", "/baz", apiTok, userTok)
	assert.Equal(t, 200, code)
	assert.Equal(t, username+"\n", body)
}

var testBuiltinAPI, testBuiltinMux = func() (*API, http.Handler) {
	m, a := NewMux([]byte("turtles"))
	return a, m
}()

func TestBulitinAPIToken(t *T) {
	code, body := req(t, testBuiltinMux, "GET", "/token", "", "")
	assert.Equal(t, 200, code)
	s := struct{ Token string }{}
	assert.Nil(t, json.Unmarshal([]byte(body), &s))
	assert.NotEqual(t, "", s.Token)
}
