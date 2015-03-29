package sig

import (
	"crypto/rand"
	. "testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Returns 10 random slices of bytes
func randByteSlices() [][]byte {
	slices := make([][]byte, 10)
	for i := range slices {
		b := make([]byte, 50)
		if _, err := rand.Read(b); err != nil {
			panic(err)
		}
		slices[i] = b
	}
	return slices
}

func TestSig(t *T) {
	datas := randByteSlices()
	secrets := randByteSlices()
	for i := range datas {
		for j := range secrets {
			sig := New(datas[i], secrets[j], 5*time.Millisecond)
			data := Extract(sig, secrets[j])
			assert.Equal(t, string(datas[i]), string(data), "secret: %v", secrets[j])
			// Check that timeout works as well
			time.Sleep(5 * time.Millisecond)
			assert.Nil(t, Extract(sig, secrets[j]), "secret: %v", secrets[j])
		}
	}
}
