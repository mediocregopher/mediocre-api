package usertok

import (
	"crypto/rand"
	. "testing"

	"github.com/stretchr/testify/assert"
)

// Returns 10 random slices of bytes
func randByteSlices() [][]byte {
	slices := make([][]byte, 0, 10)
	for range slices {
		b := make([]byte, 50)
		if _, err := rand.Read(b); err != nil {
			panic(err)
		}
		slices = append(slices, b)
	}
	return slices
}

func TestUserTok(t *T) {
	users := randByteSlices()
	secrets := randByteSlices()

	for _, user := range users {
		for _, secret := range secrets {
			userTok := New(string(user), secret)
			assert.Equal(t, user, ExtractUser(userTok, secret))
		}
	}
}
