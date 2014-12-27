package apitok

import (
	"sync"
	"time"
)

// Storage used to store data needed for rate limiting. All methods must be
// thread-safe with each other
type RateLimitStore interface {

	// Increments the given key by the given amount, and returns the value of
	// the key after the increment. If the key would go over the max value given
	// it should be instead set to that max (and have that max returned). True
	// should be returned in this case as well. The key should be assumed to be
	// 0 if it didn't previously exist
	IncrBy(key string, amount, max int64) (int64, bool)

	// Decrements the given key by the given amount, and returns the value of
	// the key after the decrement. If the key would go under the min value
	// given it should be instead set to that min (and have that min returned).
	// True should be returned in this case as well. The key should be assumed
	// to be 0 if it didn't previously exist
	DecrBy(key string, amount, min int64) (int64, bool)

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

// An implementation of RateLimitStore which keeps all data in memory protected
// by a mutex
type RateLimitMem struct {
	m map[string]keyval
	l sync.RWMutex
}

func NewRateLimitMem() *RateLimitMem {
	return &RateLimitMem{
		m: map[string]keyval{},
	}
}

// Implementation of IncrBy for RateLimitStore
func (m *RateLimitMem) IncrBy(key string, amount, max int64) (int64, bool) {
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

// Implementation of DecrBy for RateLimitStore
func (m *RateLimitMem) DecrBy(key string, amount, min int64) (int64, bool) {
	m.l.Lock()
	defer m.l.Unlock()
	var mind bool
	newAmount := m.m[key].val - amount
	if newAmount < min {
		mind = true
		newAmount = min
	}
	m.m[key] = keyval{
		val:   newAmount,
		tsMod: time.Now(),
	}
	return newAmount, mind
}

// Implementation of LastModified for RateLimitStore
func (m *RateLimitMem) LastModified(key string) time.Time {
	m.l.RLock()
	defer m.l.RUnlock()
	return m.m[key].tsMod
}

// Implementation of Clean for RateLimitStore
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
