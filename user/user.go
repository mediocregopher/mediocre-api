// Package user implements an abstraction for a basic user system. Users can be
// created, verified, have properties set on them, disabled, deleted, and more.
// All data is persisted to a redis instance or cluster, and all methods are
// compeletely thread-safe. No data is sanitized by this package
package user

import (
	"strings"
	"time"

	"github.com/mediocregopher/mediocre-api/common"
	"golang.org/x/crypto/bcrypt"
)

// ExpectedErr is an implementation of the error interface which will be used to
// indicate that the error being returned is an expected one and can sent back
// to the client
type ExpectedErr string

// Error implements the error interface
func (e ExpectedErr) Error() string {
	return string(e)
}

// Errors which can be expected from various methods in this package
var (
	ErrUserExists = ExpectedErr("user exists")
	ErrNotFound   = ExpectedErr("user not found")
	ErrBadAuth    = ExpectedErr("could not authenticate user")
	ErrDisabled   = ExpectedErr("user account is disabled")
)

// Fields found in the main user hash
const (
	emailField          = "e"
	passwordHashField   = "p"
	verifiedField       = "v"
	tsCreatedField      = "t"
	tsLastLoggedInField = "tl"
	tsModifiedField     = "tm"
	disabledField       = "d"
)

// System holds on to a Cmder and uses it to implement a basic user system
type System struct {
	c common.Cmder

	// The cost parameter to use when creating new password hashes. This
	// defaults to 11 and can be set right after instantiation
	BCryptCost int
}

// New returns a new System which will use the given Cmder as its persistence
// layer
func New(c common.Cmder) *System {
	return &System{
		c:          c,
		BCryptCost: 11,
	}
}

// Key returns a key which can be used to interact with some arbitrary user data
// directly in redis. This is useful if more complicated, lower level operations
// are needed to be done
func (s *System) Key(user string, extra ...string) string {
	k := "user:{" + user + "}"
	if len(extra) > 0 {
		k += ":" + strings.Join(extra, ":")
	}
	return k
}

func marshalTime(t time.Time) string {
	ts, _ := t.UTC().MarshalText()
	return string(ts)
}

func unmarshalTime(ts string) (time.Time, error) {
	var t time.Time
	if ts == "" {
		return t, nil
	}
	err := t.UnmarshalText([]byte(ts))
	return t.UTC(), err
}

// Create attempts to create a new user with the given email and password. If
// the user already exists ErrUserExists will be returned. If not the password
// will be hashed and stored
func (s *System) Create(user, email, password string) error {
	key := s.Key(user)
	i, err := s.c.Cmd("HSETNX", key, emailField, email).Int()
	if err != nil {
		return err
	} else if i == 0 {
		return ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.BCryptCost)
	if err != nil {
		return err
	}

	nowS := marshalTime(time.Now())
	err = s.c.Cmd(
		"HMSET", key,
		passwordHashField, hash,
		tsCreatedField, nowS,
		tsModifiedField, nowS,
	).Err
	if err != nil {
		return err
	}

	return nil
}

func (s *System) set(user string, keyvals ...interface{}) error {
	key := s.Key(user)
	nowS := marshalTime(time.Now())
	return s.c.Cmd("HMSET", key, tsModifiedField, nowS, keyvals).Err
}

// Login attempts to authenticate the user with the given password. If the
// password matches the one in the db then tsLastLoggedIn is updated on the user
// hash. Returns whether or not the user successfully logged in
//
// If this method returns true it may still return an error if updating
// lastLoggedIn failed. In reality only errors accompanied by a false really
// matter
func (s *System) Login(user, password string) (bool, error) {
	key := s.Key(user)
	l, err := s.c.Cmd("HMGET", key, passwordHashField, disabledField).List()
	if err != nil {
		return false, err
	}
	p, disabled := l[0], l[1]

	if p == "" {
		return false, ErrNotFound
	} else if disabled == "1" {
		return false, ErrDisabled
	}

	match := bcrypt.CompareHashAndPassword([]byte(p), []byte(password)) == nil
	if match {
		return true, s.set(user, tsLastLoggedInField, marshalTime(time.Now()))
	}
	return false, ErrBadAuth
}

// Get returns the Info for the given user, or nil if the user couldn't be found
func (s *System) Get(user string) (*Info, error) {
	key := s.Key(user)
	return respToInfo(user, s.c.Cmd("HGETALL", key))
}

// GetPrivate returns the PrivateInfo for the given user, or nil if the user
// couldn't be found
func (s *System) GetPrivate(user string) (*PrivateInfo, error) {
	key := s.Key(user)
	return respToPrivateInfo(user, s.c.Cmd("HGETALL", key))
}

// Verify marks the user as having been "verified", either through email or
// other means
func (s *System) Verify(user string) error {
	return s.set(user, verifiedField, "1")
}

// Disable marks the user as being disabled, meaning they have effectively
// deleted their account without actually deleting any data. They cannot log in
// and do not show up anywhere
func (s *System) Disable(user string) error {
	return s.set(user, disabledField, "1")
}

// Enable marks the user as having an account enabled. Accounts are enabled by
// default when created, this only really has an effect when an accound was
// previously Disable'd
func (s *System) Enable(user string) error {
	return s.set(user, disabledField, "0")
}
