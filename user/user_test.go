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

	i, err := s.Get(user, Public)
	require.Nil(t, err)
	assert.Equal(t, user, i["Name"])
	assert.Equal(t, "", i["Email"])
	tsCreated, err := unmarshalTime(i["TSCreated"])
	require.Nil(t, err)
	assert.True(t, tsCreated.After(start) && tsCreated.Before(end))

	pi, err := s.Get(user, Private)
	require.Nil(t, err)
	assert.Equal(t, user, pi["Name"])
	tsCreated, err = unmarshalTime(pi["TSCreated"])
	require.Nil(t, err)
	assert.True(t, tsCreated.After(start) && tsCreated.Before(end))
	// now the parts not inherited from Info
	assert.Equal(t, email, pi["Email"])
	tsModified, err := unmarshalTime(pi["TSModified"])
	require.Nil(t, err)
	assert.True(t, tsModified.After(start) && tsModified.Before(end))

	// Try to create a banned username
	err = s.Create("root", email, password)
	assert.Equal(t, ErrInvalidUsername, err)
}

func TestGetNonExistant(t *T) {
	s := testSystem(t)
	user := commontest.RandStr()

	i, err := s.Get(user, Public)
	assert.Equal(t, ErrNotFound, err)
	assert.Nil(t, i)
}

func TestInternalSet(t *T) {
	s := testSystem(t)
	s.AddField(Field{Name: "foo", Flags: Public})
	s.AddField(Field{Name: "baz", Flags: Public})
	user := commontest.RandStr()

	start := time.Now()
	err := s.set(user, "foo", "bar", "baz", "buz")
	require.Nil(t, err)
	end := time.Now()

	tsModifiedKey := s.fields["TSModified"].Key
	r := s.c.Cmd("HMGET", s.Key(user), tsModifiedKey, "foo", "baz", "box")
	l, err := r.List()
	require.Nil(t, err)

	tsm, err := unmarshalTime(l[0])
	require.Nil(t, err)
	assert.True(t, tsm.After(start) && tsm.Before(end))

	assert.Equal(t, "bar", l[1])
	assert.Equal(t, "buz", l[2])
	assert.Equal(t, "", l[3])
}

func TestInternalSetExists(t *T) {
	s := testSystem(t)
	s.AddField(Field{Name: "foo", Flags: Public})
	s.AddField(Field{Name: "baz", Flags: Public})
	userDNE := commontest.RandStr()

	err := s.setExists(userDNE, "foo", "bar", "baz", "buz")
	assert.Equal(t, ErrNotFound, err)

	user, _, _ := randUser(t, s)
	start := time.Now()
	err = s.setExists(user, "foo", "bar", "baz", "buz")
	require.Nil(t, err)
	end := time.Now()

	tsModifiedKey := s.fields["TSModified"].Key
	r := s.c.Cmd("HMGET", s.Key(user), tsModifiedKey, "foo", "baz", "box")
	l, err := r.List()
	require.Nil(t, err)

	tsm, err := unmarshalTime(l[0])
	require.Nil(t, err)
	assert.True(t, tsm.After(start) && tsm.Before(end))

	assert.Equal(t, "bar", l[1])
	assert.Equal(t, "buz", l[2])
	assert.Equal(t, "", l[3])

	// Make sure that disabling the user prevents setExists from working
	require.Nil(t, s.Disable(user))
	err = s.setExists(user, "foo", "bar1", "baz", "buz1")
	assert.Equal(t, ErrDisabled, err)
	l, err = s.c.Cmd("HMGET", s.Key(user), "foo", "baz", "box").List()
	require.Nil(t, err)
	assert.Equal(t, "bar", l[0])
	assert.Equal(t, "buz", l[1])
	assert.Equal(t, "", l[2])

	// Make sure that re-enabling the user allows setExists to work again
	require.Nil(t, s.Enable(user))
	err = s.setExists(user, "foo", "bar1", "baz", "buz1")
	require.Nil(t, err)
	l, err = s.c.Cmd("HMGET", s.Key(user), "foo", "baz", "box").List()
	require.Nil(t, err)
	assert.Equal(t, "bar1", l[0])
	assert.Equal(t, "buz1", l[1])
	assert.Equal(t, "", l[2])
}

func TestAuthenticate(t *T) {
	s := testSystem(t)
	user, _, password := randUser(t, s)

	err := s.Authenticate(user, password)
	require.Nil(t, err)

	err = s.Authenticate(user, password+"bogus")
	assert.Equal(t, ErrBadAuth, err)

	err = s.Authenticate(user+"bogus", password)
	assert.Equal(t, ErrNotFound, err)
}

func TestChangePassword(t *T) {
	s := testSystem(t)
	user, _, password := randUser(t, s)

	newPassword := commontest.RandStr()
	err := s.ChangePassword(user, newPassword)
	require.Nil(t, err)

	err = s.Authenticate(user, password)
	assert.Equal(t, ErrBadAuth, err)

	err = s.Authenticate(user, newPassword)
	assert.Nil(t, err)
}

func TestDisable(t *T) {
	s := testSystem(t)
	user, _, password := randUser(t, s)

	require.Nil(t, s.Disable(user))

	pi, err := s.Get(user, Private)
	require.Nil(t, err)
	assert.NotEqual(t, "", pi["Disabled"])

	err = s.Authenticate(user, password)
	assert.Equal(t, ErrDisabled, err)

	require.Nil(t, s.Enable(user))

	pi, err = s.Get(user, Private)
	require.Nil(t, err)
	assert.Equal(t, "", pi["Disabled"])

	err = s.Authenticate(user, password)
	assert.Nil(t, err)
}

func TestSet(t *T) {
	s := testSystem(t)
	s.AddField(Field{Name: "foo", Flags: Public})
	s.AddField(Field{Name: "bar", Flags: Public | Editable})

	userDNE := commontest.RandStr()
	err := s.Set(userDNE, Info{"foo": "foo", "bar": "bar"})
	assert.Equal(t, ErrFieldUneditable("foo"), err)

	err = s.Set(userDNE, Info{"bar": "bar"})
	assert.Equal(t, ErrNotFound, err)

	user, _, _ := randUser(t, s)
	err = s.Set(user, Info{"bar": "bar"})
	require.Nil(t, err)
	u, err := s.Get(user, Public)
	require.Nil(t, err)
	assert.Equal(t, "bar", u["bar"])

	err = s.Set(user, Info{"bar": "bar1"})
	require.Nil(t, err)
	u, err = s.Get(user, Public)
	require.Nil(t, err)
	assert.Equal(t, "bar1", u["bar"])
}
