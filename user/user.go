// Package user implements an abstraction for a basic user system. Users can be
// created, have properties set on them, disabled, deleted, and more. All data
// is persisted to a redis instance or cluster, and all methods are compeletely
// thread-safe. No data is sanitized by this package
package user

import (
	"fmt"
	"strings"
	"time"

	"github.com/mediocregopher/mediocre-api/common"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/mediocregopher/radix.v2/util"
	"golang.org/x/crypto/bcrypt"
)

// Errors which can be expected from various methods in this package
var (
	ErrUserExists = common.ExpectedErr{Code: 400, Err: "user exists"}
	ErrNotFound   = common.ExpectedErr{Code: 404, Err: "user not found"}
	ErrBadAuth    = common.ExpectedErr{Code: 400, Err: "could not authenticate user"}
	ErrDisabled   = common.ExpectedErr{Code: 400, Err: "user account is disabled"}
)

// Functions which return errors based on the related field names
var (
	ErrFieldUnknown = func(f string) error {
		return common.ExpectedErrf(400, "unknown field %q", f)
	}
	ErrFieldUneditable = func(f string) error {
		return common.ExpectedErrf(400, "field %q not editable", f)
	}
)

// HMSETXX key fieldWhichExists field value [field value...]
// Calls HMSET but only if fieldWhichExists is already set on the hash. Returns
// 1 if set was successful, 0 if it failed due to the xx condition
var hmsetxx = `
	if not redis.call('HGET', KEYS[1], ARGV[1]) then
		return 0
	end
	for i=2,#ARGV,2 do
		redis.call('HSET', KEYS[1], ARGV[i], ARGV[i+1])
	end
	return 1
`

// FieldFlag is used to indicate different behaviors for different fields, such
// as preventing them from being returned in certain circumstances, and allowing
// them to be manually edited.
type FieldFlag uint64

const (
	// Public fields will always be returned when calling Get
	Public FieldFlag = 1 << iota

	// Private fields are those that should only be shown to a verified entity,
	// and may contain private user information. Generally, only shown to the
	// logged in user
	Private

	// Hidden fields are never shown anywhere except in specific circumstances.
	Hidden

	// Editable indicates that this field is allowed to be modified manually
	Editable
)

// Field is a struct which describes a single field of a user map. A field's
// value is inherently a string.
type Field struct {

	// The name of the field. This is the key it will appear under in the user
	// map
	Name string

	// If optionally specified this will be the key the field is stored as in
	// redis (can be shorter than Name to save space)
	Key string

	// Used to determine the behavior of this field. This *must* be set to a
	// value greater than zero
	Flags FieldFlag
}

// Info represents information for a single user in the system. The fields in
// the map correspond to the fields added by AddField
type Info map[string]string

// System holds on to a Cmder and uses it to implement a basic user system. By
// default user maps have the following fields:
// * Name
// * TSCreated
// * Email (private, editable)
// * TSLastLoggedIn (private)
// * TSModified (private)
// * Disabled (private)
// * PasswordHash (hidden)
type System struct {
	c util.Cmder

	// The cost parameter to use when creating new password hashes. This
	// defaults to 11 and can be set right after instantiation
	BCryptCost int

	fields map[string]Field
}

// New returns a new System which will use the given Cmder as its persistence
// layer
func New(c util.Cmder) *System {
	s := System{
		c:          c,
		BCryptCost: 11,
		fields:     map[string]Field{},
	}
	s.AddField(Field{"Name", "_n", Public})
	s.AddField(Field{"TSCreated", "_t", Public})
	s.AddField(Field{"Email", "_e", Private | Editable})
	s.AddField(Field{"TSLastLoggedIn", "_tl", Private})
	s.AddField(Field{"TSModified", "_tm", Private})
	s.AddField(Field{"Disabled", "_d", Private})
	s.AddField(Field{"PasswordHash", "_p", Hidden})
	return &s
}

// AddField can be used just after calling New to add more fields for a single
// user map. For example, if you'd like user maps to include an image field the
// field should be added here and it will appear for appropriate  Get commands
func (s *System) AddField(f Field) {
	if f.Key == "" {
		f.Key = f.Name
	}
	for fieldName := range s.fields {
		if fieldName == f.Name || s.fields[fieldName].Key == f.Key {
			panic(fmt.Sprintf("A field %s/%s already in use", f.Name, f.Key))
		}
	}
	s.fields[f.Name] = f
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
	nowS := marshalTime(time.Now())
	tsCreatedFieldKey := s.fields["TSCreated"].Key
	i, err := s.c.Cmd("HSETNX", key, tsCreatedFieldKey, nowS).Int()
	if err != nil {
		return err
	} else if i == 0 {
		return ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.BCryptCost)
	if err != nil {
		return err
	}

	if err = s.set(user,
		"Name", user,
		"PasswordHash", hash,
		"Email", email,
	); err != nil {
		return err
	}

	return nil
}

func (s *System) set(user string, keyvals ...interface{}) error {
	args := make([]interface{}, 0, len(keyvals)+3)
	args = append(args, s.Key(user))
	args, err := s.appendKeyvalsToArgs(user, keyvals, args)
	if err != nil {
		return err
	}

	return s.c.Cmd("HMSET", args...).Err
}

// same as set, but only if the user exists. Returns ErrNotFound if user doesn't
// exist
func (s *System) setExists(user string, keyvals ...interface{}) error {
	args := make([]interface{}, 0, len(keyvals)+4)
	args = append(args, s.Key(user))
	args = append(args, s.fields["Name"].Key)
	args, err := s.appendKeyvalsToArgs(user, keyvals, args)
	if err != nil {
		return err
	}

	i, err := util.LuaEval(s.c, hmsetxx, 1, args...).Int()
	if err != nil {
		return err
	}
	if i == 0 {
		return ErrNotFound
	}
	return nil
}

// Given a set of key/value pairs, keys being field names and values being what
// they want to be set to, checks that all the fields are legitimate and adds on
// a set for the TSModified field, appending all this to the passed in args
// slice and returning the new slice
func (s *System) appendKeyvalsToArgs(
	user string, keyvals []interface{}, args []interface{},
) (
	[]interface{}, error,
) {
	tsModifiedFieldKey := s.fields["TSModified"].Key
	nowS := marshalTime(time.Now())
	args = append(args, tsModifiedFieldKey, nowS)

	for i := 0; i < len(keyvals); i += 2 {
		k := keyvals[i].(string)
		kf := s.fields[k].Key
		if kf == "" {
			return nil, ErrFieldUnknown(k)
		}
		args = append(args, kf, keyvals[i+1])
	}

	return args, nil
}

func (s *System) unset(user string, fields ...string) error {
	if len(fields) == 0 {
		return nil
	}
	keys := make([]string, len(fields))
	for i, f := range fields {
		keys[i] = s.fields[f].Key
		if keys[i] == "" {
			return ErrFieldUnknown(f)
		}
	}

	return s.c.Cmd("HDEL", s.Key(user), keys).Err
}

func (s *System) checkPassword(user, password string) error {
	u, err := s.Get(user, Hidden|Private)
	if err != nil {
		return err
	}

	if u["Disabled"] != "" {
		return ErrDisabled
	}

	p := u["PasswordHash"]
	match := bcrypt.CompareHashAndPassword([]byte(p), []byte(password)) == nil
	if !match {
		return ErrBadAuth
	}
	return nil
}

// Login attempts to authenticate the user with the given password. If the
// password matches the one in the db then tsLastLoggedIn is updated on the user
// hash. Returns whether or not the user successfully logged in
//
// If this method returns true it may still return an error if updating
// lastLoggedIn failed. In reality only errors accompanied by a false really
// matter
func (s *System) Login(user, password string) (bool, error) {
	err := s.checkPassword(user, password)
	if err != nil {
		return false, err
	}
	return true, s.set(user, "TSLastLoggedIn", marshalTime(time.Now()))
}

func (s *System) getFromResp(r *redis.Resp, filters FieldFlag) (Info, error) {
	m, err := r.Map()
	if err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return nil, ErrNotFound
	}

	rm := Info{}
	for f := range s.fields {
		filt := s.fields[f].Flags
		if filt != Public && filt&filters == 0 {
			continue
		}
		rm[f] = m[s.fields[f].Key]
	}
	return rm, nil
}

// Get returns the Info for the given user, or ErrNotFound if the user couldn't
// be found
func (s *System) Get(user string, filters FieldFlag) (Info, error) {
	key := s.Key(user)
	return s.getFromResp(s.c.Cmd("HGETALL", key), filters)
}

// Disable marks the user as being disabled, meaning they have effectively
// deleted their account without actually deleting any data. They cannot log in
// and do not show up anywhere
func (s *System) Disable(user string) error {
	return s.set(user, "Disabled", "1")
}

// Enable marks the user as having an account enabled. Accounts are enabled by
// default when created, this only really has an effect when an accound was
// previously Disable'd
func (s *System) Enable(user string) error {
	return s.unset(user, "Disabled")
}

// Set is used to manually modify a user's fields. The Info argument need only
// be filled with the fields which are desired to be changed. All fields given
// in that argument must be Editable.
func (s *System) Set(user string, i Info) error {
	keyvals := make([]interface{}, 0, len(i)*2)
	for fieldName, value := range i {
		flags := s.fields[fieldName].Flags

		if flags == 0 {
			return ErrFieldUnknown(fieldName)

		} else if flags&Editable == 0 {
			return ErrFieldUneditable(fieldName)
		}

		keyvals = append(keyvals, fieldName, value)
	}

	return s.setExists(user, keyvals...)
}
