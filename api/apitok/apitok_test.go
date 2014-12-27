package apitok

import (
	. "testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter(t *T) {
	secret := []byte("wubalubadubdub!")
	token := New(secret)
	assert.NotEqual(t, "", token)

	r := NewRateLimiter()
	r.Capacity = 5
	r.Interval = 1 * time.Second
	r.PerInterval = 1

	for i := 0; i < 5; i++ {
		assert.Equal(t, Success, r.Use(token, secret), "%#v", r.Backend)
	}
	assert.Equal(t, RateLimited, r.Use(token, secret), "%#v", r.Backend)

	time.Sleep(1 * time.Second)
	assert.Equal(t, Success, r.Use(token, secret), "%#v", r.Backend)
	assert.Equal(t, RateLimited, r.Use(token, secret), "%#v", r.Backend)
}
