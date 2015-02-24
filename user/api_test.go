package user

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	. "testing"

	"github.com/mediocregopher/mediocre-api/auth"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testSecret = []byte("woopwoop")

var testAPI = func() *auth.API {
	p, err := pool.New("tcp", "localhost:6379", 10)
	if err != nil {
		panic(err)
	}

	o := auth.NewAPIOpts()
	o.Secret = testSecret
	return NewMux(o, p).(*auth.API)
}()

func req(t *T, method, endpoint, user, body string) (int, string) {
	r, err := http.NewRequest(method, endpoint, bytes.NewBufferString(body))
	require.Nil(t, err)

	r.Header.Set("X-API-TOKEN", testAPI.NewAPIToken())
	if user != "" {
		r.Header.Set("X-USER-TOKEN", testAPI.NewUserToken(user))
	}

	w := httptest.NewRecorder()
	testAPI.ServeHTTP(w, r)
	return w.Code, w.Body.String()
}

func TestNewUser(t *T) {
	user, email, password := randStr(), randStr(), randStr()
	code, body := req(t, "POST", "/new-user", "",
		fmt.Sprintf(
			`{"Email":"%s","Username":"%s","Password":"%s"}`,
			user,
			email,
			password,
		),
	)
	assert.Equal(t, 200, code)
	assert.Equal(t, "", body)

	code, body = req(t, "POST", "/new-user", "",
		fmt.Sprintf(`{"Email":"%s", "Password":"%s"}`, email, password),
	)
	assert.Equal(t, 400, code)
	assert.Equal(t, ErrUserExists.Error()+"\n", body)
}
