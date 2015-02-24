// Package commontest holds methods which are helpful when writing tests within
// mediocre-api
package commontest

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/mediocregopher/mediocre-api/auth"
	"github.com/mediocregopher/mediocre-api/common"
	"github.com/mediocregopher/radix.v2/pool"
)

// APIStarterKit returns a populated APIOpts and a Cmder which can be used as
// generic entities for testing
func APIStarterKit() (*auth.APIOpts, common.Cmder) {
	p, err := pool.New("tcp", "localhost:6379", 10)
	if err != nil {
		panic(err)
	}

	o := auth.NewAPIOpts()
	o.Secret = []byte("SHOOPDAWOOP")
	return o, p
}

// RandStr returns a string of random alphanumeric characters
func RandStr() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
