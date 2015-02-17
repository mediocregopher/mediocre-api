// Creates tokens which authenticate and identify users in a stateless way
package usertok

import (
	"crypto/rand"
	"encoding/json"

	"github.com/mediocregopher/mediocre-api/auth/sig"
)

type userTokData struct {
	User   string
	Random []byte
}

// Given a user identifying string and a secret, returns a user token
func New(user string, secret []byte) string {
	shared := make([]byte, 128)
	if _, err := rand.Read(shared); err != nil {
		panic(err) // should probably do something else here....
	}

	u := userTokData{user, shared}
	userTokD, _ := json.Marshal(u)
	return sig.New(userTokD, secret)
}

// Given a userTok as returned by New(), extracts the user identifier that was
// passed into New() and returns it. Returns empty string if the user token
// can't be extracted due to an invalid token
func ExtractUser(userTok string, secret []byte) string {
	userTokD := sig.Extract(userTok, secret)
	if userTokD == nil {
		return ""
	}

	var u userTokData
	if err := json.Unmarshal(userTokD, &u); err != nil {
		return ""
	}
	return u.User
}
