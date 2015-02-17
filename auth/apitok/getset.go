package apitok

import (
	"sync"
	"time"
)

// RateLimitStore is used to store data needed for rate limiting. All methods
// must be thread-safe with each other
type RateLimitStore interface {

	// Increments the given key by the given amount, and returns the value of
	// the key after the increment. If the key would go over the max value given
	// it should be instead set to that max (and have that max returned). True
	// should be returned in this case as well. The key should be assumed to be
	// 0 if it didn't previously exist
	IncrByCeil(key string, amount, max int64) (int64, bool)

	// Decrements the given key by the given amount, and returns the value of
	// the key after the decrement. The key should be assumed to be 0 if it
	// didn't previously exist
	DecrBy(key string, amount int64) int64

	// Retrieve the value of the given key. The key should be assumed to be 0 if
	// it didn't previously exists
	Get(key string) int64

	// The time the key was last modified. Returns the zero time if the key did
	// not previously exist
	LastModified(key string) time.Time

	// Will be called once a minute, and should be used to atomically clean up
	// any data which hasn't been modified in more than the given duration. If
	// the backing storage method has a different way of implicitely cleaning up
	// data (e.g. redis' EXPIRE command) than this may do nothing
	Clean(time.Duration)
}

type keyval struct {
	val   int64
	tsMod time.Time
}

// RateLimitMem is an implementation of RateLimitStore which keeps all data in
// memory protected by a mutex
type RateLimitMem struct {
	m map[string]keyval
	l sync.RWMutex
}

// NewRateLimitMem returns a new RateLimitMem, ready to be used as a
// RateLimitStore
func NewRateLimitMem() *RateLimitMem {
	return &RateLimitMem{
		m: map[string]keyval{},
	}
}

// IncrByCeil is an implementation of IncrByCeil for RateLimitStore
func (m *RateLimitMem) IncrByCeil(key string, amount, max int64) (int64, bool) {
	m.l.Lock()
	defer m.l.Unlock()
	var maxd bool
	newAmount := m.m[key].val + amount
	if newAmount > max {
		maxd = true
		newAmount = max
	}
	m.m[key] = keyval{
		val:   newAmount,
		tsMod: time.Now(),
	}
	return newAmount, maxd
}

// DecrBy is an implementation of DecrBy for RateLimitStore
func (m *RateLimitMem) DecrBy(key string, amount int64) int64 {
	m.l.Lock()
	defer m.l.Unlock()
	newAmount := m.m[key].val - amount
	m.m[key] = keyval{
		val:   newAmount,
		tsMod: time.Now(),
	}
	return newAmount
}

// Get is an implementation of Get for RateLimitStore
func (m *RateLimitMem) Get(key string) int64 {
	m.l.Lock()
	defer m.l.Unlock()
	return m.m[key].val
}

// LastModified is an implementation of LastModified for RateLimitStore
func (m *RateLimitMem) LastModified(key string) time.Time {
	m.l.RLock()
	defer m.l.RUnlock()
	return m.m[key].tsMod
}

// Clean is an implementation of Clean for RateLimitStore
func (m *RateLimitMem) Clean(staleTimeout time.Duration) {
	tsThresh := time.Now().Add(-1 * staleTimeout)

	m.l.RLock()
	keysToClean := make([]string, 0, len(m.m))
	for key := range m.m {
		if tsThresh.After(m.m[key].tsMod) {
			keysToClean = append(keysToClean, key)
		}
	}
	m.l.RUnlock()

	m.l.Lock()
	defer m.l.Unlock()
	for _, key := range keysToClean {
		delete(m.m, key)
	}
}
