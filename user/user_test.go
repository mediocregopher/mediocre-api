package user

import (
	"crypto/rand"
	"encoding/hex"
	. "testing"

	"github.com/mediocregopher/radix.v2/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSystem(t *T) *System {
	c, err := redis.Dial("tcp", "localhost:6379")
	require.Nil(t, err)

	return New(c)
}

func randStr() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func TestCreate(t *T) {
	s := testSystem(t)

	user, email, password := randStr(), randStr(), randStr()
	assert.Nil(t, s.Create(user, email, password))
	assert.Equal(t, ErrUserExists, s.Create(user, email, password))

	// TODO make sure that the user is retrieved correctly at this point
}
