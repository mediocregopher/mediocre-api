# mediocre-api/xff

[![GoDoc](https://godoc.org/github.com/mediocregopher/mediocre-api/xff?status.svg)](https://godoc.org/github.com/mediocregopher/mediocre-api/xff)

A simple piece of middleware for go's standard `net/http` package which fills in
the `RemoteAddr` field of all `*http.Request` structs with the correct value,
taking into account `X-Forwarded-For` headers on the original request. In
addition, this package:

* Correctly works with IPv6 addresses

* Correctly ignores internal/loopback addresses that may be set in
  `X-Forwarded-For`

Check the godoc for an example
