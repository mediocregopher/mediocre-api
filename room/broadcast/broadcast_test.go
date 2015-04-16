package broadcast

import (
	. "testing"
	"time"

	"github.com/mediocregopher/mediocre-api/common/commontest"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/mediocregopher/radix.v2/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSystem(t *T) *System {
	p, err := pool.New("tcp", "localhost:6379", 10)
	require.Nil(t, err)

	s := New(p)
	s.AlivenessPeriod = 1
	s.Secret = []byte("TURTLES")
	return s
}

func assertUserBroadcastID(t *T, s *System, user string, id ID) {
	idTest, err := s.GetBroadcastID(user)
	require.Nil(t, err)
	assert.Equal(t, id, idTest)
}

func TestID(t *T) {
	s := testSystem(t)
	for i := 0; i < 10000; i++ {
		user := commontest.RandStr()
		id, sig := s.NewID(user)
		assert.True(t, s.Verify(id, sig), "id: %s sig: %s", id, sig)
		assert.Equal(t, user, id.User())

		wrongID := ID(commontest.RandStr())
		assert.False(t, s.Verify(wrongID, sig), "wrong id: %s sig: %s", wrongID, sig)
	}
}

func TestStartBroadcast(t *T) {
	s := testSystem(t)
	user := commontest.RandStr()
	id, _, err := s.StartBroadcast(user)
	require.Nil(t, err)
	assert.True(t, len(id) > 0)

	assertUserBroadcastID(t, s, user, id)

	_, _, err = s.StartBroadcast(user)
	assert.Equal(t, ErrUserIsBroadcasting, err)
}

func TestStillAlive(t *T) {
	s := testSystem(t)
	user := commontest.RandStr()
	_, _, err := s.StartBroadcast(user)
	require.Nil(t, err)

	time.Sleep(1500 * time.Millisecond)
	assertUserBroadcastID(t, s, user, "")

	id, _, err := s.StartBroadcast(user)
	require.Nil(t, err)

	time.Sleep(500 * time.Millisecond)
	require.Nil(t, s.StillAlive(id))
	assertUserBroadcastID(t, s, user, id)

	time.Sleep(750 * time.Millisecond)
	assertUserBroadcastID(t, s, user, id)

	time.Sleep(500 * time.Millisecond)
	assertUserBroadcastID(t, s, user, "")

	assert.Equal(t, ErrBroadcastEnded, s.StillAlive(id))
}

func TestEnded(t *T) {
	s := testSystem(t)
	user := commontest.RandStr()
	id, _, err := s.StartBroadcast(user)
	require.Nil(t, err)

	assertUserBroadcastID(t, s, user, id)
	require.Nil(t, s.Ended(id))
	assertUserBroadcastID(t, s, user, "")

	assert.Equal(t, ErrBroadcastEnded, s.Ended(id))
}

func TestExpireEqual(t *T) {
	p, err := redis.Dial("tcp", "localhost:6379")
	require.Nil(t, err)

	// Set value to "key"
	key := commontest.RandStr()
	require.Nil(t, p.Cmd("SET", key, "key").Err)

	// Try to set expire with wrong value, should return 0 and not have any ttl
	i, err := util.LuaEval(p, expireEqual, 1, key, 30, "turtle").Int()
	require.Nil(t, err)
	assert.Equal(t, 0, i)

	ttl, err := p.Cmd("TTL", key).Int()
	require.Nil(t, err)
	assert.Equal(t, -1, ttl)

	// Try to set expire with correct value, should return 1 and have a ttl
	i, err = util.LuaEval(p, expireEqual, 1, key, 30, "key").Int()
	require.Nil(t, err)
	assert.Equal(t, 1, i)

	ttl, err = p.Cmd("TTL", key).Int()
	require.Nil(t, err)
	assert.True(t, ttl > 25)
}

func TestDelEqual(t *T) {
	p, err := redis.Dial("tcp", "localhost:6379")
	require.Nil(t, err)

	// Set value to "key"
	key := commontest.RandStr()
	require.Nil(t, p.Cmd("SET", key, "key").Err)

	// Try to delete with wrong value, should return 0 and the value still be
	// there
	i, err := util.LuaEval(p, delEqual, 1, key, "turtle").Int()
	require.Nil(t, err)
	assert.Equal(t, 0, i)

	s, err := p.Cmd("GET", key).Str()
	require.Nil(t, err)
	assert.Equal(t, "key", s)

	// Try to delete with correct value, should return 0 and the value not be
	// there anymore
	i, err = util.LuaEval(p, delEqual, 1, key, "key").Int()
	require.Nil(t, err)
	assert.Equal(t, 1, i)

	r := p.Cmd("GET", key)
	assert.True(t, r.IsType(redis.Nil))
}
