// Package commontest holds methods which are helpful when writing tests within
// mediocre-api
package commontest

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/mediocregopher/mediocre-api/auth"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/util"
)

// APIStarterKit returns a populated APIOpts and a Cmder which can be used as
// generic entities for testing
func APIStarterKit() (*auth.APIOpts, util.Cmder) {
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

//RandEmail returns a string which could plausibly be an email (but definitely
//isn't a real one)
func RandEmail() string {
	s := RandStr()
	return fmt.Sprintf("%s@%s.com", s[:4], s[4:8])
}
