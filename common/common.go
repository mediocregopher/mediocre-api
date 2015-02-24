// Package common is used to hold things that multiple packags within the
// mediocre-api toolkit will need to use or reference
package common

import "github.com/mediocregopher/radix.v2/redis"

// Cmder is an interface which is implemented by both the standard radix client,
// the its client pool, and its cluster client, and is used in order to interact
// with either in a transparent way
type Cmder interface {
	Cmd(string, ...interface{}) *redis.Resp
}
