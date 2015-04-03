// Package room implements an abstraction for a basic room system. Rooms can be
// checked in to and checkd out of by users
package room

import (
	"strings"
	"time"

	"github.com/mediocregopher/mediocre-api/common"
	"github.com/mediocregopher/radix.v2/cluster"
	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/mediocregopher/radix.v2/util"
)

// System holds on to a Cmder and uses it to implement a basic room system
type System struct {
	c      common.Cmder
	o      *Opts
	stopCh chan struct{}
}

// Opts are different options which may be passed into New when creating a
// system. They all have sane defaults which will cover most use cases
type Opts struct {

	// Prefix can be used if you wish to have two separate room systems being
	// persisted on the same Cmder. Prefix will be prepended to all key names
	Prefix string

	// CheckInPeriod indicates how long a user has to check in to a room before
	// they are recorded as not being in it anymore. It should not be set to
	// less than 1 second. Defaults to 30 seconds
	CheckInPeriod time.Duration
}

// New returns a new System which will use the given Cmder as its persistence
// layer. The passed in Opts may be used to modify behavior of the System, or
// may be nil to just use the defaults
func New(c common.Cmder, o *Opts) *System {
	if o == nil {
		o = &Opts{}
	}
	if o.CheckInPeriod < time.Second {
		o.CheckInPeriod = 30 * time.Second
	}

	s := System{
		c:      c,
		o:      o,
		stopCh: make(chan struct{}),
	}
	go s.spin()
	return &s
}

// Key returns a key which can be used to interact with some arbitrary room data
// directly in redis. This is useful if more complicated, lower level operations
// are needed to be done
func (s *System) Key(room string, extra ...string) string {
	k := "room:" + s.o.Prefix + ":{" + room + "}"
	if len(extra) > 0 {
		k += ":" + strings.Join(extra, ":")
	}
	return k
}

// CheckIn records that a user with the given id has joined the given room. The
// user must check in periodically (see the CheckInPeriod field of System) or
// they will be recorded as not in the room anymore
func (s *System) CheckIn(room, id string) error {
	now := time.Now().UTC().UnixNano()
	key := s.Key(room)
	return s.c.Cmd("ZADD", key, now, id).Err
}

// CheckOut records that a user is no longer in a room
func (s *System) CheckOut(room, id string) error {
	key := s.Key(room)
	return s.c.Cmd("ZREM", key, id).Err
}

// Members returns the list of user ids currently checked into a room
func (s *System) Members(room string) ([]string, error) {
	key := s.Key(room)
	return s.c.Cmd("ZRANGE", key, 0, -1).List()
}

// Cardinality returns the number of user ids currently checked into a room
func (s *System) Cardinality(room string) (int64, error) {
	key := s.Key(room)
	return s.c.Cmd("ZCARD", key).Int64()
}

// Stop cleans up any go routines that this room system has running for it. It
// does not remove any persisted data nor close its Cmder
func (s *System) Stop() {
	close(s.stopCh)
}

func (s *System) spin() {
	tick := time.NewTicker(s.o.CheckInPeriod / 2)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			s.removeIdle()
		case <-s.stopCh:
			return
		}
	}
}

func (s *System) removeIdle() error {
	expire := time.Now().UTC().Add(-s.o.CheckInPeriod).UnixNano()
	ch := make(chan string)
	var err error
	if c, ok := s.c.(*cluster.Cluster); ok {
		go func() {
			err = util.ScanCluster(c, ch, s.Key("*"))
		}()
	} else if p, ok := s.c.(*pool.Pool); ok {
		c, err := p.Get()
		if err != nil {
			return err
		}
		go func() {
			err = util.Scan(c, ch, "SCAN", "", s.Key("*"))
		}()
		p.Put(c)
	} else if c, ok := s.c.(*redis.Client); ok {
		go func() {
			err = util.Scan(c, ch, "SCAN", "", s.Key("*"))
		}()
	} else {
		panic("unknown Cmder passed in, sorry :(")
	}

	for key := range ch {
		// TODO We can't report an error from here unfortunately. That's
		// something I'll need to address in radix.v2
		s.c.Cmd("ZREMRANGEBYSCORE", key, "-inf", expire)
	}

	return err
}
