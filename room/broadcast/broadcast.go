// Package broadcast implements a generic system where a user can generate
// content which can then be consumed by multiple other users. This package
// mostly handles whether or not a user is broadcasting, and what id they are
// broadcasting to
//
// - A user can only have a single broadcast at a time
//
// - It must be periodically verified that a user is still broadcasting
//
// - A signature is given when starting a broadcast which can optionally be
//   later used to authenticate a broadcast ID
//
package broadcast

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/mediocregopher/mediocre-api/common"
	"github.com/mediocregopher/mediocre-api/room"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/mediocregopher/radix.v2/util"
)

// Errors which can be expected from various methods in this package
var (
	ErrUserIsBroadcasting = common.ExpectedErr{400, "user already broadcasting"}
)

// Errors which should not happen unless there is a bug somewhere
var (
	errInvalidID      = errors.New("invalid broadcast.ID")
	errBroadcastEnded = errors.New("broadcast already ended")
)

// EXPIREEQUAL KEY SECONDS VALUE
// Sets the given key's expire time to the given seconds, but only if the key's
// current value is equal to VALUE. Returns 1 if set, 0 if not
var expireEqual = `
	local v = redis.call('GET', KEYS[1])
	if v == ARGV[2] then
		redis.call('EXPIRE', KEYS[1], ARGV[1])
		return 1
	else
		return 0
	end
`

// DELEQUAL KEY VALUE
// Deletes the given key, but only if the key's current value is equal to VALUE.
// Returns 1 if the key was deleted, 0 otherwise
var delEqual = `
	local v = redis.call('GET', KEYS[1])
	if v == ARGV[1] then
		return redis.call('DEL', KEYS[1])
	else
		return 0
	end
`

// System holds on to a room.System and implements a broadcast system around it,
// using the room.System to track what users are in what broadcasts
type System struct {
	c util.Cmder
	*room.System
	secret []byte

	// Prefix can be filled in on a System returned from New, and is used as
	// part of a prefix on all keys used by this system. Useful if you want to
	// have two broadcast Systems using the same Cmder
	Prefix string

	// This is the amount of seconds which is allowed to elapse with no
	// StillBroadcasting calls for a broadcast before it is considered dead.
	// Defaults to 30
	AlivenessPeriod int
}

// New returns a new initialized system. The secret will be used to sign
// broadcast ids for users actually creating a broadcast.
func New(c util.Cmder, secret []byte) *System {
	return &System{
		c:               c,
		secret:          secret,
		AlivenessPeriod: 30,
	}
}

// ID represents the unique identifier for a broadcast. IDs have certain data
// embedded in them, and methods for retrieving that data
type ID string

// User returns the name of the user encoded into the id
func (id ID) User() string {
	idDec, err := base64.StdEncoding.DecodeString(string(id))
	if err != nil {
		return ""
	}
	idStr := string(idDec)
	i := strings.LastIndex(idStr, ":")
	if i < 0 {
		return ""
	}
	return idStr[:i]
}

// NewID returns a new broadcast ID for the given user, along with a signature
// which can verify that the holder of the id is the true owner. This method
// makes no database changes, see StartBroadcast if that's what you're looking
// for
func (s *System) NewID(user string) (ID, string) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	benc := base64.StdEncoding.EncodeToString(b)
	id := user + ":" + benc
	id64 := base64.StdEncoding.EncodeToString([]byte(id))

	h := hmac.New(sha1.New, s.secret)
	h.Write([]byte(id))
	sig := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return ID(id64), sig
}

// Verify returns wheither or not the given sig is the valid signature for the
// given ID, i.e. they were both returned from the same call to NewID or
// StartBroadcast
func (s *System) Verify(id ID, sig string) bool {
	idDec, err := base64.StdEncoding.DecodeString(string(id))
	if err != nil {
		return false
	}
	h := hmac.New(sha1.New, s.secret)
	h.Write(idDec)
	realSig := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return realSig == sig
}

func (s *System) userKey(user string) string {
	k := "broadcast:" + s.Prefix + ":user:{" + user + "}"
	return k
}

// StartBroadcast returns a unique broadcast id for the user to use, and the
// signature for that id which can be used to verify they are the real
// broadcaster. This will error if the user is already broadcasting
func (s *System) StartBroadcast(user string) (ID, string, error) {
	id, sig := s.NewID(user)
	ukey := s.userKey(user)
	r := s.c.Cmd("SET", ukey, id, "EX", s.AlivenessPeriod, "NX")
	if r.Err != nil {
		return "", "", r.Err
	} else if r.IsType(redis.Nil) {
		return "", "", ErrUserIsBroadcasting
	}
	return id, sig, nil
}

// StillAlive records that the broadcast is still actively going. This must be
// called periodically or the user will no longer be considered broadcasting,
// see AlivenessPeriod
func (s *System) StillAlive(id ID) error {
	user := id.User()
	if user == "" {
		return errInvalidID
	}
	key := s.userKey(user)
	i, err := util.LuaEval(s.c, expireEqual, 1, key, s.AlivenessPeriod, string(id)).Int()
	if err != nil {
		return err
	}
	if i == 0 {
		return errBroadcastEnded
	}
	return nil
}

// End records that a broadcast has ended and that the user is no longer
// broadcasting
func (s *System) EndBroadcast(id ID) error {
	user := id.User()
	if user == "" {
		return errInvalidID
	}
	key := s.userKey(user)

	i, err := util.LuaEval(s.c, delEqual, 1, key, string(id)).Int()
	if err != nil {
		return err
	}
	if i == 0 {
		return errBroadcastEnded
	}
	return nil
}

// GetBroadcastID returns the currently active broadcast id for the user, or
// empty string if they are not broadcasting. An error is only returned in the
// case of a database error
func (s *System) GetBroadcastID(user string) (ID, error) {
	key := s.userKey(user)
	r := s.c.Cmd("GET", key)
	if r.IsType(redis.Nil) {
		return "", nil
	}
	idStr, err := r.Str()
	if err != nil {
		return "", err
	}
	id := ID(idStr)
	if id.User() != user {
		// This isn't expected to happen, but I'd like to enforce that any ID
		// returned from this package is a valid one
		return "", errInvalidID
	}
	return id, nil
}
