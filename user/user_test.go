package user

import (
	. "testing"
	"time"

	"github.com/mediocregopher/mediocre-api/common/commontest"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSystem(t *T) *System {
	p, err := pool.New("tcp", "localhost:6379", 10)
	require.Nil(t, err)

	return New(p)
}

func randUser(t *T, s *System) (string, string, string) {
	user := commontest.RandStr()
	email := commontest.RandStr()
	password := commontest.RandStr()
	require.Nil(t, s.Create(user, email, password))
	return user, email, password
}

func TestCreateGet(t *T) {
	s := testSystem(t)
	start := time.Now()

	user, email, password := randUser(t, s)
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
	user := commontest.RandStr()

	i, err := s.Get(user)
	assert.Equal(t, ErrNotFound, err)
	assert.Nil(t, i)

	pi, err := s.GetPrivate(user)
	assert.Equal(t, ErrNotFound, err)
	assert.Nil(t, pi)
}

func TestInternalSet(t *T) {
	s := testSystem(t)
	user := commontest.RandStr()

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
	user, _, password := randUser(t, s)

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
	assert.Equal(t, ErrNotFound, err)
	assert.False(t, ok)
}

func TestVerify(t *T) {
	s := testSystem(t)
	user, _, _ := randUser(t, s)

	pi, err := s.GetPrivate(user)
	require.Nil(t, err)
	assert.False(t, pi.Verified)

	require.Nil(t, s.Verify(user))

	pi, err = s.GetPrivate(user)
	require.Nil(t, err)
	assert.True(t, pi.Verified)
}

func TestDisable(t *T) {
	s := testSystem(t)
	user, _, password := randUser(t, s)

	require.Nil(t, s.Disable(user))

	pi, err := s.GetPrivate(user)
	require.Nil(t, err)
	assert.True(t, pi.Disabled)

	ok, err := s.Login(user, password)
	assert.Equal(t, ErrDisabled, err)
	assert.False(t, ok)

	require.Nil(t, s.Enable(user))

	pi, err = s.GetPrivate(user)
	require.Nil(t, err)
	assert.False(t, pi.Disabled)

	ok, err = s.Login(user, password)
	assert.Nil(t, err)
	assert.True(t, ok)
}
