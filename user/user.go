// Package user implements an abstraction for a basic user system. Users can be
// created, verified, have properties set on them, disabled, deleted, and more.
// All data is persisted to a redis instance or cluster, and all methods are
// compeletely thread-safe. No data is sanitized by this package
package user

import (
	"strings"

	"github.com/mediocregopher/radix.v2/redis"
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

var (
	ErrUserExists = ExpectedErr("user already exists")
)

const (
	emailField        = "e"
	passwordHashField = "p"
	verifiedField     = "v"
)

// Cmder is an interface which is implemented by both the standard radix client,
// the its client pool, and its cluster client, and is used in order to interact
// with either in a transparent way
type Cmder interface {
	Cmd(string, ...interface{}) *redis.Resp
}

// System holds on to a Cmder and uses it to implement a basic user system
type System struct {
	c Cmder

	// The cost parameter to use when creating new password hashes. This
	// defaults to 11 and can be set right after instantiation
	BCryptCost int
}

// New returns a new System which will use the given Cmder as its persistence
// layer
func New(c Cmder) *System {
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

	err = s.c.Cmd("HMSET", key, passwordHashField, hash).Err
	if err != nil {
		return err
	}

	return nil
}
