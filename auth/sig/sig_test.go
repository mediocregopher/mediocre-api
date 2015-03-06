package sig

import (
	"crypto/rand"
	. "testing"

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
			sig := New(datas[i], secrets[j])
			data := Extract(sig, secrets[j])
			assert.Equal(t, string(datas[i]), string(data), "secret: %v", secrets[j])
		}
	}
}
