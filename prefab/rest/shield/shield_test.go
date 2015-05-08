package main

import (
	. "testing"

	"github.com/mediocregopher/mediocre-api/common/commontest"
	"github.com/stretchr/testify/assert"
)

var testMux = shieldMux("turtles")

func TestBulitinAPIToken(t *T) {
	s := struct{ Token string }{}
	commontest.AssertReqJSON(t, testMux, "GET", "/shield/token", "", &s)
	assert.NotEqual(t, "", s.Token)
}
