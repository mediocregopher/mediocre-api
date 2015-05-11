package main

import (
	"fmt"
	"net/http"
	. "testing"

	"github.com/mediocregopher/mediocre-api/common"
	"github.com/mediocregopher/mediocre-api/common/commontest"
	"github.com/mediocregopher/mediocre-api/pickyjson"
	"github.com/mediocregopher/mediocre-api/user"
	"github.com/stretchr/testify/assert"
)

var testMux = func() http.Handler {
	cmder := commontest.APIStarterKit()
	return UserMux(cmder)
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
	u := commontest.RandStr()
	email := commontest.RandEmail()
	password := commontest.RandStr()

	reqBody := fmt.Sprintf(`{"Username":"%s","Password":"%s"}`, u, password)

	// Sanity check to make sure required parameters are being checked
	// correctly, we don't need to do this for all tests though
	expectedErr := pickyjson.ErrFieldRequiredf("Email").(common.ExpectedErr)
	commontest.AssertReqErr(t, testMux, "POST", "/new-user", reqBody, expectedErr)

	reqBody = fmt.Sprintf(
		`{"Email":"%s","Username":"%s","Password":"%s"}`,
		email,
		u,
		password,
	)
	commontest.AssertReq(t, testMux, "POST", "/new-user", reqBody, "")
	commontest.AssertReqErr(t, testMux, "POST", "/new-user", reqBody, user.ErrUserExists)
}

func TestAPIUserGet(t *T) {
	u, email, _ := testAPICreateUser(t)
	url := fmt.Sprintf("/%s", u)
	var i user.Info

	commontest.AssertReqJSON(t, testMux, "GET", url, "", &i)
	assert.Equal(t, u, i["Name"])
	assert.Equal(t, "", i["Email"])

	commontest.AssertReqJSON(t, testMux, "GET", url+"?_asUser="+u, "", &i)
	assert.Equal(t, u, i["Name"])
	assert.Equal(t, email, i["Email"])

	u404 := commontest.RandStr()
	url = fmt.Sprintf("/%s", u404)
	commontest.AssertReqErr(t, testMux, "GET", url, "", user.ErrNotFound)
}

func TestAPIUserSet(t *T) {
	u, email, _ := testAPICreateUser(t)
	url := fmt.Sprintf("/%s", u)
	urlAs := fmt.Sprintf("/%s?_asUser=%s", u, u)
	newEmail := "foo_" + email

	reqBody := fmt.Sprintf(`{"Email":"%s"}`, newEmail)
	commontest.AssertReqErr(t, testMux, "POST", url, reqBody, user.ErrBadAuth)
	commontest.AssertReq(t, testMux, "POST", urlAs, reqBody, "")

	var i user.Info
	commontest.AssertReqJSON(t, testMux, "GET", urlAs, "", &i)
	assert.Equal(t, u, i["Name"])
	assert.Equal(t, newEmail, i["Email"])
}

func TestAPIUserChangePassword(t *T) {
	u, _, oldPassword := testAPICreateUser(t)
	newPassword := commontest.RandStr()
	url := fmt.Sprintf("/%s/password", u)
	urlAs := fmt.Sprintf("/%s/password?_asUser=%s", u, u)
	urlAuth := fmt.Sprintf("/%s/auth?_asUser=%s", u, u)

	reqBody := fmt.Sprintf(`{"OldPassword":"aaaaaa","NewPassword":"%s"}`, newPassword)
	commontest.AssertReqErr(t, testMux, "POST", url, reqBody, user.ErrBadAuth)
	commontest.AssertReqErr(t, testMux, "POST", urlAs, reqBody, user.ErrBadAuth)

	// Ensure that the last two calls didn't actually change the password
	reqBody = fmt.Sprintf(`{"Password":"%s"}`, oldPassword)
	commontest.AssertReq(t, testMux, "POST", urlAuth, reqBody, "")

	reqBody = fmt.Sprintf(`{"OldPassword":"%s","NewPassword":"%s"}`, oldPassword, newPassword)
	commontest.AssertReq(t, testMux, "POST", urlAs, reqBody, "")

	// Ensure that the old password now doesn't work and the new one does
	reqBody = fmt.Sprintf(`{"Password":"%s"}`, oldPassword)
	commontest.AssertReqErr(t, testMux, "POST", urlAuth, reqBody, user.ErrBadAuth)
	reqBody = fmt.Sprintf(`{"Password":"%s"}`, newPassword)
	commontest.AssertReq(t, testMux, "POST", urlAuth, reqBody, "")
}

// TestAPIUserAuth tests retrieving a user token from the api. Essentially,
// logging in
func TestAPIUserAuth(t *T) {
	u, _, password := testAPICreateUser(t)
	url := fmt.Sprintf("/%s/auth", u)

	reqBody := `{"Password":"aaaaaa"}`
	commontest.AssertReqErr(t, testMux, "POST", url, reqBody, user.ErrBadAuth)

	reqBody = fmt.Sprintf(`{"Password":"%s"}`, password)
	commontest.AssertReq(t, testMux, "POST", url, reqBody, "")
}
