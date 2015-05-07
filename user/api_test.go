package user

import (
	"fmt"
	"net/http"
	. "testing"

	"github.com/mediocregopher/mediocre-api/common"
	"github.com/mediocregopher/mediocre-api/common/commontest"
	"github.com/mediocregopher/mediocre-api/pickyjson"
	"github.com/stretchr/testify/assert"
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

func TestAPIUserGet(t *T) {
	user, email, _ := testAPICreateUser(t)
	url := fmt.Sprintf("/%s", user)
	var i Info

	commontest.AssertReqJSON(t, testMux, "GET", url, "", &i)
	assert.Equal(t, user, i["Name"])
	assert.Equal(t, "", i["Email"])

	commontest.AssertReqJSON(t, testMux, "GET", url+"?_asUser="+user, "", &i)
	assert.Equal(t, user, i["Name"])
	assert.Equal(t, email, i["Email"])

	user404 := commontest.RandStr()
	url = fmt.Sprintf("/%s", user404)
	commontest.AssertReqErr(t, testMux, "GET", url, "", ErrNotFound)
}

func TestAPIUserSet(t *T) {
	user, email, _ := testAPICreateUser(t)
	url := fmt.Sprintf("/%s", user)
	urlAs := fmt.Sprintf("/%s?_asUser=%s", user, user)
	newEmail := "foo_" + email

	reqBody := fmt.Sprintf(`{"Email":"%s"}`, newEmail)
	commontest.AssertReqErr(t, testMux, "POST", url, reqBody, ErrBadAuth)
	commontest.AssertReq(t, testMux, "POST", urlAs, reqBody, "")

	var i Info
	commontest.AssertReqJSON(t, testMux, "GET", urlAs, "", &i)
	assert.Equal(t, user, i["Name"])
	assert.Equal(t, newEmail, i["Email"])
}

func TestAPIUserChangePassword(t *T) {
	user, _, oldPassword := testAPICreateUser(t)
	newPassword := commontest.RandStr()
	url := fmt.Sprintf("/%s/password", user)
	urlAs := fmt.Sprintf("/%s/password?_asUser=%s", user, user)
	urlAuth := fmt.Sprintf("/%s/auth?_asUser=%s", user, user)

	reqBody := fmt.Sprintf(`{"OldPassword":"aaaaaa","NewPassword":"%s"}`, newPassword)
	commontest.AssertReqErr(t, testMux, "POST", url, reqBody, ErrBadAuth)
	commontest.AssertReqErr(t, testMux, "POST", urlAs, reqBody, ErrBadAuth)

	// Ensure that the last two calls didn't actually change the password
	reqBody = fmt.Sprintf(`{"Password":"%s"}`, oldPassword)
	commontest.AssertReq(t, testMux, "POST", urlAuth, reqBody, "")

	reqBody = fmt.Sprintf(`{"OldPassword":"%s","NewPassword":"%s"}`, oldPassword, newPassword)
	commontest.AssertReq(t, testMux, "POST", urlAs, reqBody, "")

	// Ensure that the old password now doesn't work and the new one does
	reqBody = fmt.Sprintf(`{"Password":"%s"}`, oldPassword)
	commontest.AssertReqErr(t, testMux, "POST", urlAuth, reqBody, ErrBadAuth)
	reqBody = fmt.Sprintf(`{"Password":"%s"}`, newPassword)
	commontest.AssertReq(t, testMux, "POST", urlAuth, reqBody, "")
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
