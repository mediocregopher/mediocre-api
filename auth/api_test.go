package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	. "testing"

	"github.com/mediocregopher/mediocre-api/common"
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
				fmt.Fprintln(w, r.FormValue("_asUser"))
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

func assertReqErr(
	t *T, api http.Handler, method, endpnt, apiTok, userTok string,
	err common.ExpectedErr,
) {
	code, body := req(t, api, method, endpnt, apiTok, userTok)
	assert.Equal(t, err.Code, code)
	assert.Equal(t, err.Err+"\n", body)
}

func assertReq(
	t *T, api http.Handler, method, endpnt, apiTok, userTok, expectedBody string,
) {
	code, body := req(t, api, method, endpnt, apiTok, userTok)
	assert.Equal(t, 200, code)
	assert.Equal(t, expectedBody+"\n", body)
}

func TestNoAPIToken(t *T) {
	assertReq(t, testMux, "GET", "/foo", "", "", "foo")
}

func TestAPIToken(t *T) {
	apiTok := testAPI.NewAPIToken()
	assertReqErr(t, testMux, "GET", "/bar", "", "", ErrAPITokenMissing)
	assertReqErr(t, testMux, "GET", "/bar", "blah blah blah", "", ErrAPITokenInvalid)
	assertReq(t, testMux, "GET", "/bar", apiTok, "", "bar")
}

func TestUserToken(t *T) {
	username := "morty"
	apiTok := testAPI.NewAPIToken()
	userTok := testAPI.NewUserToken(username)

	assertReq(t, testMux, "GET", "/baz", apiTok, "", "baz")
	assertReqErr(t, testMux, "POST", "/baz", apiTok, "", ErrUserTokenMissing)
	assertReqErr(t, testMux, "POST", "/baz", apiTok, "blah blah blah", ErrUserTokenInvalid)
	assertReq(t, testMux, "POST", "/baz", apiTok, userTok, username+"\n"+username)
}
