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
	email := commontest.RandEmail()
	password := commontest.RandStr()

	// Sanity check to make sure required parameters are being checked
	// correctly, we don't need to do this for all tests though
	code, body := authtest.Req(testAPI, "POST", "/new-user", "",
		fmt.Sprintf(
			`{"Username":"%s","Password":"%s"}`,
			user,
			password,
		),
	)
	assert.Equal(t, 400, code)
	assert.Equal(t, "field Email required\n", body)

	reqBody := fmt.Sprintf(
		`{"Email":"%s","Username":"%s","Password":"%s"}`,
		email,
		user,
		password,
	)

	code, body = authtest.Req(testAPI, "POST", "/new-user", "", reqBody)
	assert.Equal(t, 200, code)
	assert.Equal(t, "", body)

	code, body = authtest.Req(testAPI, "POST", "/new-user", "", reqBody)
	assert.Equal(t, 400, code)
	assert.Equal(t, ErrUserExists.Error()+"\n", body)
}
