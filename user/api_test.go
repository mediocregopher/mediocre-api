package user

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	. "testing"

	"github.com/mediocregopher/mediocre-api/common"
	"github.com/mediocregopher/mediocre-api/common/commontest"
	"github.com/mediocregopher/mediocre-api/pickyjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testMux = func() http.Handler {
	cmder := commontest.APIStarterKit()
	return NewMux(cmder)
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

	commontest.AssertReq(t, testMux, "POST", "/new-user", reqBody, "")
	return user, email, password
}

func TestAPINewUser(t *T) {
	user := commontest.RandStr()
	email := commontest.RandEmail()
	password := commontest.RandStr()

	reqBody := fmt.Sprintf(`{"Username":"%s","Password":"%s"}`, user, password)

	// Sanity check to make sure required parameters are being checked
	// correctly, we don't need to do this for all tests though
	expectedErr := pickyjson.ErrFieldRequiredf("Email").(common.ExpectedErr)
	commontest.AssertReqErr(t, testMux, "POST", "/new-user", reqBody, expectedErr)

	reqBody = fmt.Sprintf(
		`{"Email":"%s","Username":"%s","Password":"%s"}`,
		email,
		user,
		password,
	)
	commontest.AssertReq(t, testMux, "POST", "/new-user", reqBody, "")
	commontest.AssertReqErr(t, testMux, "POST", "/new-user", reqBody, ErrUserExists)
}

// TestAPIUserAuth tests retrieving a user token from the api. Essentially,
// logging in
func TestAPIUserAuth(t *T) {
	user, _, password := testAPICreateUser(t)
	url := fmt.Sprintf("/%s/auth", user)

	reqBody := `{"Password":"aaaaaa"}`
	commontest.AssertReqErr(t, testMux, "POST", url, reqBody, ErrBadAuth)

	reqBody = fmt.Sprintf(`{"Password":"%s"}`, password)
	commontest.AssertReq(t, testMux, "POST", url, reqBody, "")
}

func requireJSONUnmarshal(t *T, body string, i interface{}) {
	require.Nil(t, json.Unmarshal([]byte(body), &i), string(debug.Stack()))
}

func TestAPIUserGet(t *T) {
	user, email, _ := testAPICreateUser(t)
	url := fmt.Sprintf("/%s", user)
	var i Info

	code, body := commontest.Req(t, testMux, "GET", url, "")
	assert.Equal(t, 200, code)
	requireJSONUnmarshal(t, body, &i)
	assert.Equal(t, user, i["Name"])
	assert.Equal(t, "", i["Email"])

	code, body = commontest.Req(t, testMux, "GET", url+"?_asUser="+user, "")
	assert.Equal(t, 200, code)
	requireJSONUnmarshal(t, body, &i)
	assert.Equal(t, user, i["Name"])
	assert.Equal(t, email, i["Email"])

	user404 := commontest.RandStr()
	url = fmt.Sprintf("/%s", user404)
	commontest.AssertReqErr(t, testMux, "GET", url, "", ErrNotFound)
}
