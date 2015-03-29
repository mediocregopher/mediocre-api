// Package sig is a package for creating and verifying signed arbitrary data
package sig

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"strings"
	"time"
)

// New returns a string which is a combination of the given data and a signature
// of the given data, signed by the given secret
func New(data, secret []byte, timeout time.Duration) string {
	expires := time.Now().Add(timeout)
	expiresB, _ := expires.MarshalBinary()

	h := hmac.New(sha1.New, secret)
	h.Write(data)
	h.Write(expiresB)
	sum := h.Sum(nil)
	sum64 := base64.StdEncoding.EncodeToString(sum)
	expires64 := base64.StdEncoding.EncodeToString(expiresB)
	data64 := base64.StdEncoding.EncodeToString(data)
	return data64 + ":" + expires64 + ":" + sum64
}

// Extract extracts the encoded, signed data in the given sig. Returns nil if
// the data can't be decoded or verified
func Extract(sig string, secret []byte) []byte {
	i := strings.IndexByte(sig, ':')
	if i < 0 {
		return nil
	}

	data64 := sig[:i]
	sig = sig[i+1:]
	data, err := base64.StdEncoding.DecodeString(data64)
	if err != nil {
		return nil
	}

	i = strings.IndexByte(sig, ':')
	if i < 0 {
		return nil
	}

	expires64 := sig[:i]
	sig = sig[i+1:]
	expiresB, err := base64.StdEncoding.DecodeString(expires64)
	if err != nil {
		return nil
	}
	var expires time.Time
	if err = expires.UnmarshalBinary(expiresB); err != nil {
		return nil
	}
	if time.Now().After(expires) {
		return nil
	}

	sum64 := sig
	sum, err := base64.StdEncoding.DecodeString(sum64)
	if err != nil {
		return nil
	}

	h := hmac.New(sha1.New, secret)
	h.Write(data)
	h.Write(expiresB)
	if !hmac.Equal(h.Sum(nil), sum) {
		return nil
	}

	return data
}

// Verify is a shortcut for Extract(sig, secret) != nil
func Verify(sig string, secret []byte) bool {
	return Extract(sig, secret) != nil
}
