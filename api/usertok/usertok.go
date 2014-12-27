// Implements lightweight signatures for arbitrary data. Users receive a token
// (which is public) and a secret (which is obviously private). They can then
// create a signature of arbitrary data by creating a HMAC-SHA1 of that data
// with their secret as the hmac secret, and base64 encoding the result of that
// to create the signature. Given the user token and the user created signature,
// this package can determine if the user did indeed sign that data.
package usertok

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"

	"github.com/mediocregopher/mediocre-api/api/sig"
)

type userTokData struct {
	User   string
	Random []byte
}

// Given a user identifying string and a secret, returns a user token and the
// secret for that token.
func New(user string, secret []byte) (string, string) {
	shared := make([]byte, 128)
	if _, err := rand.Read(shared); err != nil {
		panic(err) // should probably do something else here....
	}

	u := userTokData{user, shared}
	userTokD, _ := json.Marshal(u)
	userTok := base64.StdEncoding.EncodeToString(userTokD)
	userSecret := sig.NewSigOnly(userTokD, secret)

	return userTok, userSecret
}

// Given a userSecret (exactly as returned by New()) returns a userSig of the
// given data which can be passed into Verify() successfully. This is more of a
// reference than meant to actually be used, since anything generating a userSig
// would presumably be an external package or service
func UserSign(data []byte, userSecret string) string {
	h := hmac.New(sha1.New, []byte(userSecret))
	h.Write(data)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Returns whether or not it can be verified that the user with the given user
// token created userSig, which signs data. userSig must be base64 encoded.
func Verify(userTok, userSig string, data, secret []byte) bool {
	userTokD, _ := base64.StdEncoding.DecodeString(userTok)
	userSecret := []byte(sig.NewSigOnly(userTokD, secret))
	return sig.VerifySigOnly(userSig, data, userSecret)
}

// Given a userTok as returned by New(), extracts the user identifier that was
// passed into New()
func ExtractUser(userTok string) string {
	userTokD, _ := base64.StdEncoding.DecodeString(userTok)
	var u userTokData
	if err := json.Unmarshal(userTokD, &u); err != nil {
		return ""
	}
	return u.User
}
