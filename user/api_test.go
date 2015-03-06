package user

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	. "testing"

	"github.com/mediocregopher/mediocre-api/auth"
	"github.com/mediocregopher/mediocre-api/auth/authtest"
	"github.com/mediocregopher/mediocre-api/common/commontest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testAPI = func() *auth.API {
	return NewMux(commontest.APIStarterKit()).(*auth.API)
}()

func testAPICreateUser(t *T) (string, string, string) {
	user := commontest.RandStr()
	email := commontest.RandEmail()
	password := commontest.RandStr()

	reqBody := fmt.Sprintf(
		`{"Email":"%s","Username":"%s","Password":"%s"}`,
		email,
		user,
		password,
	)

	code, body := authtest.Req(testAPI, "POST", "/new-user", "", reqBody)
	assert.Equal(t, 200, code)
	assert.Equal(t, "", body)
	return user, email, password
}

func TestAPINewUser(t *T) {
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

// TestAPIUserToken tests retrieving a user token from the api. Essentially,
// logging in
func TestAPIUserToken(t *T) {
	user, _, password := testAPICreateUser(t)
	url := fmt.Sprintf("/%s/token", user)

	reqBody := `{"Password":"aaaaaa"}`
	code, body := authtest.Req(testAPI, "GET", url, "", reqBody)
	assert.Equal(t, 400, code)
	assert.Equal(t, ErrBadAuth.Error()+"\n", body)

	reqBody = fmt.Sprintf(`{"Password":"%s"}`, password)
	code, body = authtest.Req(testAPI, "GET", url, "", reqBody)
	assert.Equal(t, 200, code)
	s := struct{ Token string }{}
	assert.Nil(t, json.Unmarshal([]byte(body), &s))
	assert.NotEqual(t, "", s.Token)
}

func requireJSONUnmarshal(t *T, body string, i interface{}) {
	require.Nil(t, json.Unmarshal([]byte(body), &i), string(debug.Stack()))
}

func TestAPIUserGet(t *T) {
	user, email, _ := testAPICreateUser(t)
	url := fmt.Sprintf("/%s", user)
	var i PrivateInfo

	code, body := authtest.Req(testAPI, "GET", url, "", "")
	assert.Equal(t, 200, code)
	requireJSONUnmarshal(t, body, &i)
	assert.Equal(t, i.Name, user)
	assert.Equal(t, i.Email, "")

	code, body = authtest.Req(testAPI, "GET", url, user, "")
	assert.Equal(t, 200, code)
	requireJSONUnmarshal(t, body, &i)
	assert.Equal(t, i.Name, user)
	assert.Equal(t, i.Email, email)

	user404 := commontest.RandStr()
	url = fmt.Sprintf("/%s", user404)
	code, body = authtest.Req(testAPI, "GET", url, "", "")
	assert.Equal(t, 404, code)
	assert.Equal(t, ErrNotFound.Error()+"\n", body)
}
