// A simple package for creating and verifying signed arbitrary data
package sig

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"strings"
)

// Returns a string which is a combination of the given data and a signature of
// the given data, signed by the given secret
func New(data, secret []byte) string {
	sum64 := NewSigOnly(data, secret)
	data64 := base64.StdEncoding.EncodeToString(data)
	return data64 + ":" + sum64
}

// Returns the signature of the data, without the data itself prepended on like
// New() does. This cannot be passed into Extract() or Verify()
func NewSigOnly(data, secret []byte) string {
	h := hmac.New(sha1.New, secret)
	h.Write(data)
	sum := h.Sum(nil)
	return base64.StdEncoding.EncodeToString(sum)
}

// Extracts the encoded, signed data in the given sig. Returns nil if the data
// can't be decoded or verified
func Extract(sig string, secret []byte) []byte {
	i := strings.IndexByte(sig, ':')
	if i < 0 {
		return nil
	}

	data64 := sig[:i]
	data, err := base64.StdEncoding.DecodeString(data64)
	if err != nil {
		return nil
	}

	sum64 := sig[i+1:]
	sum, err := base64.StdEncoding.DecodeString(sum64)
	if err != nil {
		return nil
	}

	h := hmac.New(sha1.New, secret)
	h.Write(data)
	if !hmac.Equal(h.Sum(nil), sum) {
		return nil
	}

	return data
}

// Shortcut for Extract(sig, secret) != nil
func Verify(sig string, secret []byte) bool {
	return Extract(sig, secret) != nil
}

// Returns whether or not the given signature (returned from NewSigOnly, or
// base64 encoded by something else) was used to sign data using secret
func VerifySigOnly(sig string, data, secret []byte) bool {
	h := hmac.New(sha1.New, secret)
	h.Write(data)
	sum := h.Sum(nil)

	sigD, _ := base64.StdEncoding.DecodeString(sig)
	return hmac.Equal(sum, sigD)
}
