package room

import (
	"runtime/debug"
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

	return New(p, &Opts{CheckInPeriod: 1 * time.Second})
}

func assertRoomMembers(t *T, s *System, room string, members ...string) {
	l, err := s.Members(room)
	require.Nil(t, err)

	mExpect := map[string]bool{}
	for i := range members {
		mExpect[members[i]] = true
	}

	mFound := map[string]bool{}
	for i := range l {
		mFound[l[i]] = true
	}

	assert.Equal(t, mExpect, mFound, string(debug.Stack()))

	c, err := s.Cardinality(room)
	require.Nil(t, err)
	assert.Equal(t, len(mExpect), c, string(debug.Stack()))
}

func TestCheckIn(t *T) {
	s := testSystem(t)
	room1 := commontest.RandStr()
	room2 := commontest.RandStr()
	user1 := commontest.RandStr()
	user2 := commontest.RandStr()

	// Make sure rooms start off empty
	assertRoomMembers(t, s, room1)
	assertRoomMembers(t, s, room2)

	// Check some people in
	require.Nil(t, s.CheckIn(room1, user1))
	require.Nil(t, s.CheckIn(room2, user2))
	assertRoomMembers(t, s, room1, user1)
	assertRoomMembers(t, s, room2, user2)

	// Make sure re-checking them in doesn't do anything
	require.Nil(t, s.CheckIn(room1, user1))
	require.Nil(t, s.CheckIn(room2, user2))
	assertRoomMembers(t, s, room1, user1)
	assertRoomMembers(t, s, room2, user2)

	// Check them into more than one room
	require.Nil(t, s.CheckIn(room1, user2))
	assertRoomMembers(t, s, room1, user1, user2)
	assertRoomMembers(t, s, room2, user2)
	require.Nil(t, s.CheckIn(room2, user1))
	assertRoomMembers(t, s, room1, user1, user2)
	assertRoomMembers(t, s, room2, user1, user2)
}

func TestLeave(t *T) {
	s := testSystem(t)
	room1 := commontest.RandStr()
	room2 := commontest.RandStr()
	room3 := commontest.RandStr()
	user1 := commontest.RandStr()
	user2 := commontest.RandStr()
	user3 := commontest.RandStr()

	// Check some people in
	require.Nil(t, s.CheckIn(room1, user1))
	require.Nil(t, s.CheckIn(room2, user2))
	require.Nil(t, s.CheckIn(room3, user3))
	assertRoomMembers(t, s, room1, user1)
	assertRoomMembers(t, s, room2, user2)
	assertRoomMembers(t, s, room3, user3)

	// Wait a half second, check one out,check one back in, the last does
	// nothing
	time.Sleep(500 * time.Millisecond)
	require.Nil(t, s.CheckOut(room1, user1))
	require.Nil(t, s.CheckIn(room2, user2))
	assertRoomMembers(t, s, room1)
	assertRoomMembers(t, s, room2, user2)
	assertRoomMembers(t, s, room3, user3)

	// Wait another half second, make sure automatic cleanup is working
	time.Sleep(500 * time.Millisecond)
	assertRoomMembers(t, s, room1)
	assertRoomMembers(t, s, room2, user2)
	assertRoomMembers(t, s, room3)

	// Checking out of a room you're not in shouldn't have an effect
	require.Nil(t, s.CheckOut(room1, user3))
	require.Nil(t, s.CheckOut(room2, user3))
	require.Nil(t, s.CheckOut(room3, user3))
	assertRoomMembers(t, s, room1)
	assertRoomMembers(t, s, room2, user2)
	assertRoomMembers(t, s, room3)
}
