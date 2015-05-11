package main

import (
	"fmt"
	"net/http/httptest"
	. "testing"

	"github.com/mediocregopher/mediocre-api/common/commontest"
	userPrefab "github.com/mediocregopher/mediocre-api/prefab/rest/user"
	"github.com/mediocregopher/mediocre-api/user"
	"github.com/stretchr/testify/assert"
)

func TestAPIToken(t *T) {
	testMux := newShieldMux("turtles", "")
	s := struct{ Token string }{}
	commontest.AssertReqJSON(t, testMux, "GET", "/shield/token", "", &s)
	assert.NotEqual(t, "", s.Token)
}

func TestUser(t *T) {
	cmder := commontest.APIStarterKit()
	userMux := userPrefab.UserMux(cmder)
	userServer := httptest.NewServer(userMux)
	testMux := newShieldMux("apples", userServer.URL)
	u := commontest.RandStr()
	email := commontest.RandEmail()
	password := commontest.RandStr()

	// Create a user
	reqBody := fmt.Sprintf(
		`{"Email":"%s","Username":"%s","Password":"%s"}`,
		email,
		u,
		password,
	)
	r := testMux.a.NewRequest("POST", "/user/new-user", reqBody, "")
	commontest.AssertReqRaw(t, testMux, r, "")

	// Try authentiating as that user
	reqBody = fmt.Sprintf(`{"Password":"%s"}`, password)
	url := fmt.Sprintf("/user/%s/auth", u)
	r = testMux.a.NewRequest("POST", url, reqBody, "")
	s := struct{ Token string }{}
	commontest.AssertReqRawJSON(t, testMux, r, &s)
	assert.NotEqual(t, "", s.Token)

	// Retrieve that user, not logged in
	url = fmt.Sprintf("/user/%s", u)
	r = testMux.a.NewRequest("GET", url, "", "")
	info := user.Info{}
	commontest.AssertReqRawJSON(t, testMux, r, &info)
	assert.Equal(t, u, info["Name"])
	assert.Equal(t, "", info["Email"])

	// Retrieve that user, logged in this time
	url = fmt.Sprintf("/user/%s", u)
	r = testMux.a.NewRequest("GET", url, "", u)
	info = user.Info{}
	commontest.AssertReqRawJSON(t, testMux, r, &info)
	assert.Equal(t, u, info["Name"])
	assert.Equal(t, email, info["Email"])
}
