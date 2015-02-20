package user

import (
	"time"

	"github.com/mediocregopher/radix.v2/redis"
)

// Info describes a user's basic, publicly accessible information stored in the
// user hash
type Info struct {
	Name    string
	Created time.Time
}

// PrivateInfo includes information found in Info as well as other data from the
// user hash that should only be shown to the user themselves
type PrivateInfo struct {
	Info
	Email        string
	Verified     bool
	LastLoggedIn time.Time
	Modified     time.Time
}

func mapToInfo(user string, m map[string]string) (*Info, error) {
	if len(m) == 0 {
		return nil, nil
	}

	var i Info
	var err error
	i.Name = user
	if i.Created, err = unmarshalTime(m[tsCreatedField]); err != nil {
		return nil, err
	}
	return &i, nil
}

func respToInfo(user string, r *redis.Resp) (*Info, error) {
	m, err := r.Map()
	if err != nil {
		return nil, err
	}
	return mapToInfo(user, m)
}

func respToPrivateInfo(user string, r *redis.Resp) (*PrivateInfo, error) {
	m, err := r.Map()
	if err != nil {
		return nil, err
	}
	i, err := mapToInfo(user, m)
	if err != nil {
		return nil, err
	}
	if i == nil {
		return nil, nil
	}

	var pi PrivateInfo
	pi.Info = *i
	pi.Email = m[emailField]
	pi.Verified = m[verifiedField] != ""
	if pi.LastLoggedIn, err = unmarshalTime(m[tsLastLoggedInField]); err != nil {
		return nil, err
	}
	if pi.Modified, err = unmarshalTime(m[tsModifiedField]); err != nil {
		return nil, err
	}
	return &pi, nil
}