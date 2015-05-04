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
	assert.Equal(t, "", pi["Verified"])
	tsModified, err := unmarshalTime(pi["TSModified"])
	require.Nil(t, err)
	assert.True(t, tsModified.After(start) && tsModified.Before(end))
	tsLastLoggedIn, err := unmarshalTime(pi["TSLastLoggedIn"])
	require.Nil(t, err)
	assert.True(t, tsLastLoggedIn.IsZero())
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
	err = s.set(user, "foo", "bar", "baz", "buz")
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

func TestLogin(t *T) {
	s := testSystem(t)
	user, _, password := randUser(t, s)

	start := time.Now()
	ok, err := s.Login(user, password)
	require.Nil(t, err)
	require.True(t, ok)
	end := time.Now()

	tsLastLoggedInFieldKey := s.fields["TSLastLoggedIn"].Key
	tsls, err := s.c.Cmd("HGET", s.Key(user), tsLastLoggedInFieldKey).Str()
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

	pi, err := s.Get(user, Private)
	require.Nil(t, err)
	assert.Equal(t, "", pi["Verified"])

	require.Nil(t, s.Verify(user))

	pi, err = s.Get(user, Private)
	require.Nil(t, err)
	assert.NotEqual(t, "", pi["Verified"])
}

func TestDisable(t *T) {
	s := testSystem(t)
	user, _, password := randUser(t, s)

	require.Nil(t, s.Disable(user))

	pi, err := s.Get(user, Private)
	require.Nil(t, err)
	assert.NotEqual(t, "", pi["Disabled"])

	ok, err := s.Login(user, password)
	assert.Equal(t, ErrDisabled, err)
	assert.False(t, ok)

	require.Nil(t, s.Enable(user))

	pi, err = s.Get(user, Private)
	require.Nil(t, err)
	assert.Equal(t, "", pi["Disabled"])

	ok, err = s.Login(user, password)
	assert.Nil(t, err)
	assert.True(t, ok)
}

func TestSet(t *T) {
	s := testSystem(t)
	s.AddField(Field{Name: "foo", Flags: Public})
	s.AddField(Field{Name: "bar", Flags: Public | Editable})
	s.AddField(Field{Name: "baz", Flags: Public | EditableWithPassword})

	userDNE := commontest.RandStr()
	err := s.Set(userDNE, "", false, Info{"foo": "foo", "bar": "bar"})
	assert.Equal(t, ErrFieldUneditable("foo"), err)

	err = s.Set(userDNE, "", false, Info{"bar": "bar"})
	assert.Equal(t, ErrNotFound, err)

	err = s.Set(userDNE, "", false, Info{"bar": "bar", "baz": "baz"})
	assert.Equal(t, ErrFieldRequiresPassword("baz"), err)

	err = s.Set(userDNE, "", true, Info{"bar": "bar", "baz": "baz"})
	assert.Equal(t, ErrNotFound, err)

	user, _, password := randUser(t, s)
	err = s.Set(user, "", false, Info{"bar": "bar"})
	require.Nil(t, err)
	u, err := s.Get(user, Public)
	require.Nil(t, err)
	assert.Equal(t, "bar", u["bar"])

	err = s.Set(user, "WRONG", false, Info{"bar": "bar1", "baz": "baz1"})
	assert.Equal(t, ErrBadAuth, err)

	err = s.Set(user, password, false, Info{"bar": "bar1", "baz": "baz1"})
	require.Nil(t, err)
	u, err = s.Get(user, Public)
	require.Nil(t, err)
	assert.Equal(t, "bar1", u["bar"])
	assert.Equal(t, "baz1", u["baz"])

	err = s.Set(user, "WRONG", true, Info{"bar": "bar2", "baz": "baz2"})
	require.Nil(t, err)
	u, err = s.Get(user, Public)
	require.Nil(t, err)
	assert.Equal(t, "bar2", u["bar"])
	assert.Equal(t, "baz2", u["baz"])
}
