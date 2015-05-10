# mediocre-api/auth

[![GoDoc](https://godoc.org/github.com/mediocregopher/mediocre-api/auth?status.svg)](https://godoc.org/github.com/mediocregopher/mediocre-api/auth)

A simple http middleware. It provides:

* Rate-limiting based on an api token (or optionally on an ip address). Rate
  limiting is based on actual time to complete requests, not just number of
  requests

* User authentication

* Interoperability with other packages besides `net/http`

* Swappable storage backend for rate-limiting data

## API Token

The api token is the basis of rate-limiting in most cases. All requests by
default require an api key to be set on the header `X-API-TOKEN`, which must be
retrieved from the api itself.

**API tokens are only valid for 3 hours. The application must expect this and
get a new one periodically**

### Rate-Limiting

Rate limiting is based on a time bucket system. You can read more about it in
the apitok package docs. Parameters for rate-limiting on the api can be set by
modifying the `RateLimiter`'s fields on the `API` struct returned from
`NewAPI`.

The storage backend for rate limiting can also be changed. By default the data
is stored in memory, but anything implementing `apitok.RateLimitStore` can
replace it in the `api.RateLimiter.Backend` field.

Any modifications to rate limiting fields must be done before the call to
`ListenAndServe`.

## User authentication

User authentication is based upon a simple user token system. A client retrieves
a user token from the api, which authenticates the user however it wants and
returns a token generated through `NewUserToken`. This token must be included
with any requests that require user authentication as the `X-USER-TOKEN` header.
The api may retrieve the authenticated user identifier using `GetUser`.

## Example

Here's an example simple but complete api, and an explanation for each step:

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/mediocregopher/mediocre-api/auth"
)

func main() {
	// Rate-limiting will not work without a set secret, nor will user
	// authentication. If clients are load-balanced across multiple instances of
	// this process the processes must all be using the same secret
	a := auth.NewAPI()
	a.Secret = []byte("wubalubadubdub!")

	// By default all requests require an api token, and are rate-limited based
	// on that. There needs to be an endpoint for the client to retrieve an api
	// token though. This endpoint will do just that. It will be rate limited
	// based on ip instead of api token, and directly returns a new api token
	// which can be used by the client
	http.Handle("/new-api-token", a.Wrapper(auth.IPRateLimited)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, a.NewAPIToken())
		}),
	))

	// A single Wrapper can be used for multiple endpoints to make code more
	// readable, and to better interface with packages like alice
	authDefault := a.Wrapper(auth.Default)

	// A normal request is, as mentioned before, rate-limited by api token,
	// which must be set on the X-API-TOKEN header.
	http.Handle("/echo", authDefault(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			msg := r.FormValue("message")
			fmt.Fprintln(w, msg)
		}),
	))

	// In order to use user authenticated endpoints the user must retrieve a
	// user token, and use that to authenticate their requests. The following
	// lets a user login and retrieve their token
	http.Handle("/login", authDefault(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username := r.FormValue("username")
			userTok := a.NewUserToken(username)
			fmt.Fprintln(w, userTok)
		}),
	))

	// This endpoint is only available to users who have logged in and properly
	// send their user token (through X-USER-TOKEN). It retrieves their username
	// that thay're logged in as and welcomes them to the site
	http.Handle("/welcome", a.Wrapper(auth.RequireUserAuthAlways)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username := r.FormValue("_asUser")
			fmt.Fprintf(w, "Welcome to the site, %s!", username)
		}),
	))

	http.ListenAndServe(":8080", nil)
}
```
