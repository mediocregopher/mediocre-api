package apitok

import (
	"encoding/json"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/mediocregopher/mediocre-api/auth/sig"
)

type apiTokData struct {
	UUID string
	TS   time.Time
}

// New returns an api token, signed with the given secret
func New(secret []byte) string {
	tok := apiTokData{
		UUID: uuid.New(),
		TS:   time.Now(),
	}
	d, _ := json.Marshal(tok)
	return sig.New(d, secret)
}

// RateLimiter implements a token bucket rate limiting system on a per-api-token
// basis, except instead of tokens in the bucket we instead use time. When a
// request is made it's first checked if the bucket is empty, if so the request
// is rejected. When the request is completed the time it took to complete is
// removed from the bucket (the bucket may have negative time in it)
//
// At intervals new time is added to the bucket, up to a specified maximum
// capacity. This system has a few nice qualities:
// * Bursts of load are allowed, but not sustained load
// * Load is determined by actual time per request, so the system can't be
//   easily gamed by making high cost requests.
// * It's pretty cheap and easy to implement
type RateLimiter struct {

	// The maximum time available per api token. Default is 30 seconds
	Capacity time.Duration

	// How often time is added to each bucket. Default is 5 seconds
	Interval time.Duration

	// How much time is added to each bucket every Interval. Default is 5
	// seconds
	PerInterval time.Duration

	// Where to actually store data pertaining to the RateLimiter. Default is
	// a new instance of RateLimitMem (which stores all data in memory)
	Backend RateLimitStore
}

// NewRateLimiter returns a new RateLimiter initialized with all default values.
// The fields can be changed to the desired values before the RateLimiter starts
// being used
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		Capacity:    30 * time.Second,
		Interval:    5 * time.Second,
		PerInterval: 5 * time.Second,
		Backend:     NewRateLimitMem(),
	}
}

// UseResult describes each of the outcomes that can occur when calling CanUse()
// or CanUseRaw()
type UseResult int

const (
	// Success means that the token can be used (the bucket for it isn't empty)
	Success UseResult = iota

	// TokenInvalid means that the token isn't a valid token
	TokenInvalid

	// RateLimited means that the token can't be used because its bucket is
	// empty
	RateLimited
)

// CanUse attempts to use the given api token (calling sig.Verify on it first).
// May return any of the UseResults
func (r *RateLimiter) CanUse(token string, secret []byte) UseResult {
	if !sig.Verify(token, secret) {
		return TokenInvalid
	}

	return r.CanUseRaw(token)
}

// CanUseRaw checks if you can "use" the given identifier, which could be
// anything, checking that it has a non-zero amount of time in its bucket first.
// Will either return Success or RateLimited
func (r *RateLimiter) CanUseRaw(identifier string) UseResult {

	// TODO there's a slight bug in this portion. If there's time in the bucket
	// to use and something is using it in a tight enough loop so that toAdd is
	// always zero (since time since last modified is always so small), but
	// enough time passes that time *should* have been added, it might happen
	// that an app gets blocked when it shouldn't.
	lm := r.Backend.LastModified(identifier)
	since := time.Since(lm)
	toAdd := (since / r.Interval) * r.PerInterval

	var timeLeft int64
	if toAdd > 0 {
		timeLeft, _ = r.Backend.IncrByCeil(identifier, toAdd.Nanoseconds(), r.Capacity.Nanoseconds())
	} else {
		timeLeft = r.Backend.Get(identifier)
	}

	if timeLeft <= 0 {
		return RateLimited
	}

	return Success
}

// Use removes the given amount of time for the identifier. Assumes that the
// identifier is legitimate.
func (r *RateLimiter) Use(identifier string, toRemove time.Duration) {
	r.Backend.DecrBy(identifier, toRemove.Nanoseconds())
}
