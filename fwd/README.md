# mediocre-api/fwd

[![GoDoc](https://godoc.org/github.com/mediocregopher/mediocre-api/fwd?status.svg)](https://godoc.org/github.com/mediocregopher/mediocre-api/fwd)

Implements forwarding requests from one http endpoint to another. Useful when
you have some kind of internal api which you wish to expose to the world, but
you want to place some kind of proxy in front which will handle rate-limiting
and authentication and such.

```go
// Forwards a request to another endpoint
http.Handle("/foo", fwd.Abs("http://127.0.0.1:8081/other/foo", nil))

// Forwards a request to another endpoint, appending the URL.Path of the
// original to be relative to this one. e.g. "/fuz/fiz" becomes
// "http://127.0.0.1:8080/rel/fuz/fiz"
http.Handle("/fuz", fwd.Rel("http://127.0.0.1:8081", "/rel", nil))
```
