package apitok

import (
	"encoding/json"
	"time"

	"code.google.com/p/go-uuid/uuid"

	"github.com/mediocregopher/mediocre-api/api/sig"
)

type apiTokData struct {
	UUID string
	TS   time.Time
}

// Returns an api token, signed with the given secret
func New(secret []byte) string {
	tok := apiTokData{
		UUID: uuid.New(),
		TS:   time.Now(),
	}
	d, _ := json.Marshal(tok)
	return sig.New(d, secret)
}

// Implements a token bucket rate limiting system on a per-api-token basis. From
// here on the items each bucket is filled with will be called "passes" to avoid
// conflicting names.
//
// Each api token has its own pass bucket. The bucket is periodically filled
// with passes (at a rate of PerInterval passes every Interval), with a maximum
// of Capactiy passes before new passes may not be added. Every time an api
// token wants to take an action it removes a single pass from its bucket. This
// allows for request bursts (assuming Capacity is greater than PerInterval),
// while at the same time preventing sustained high load.
type RateLimiter struct {

	// The maximum number of passes available per api token. Default is 30
	Capacity int64

	// How often new passes are added to each bucket. Default is 5 seconds
	Interval time.Duration

	// How many passes are added to each bucket every Interval. Default is 5
	PerInterval int64

	// Time before an api token is no longer considered valid no matter what
	TokenTimeout time.Duration

	// Where to actually store data pertaining to the RateLimiter. Default is
	// a new instance of RateLimitMem (which stores all data in memory)
	Backend RateLimitStore
}

// Returns a new RateLimiter initialized with all default values. The fields can
// be changed to the desired values before the RateLimiter starts being used
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		Capacity:     30,
		Interval:     5 * time.Second,
		PerInterval:  5,
		TokenTimeout: 24 * time.Hour,
		Backend:      NewRateLimitMem(),
	}
}

// UseResult describes each of the outcomes that can occur when calling Use() or
// UseRaw()
type UseResult int

const (
	Success UseResult = iota
	TokenInvalid
	TokenExpired
	RateLimited
)

// Attempts to use the given api token. May return any of the UseResults
func (r *RateLimiter) Use(token string, secret []byte) UseResult {
	d := sig.Extract(token, secret)
	if d == nil {
		return TokenInvalid
	}
	var tok apiTokData
	if err := json.Unmarshal(d, &tok); err != nil {
		return TokenInvalid
	}

	if time.Since(tok.TS) > r.TokenTimeout {
		return TokenExpired
	}

	return r.UseRaw(token)
}

// Attempts to "use" the given identifier, checking that it has enough passes in
// its bucket first. Will either return Success or RateLimited
func (r *RateLimiter) UseRaw(identifier string) UseResult {

	// TODO there's a slight bug in this portion. If there's passes in the
	// bucket to use and something is using them in a tight enough loop so that
	// toAdd is always zero (since time since last modified is always so small),
	// but enough time passes that passes *should* have been added, it might
	// happen that an app gets blocked when it shouldn't.
	lm := r.Backend.LastModified(identifier)
	since := time.Since(lm)
	toAdd := (since.Nanoseconds() / r.Interval.Nanoseconds()) * r.PerInterval
	if toAdd > 0 {
		r.Backend.IncrBy(identifier, toAdd, r.Capacity)
	}

	numPasses, floored := r.Backend.DecrBy(identifier, 1, 0)

	// If it is zero it was 1 before we took one away, so it still passes
	if floored || numPasses < 0 {
		return RateLimited
	}

	return Success
}
