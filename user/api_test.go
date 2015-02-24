package user

import (
	"fmt"
	. "testing"

	"github.com/mediocregopher/mediocre-api/auth"
	"github.com/mediocregopher/mediocre-api/auth/authtest"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/stretchr/testify/assert"
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

func TestNewUser(t *T) {
	user, email, password := randStr(), randStr(), randStr()
	code, body := authtest.Req(testAPI, "POST", "/new-user", "",
		fmt.Sprintf(
			`{"Email":"%s","Username":"%s","Password":"%s"}`,
			user,
			email,
			password,
		),
	)
	assert.Equal(t, 200, code)
	assert.Equal(t, "", body)

	code, body = authtest.Req(testAPI, "POST", "/new-user", "",
		fmt.Sprintf(`{"Email":"%s", "Password":"%s"}`, email, password),
	)
	assert.Equal(t, 400, code)
	assert.Equal(t, ErrUserExists.Error()+"\n", body)
}
