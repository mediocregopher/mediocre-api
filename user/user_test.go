package user

import (
	"crypto/rand"
	"encoding/hex"
	. "testing"
	"time"

	"github.com/mediocregopher/radix.v2/pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSystem(t *T) *System {
	p, err := pool.New("tcp", "localhost:6379", 10)
	require.Nil(t, err)

	return New(p)
}

func randStr() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func TestCreateGet(t *T) {
	s := testSystem(t)
	start := time.Now()

	user, email, password := randStr(), randStr(), randStr()
	assert.Nil(t, s.Create(user, email, password))
	assert.Equal(t, ErrUserExists, s.Create(user, email, password))

	end := time.Now()

	i, err := s.Get(user)
	require.Nil(t, err)
	assert.Equal(t, user, i.Name)
	assert.True(t, i.Created.After(start) && i.Created.Before(end))

	pi, err := s.GetPrivate(user)
	require.Nil(t, err)
	assert.Equal(t, user, pi.Name)
	assert.True(t, pi.Created.After(start) && pi.Created.Before(end))
	// now the parts not inherited from Info
	assert.Equal(t, email, pi.Email)
	assert.Equal(t, false, pi.Verified)
	assert.True(t, pi.Modified.After(start) && pi.Modified.Before(end))
	assert.True(t, pi.LastLoggedIn.IsZero())
}

func TestGetNonExistant(t *T) {
	s := testSystem(t)
	user := randStr()

	i, err := s.Get(user)
	require.Nil(t, err)
	assert.Nil(t, i)

	pi, err := s.GetPrivate(user)
	require.Nil(t, err)
	assert.Nil(t, pi)
}

func TestInternalSet(t *T) {
	s := testSystem(t)
	user := randStr()

	start := time.Now()
	err := s.set(user, "foo", "bar", "baz", "buz")
	require.Nil(t, err)
	end := time.Now()

	t.Log(s.Key(user))
	r := s.c.Cmd("HMGET", s.Key(user), tsModifiedField, "foo", "baz", "box")
	l, err := r.List()
	require.Nil(t, err)

	tsm, err := unmarshalTime(l[0])
	require.Nil(t, err)
	assert.True(t, tsm.After(start) && tsm.Before(end))

	assert.Equal(t, "bar", l[1])
	assert.Equal(t, "buz", l[2])
	assert.Equal(t, "", l[3])
}

func TestLogin(t *T) {
	s := testSystem(t)
	user, email, password := randStr(), randStr(), randStr()
	assert.Nil(t, s.Create(user, email, password))

	start := time.Now()
	ok, err := s.Login(user, password)
	require.Nil(t, err)
	require.True(t, ok)
	end := time.Now()

	tsls, err := s.c.Cmd("HGET", s.Key(user), tsLastLoggedInField).Str()
	require.Nil(t, err)
	tsl, err := unmarshalTime(tsls)
	require.Nil(t, err)
	assert.True(t, tsl.After(start) && tsl.Before(end))

	ok, err = s.Login(user, password+"bogus")
	assert.Equal(t, ErrBadAuth, err)
	assert.False(t, ok)

	ok, err = s.Login(user+"bogus", password)
	assert.Equal(t, ErrUserNotFound, err)
	assert.False(t, ok)
}
