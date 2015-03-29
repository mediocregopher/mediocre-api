// Package usertok creates tokens which authenticate and identify users in a
// stateless way
package usertok

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/mediocregopher/mediocre-api/auth/sig"
)

var b64 = base64.StdEncoding

// New returns a new user token given a user identifying string and a secret
func New(user string, secret []byte) string {
	shared := make([]byte, 16)
	if _, err := rand.Read(shared); err != nil {
		panic(err) // should probably do something else here....
	}
	userB := []byte(user)

	userL := b64.EncodedLen(len(userB))
	l := userL + b64.EncodedLen(len(shared)) + 1
	data := make([]byte, l)
	b64.Encode(data, userB)
	data[userL] = ':'
	b64.Encode(data[userL+1:], shared)

	return sig.New(data, secret, 48*time.Hour)
}

// ExtractUser takes in a userTok as returned by New() and extracts the user
// identifier that was passed into New() and returns it. Returns empty string if
// the user token can't be extracted due to an invalid token
func ExtractUser(userTok string, secret []byte) string {
	data := sig.Extract(userTok, secret)
	if data == nil {
		return ""
	}

	parts := bytes.SplitN(data, []byte(":"), 2)
	if len(parts) != 2 {
		return ""
	}

	userB, err := b64.DecodeString(string(parts[0]))
	if err != nil {
		return ""
	}

	return string(userB)
}
