package user

import (
	"fmt"
	. "testing"

	"github.com/mediocregopher/mediocre-api/auth"
	"github.com/mediocregopher/mediocre-api/auth/authtest"
	"github.com/mediocregopher/mediocre-api/common/commontest"
	"github.com/stretchr/testify/assert"
)

var testAPI = func() *auth.API {
	return NewMux(commontest.APIStarterKit()).(*auth.API)
}()

func TestNewUser(t *T) {
	user := commontest.RandStr()
	email := commontest.RandStr()
	password := commontest.RandStr()

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
