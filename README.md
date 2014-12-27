# mediocre-api

A simple wrapper around go's standard `net/http` package. It provides:

* Rate-limiting based on an api token (or optionally on an ip address)

* User authentication and request signing

* Interoperability with other packages besides `net/http`. See the Muxer
  interface definition

* Swappable storage backend for rate-limiting data

## Example

Here's an example simple but complete api, and an explanation for each step:

```go
package main

import (
	"fmt"
	"net/http"
	"github.com/mediocregopher/mediocre-api/api"
)

func main() {
	// The mediocre api wraps a normal http request muxer. It can wrap anything
	// else which implements the Muxer interface as well.
	mux := http.NewServeMux()
	a := api.NewAPI(mux)

	// Rate-limiting will not work without a set secret, nor will user
	// authentication. If clients are load-balanced across multiple instances of
	// this process the processes must all be using the same secret
	a.Secret = []byte("wubalubadubdub!")

	// By default all requests require an api token, and are rate-limited based
	// on that. There needs to be an endpoint for the client to retrieve an api
	// token though. This endpoint will do just that. It will be rate limited
	// based on ip instead of api token, and directly returns a new api token
	// which can be used by the client
	tokenEndpt := "/new-api-token"
	a.SetHandlerFlags(tokenEndpt, api.IPRateLimited | api.NoAPITokenRequired)
	mux.HandleFunc(tokenEndpt, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, a.NewAPIToken())
	})

	// A normal request is, as mentioned before, rate-limited by api token,
	// which must be set on the X-API-TOKEN header. Defining a normal endpoint
	// doesn't require any decoration
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		msg := r.FormValue("message")
		fmt.Fprintln(w, msg)
	})

	// In order to use user authenticated endpoints the user must retrieve a
	// user token and user secret, and use the secret to sign their requests.
	// The following lets a user login and retrieve their tokens
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		username := r.FormValue("username")
		userTok, userSecret := a.NewUserTokenSecret(username)
		fmt.Fprintln(w, userTok)
		fmt.Fprintln(w, userSecret)
	})

	// This endpoint is only available to users who have logged in and properly
	// send their user token (through X-USER-TOKEN) and a request signature
	// (through X-USER-SIGNATURE). It retrieves their username that thay're
	// logged in as and welcomes them to the site
	welcomeEndpt := "/welcome"
	a.SetHandlerFlags(welcomeEndpt, api.RequireUserAuth)
	mux.HandleFunc(welcomeEndpt, func(w http.ResponseWriter, r *http.Request) {
		username := api.GetUser(r)
		fmt.Fprintf(w, "Welcome to the site, %s!", username)
	})

	// We serve requests using our api, which will hand off the actual requests
	// (once they're validated) to the mux we made
	http.ListenAndServe(":8080", a)
}
```

## API Token

The api token is the basis of rate-limiting in most cases. All requests by
default require an api key to be set on the header `X-API-TOKEN`, which must be
retrieved from the api itself.

### Rate-Limiting

Rate limiting is based on a token bucket system. You can read more about it in
the apitok package docs. Parameters for rate-limiting on the api can be set by
modifying the `RateLimiter`'s fields on the `API` struct returned from
`NewAPI`.

The storage backend for rate limiting can also be changed. By default the data
is stored in memory, but anything implementing `apitok.RateLimitStore` can
replace it in the `api.RateLimiter.Backend` field.

Any modifications to rate limiting fields must be done before the call to
`ListenAndServe`.

## User authentication

User authentication is based upon a token and a secret, both of which are
retrieved through the api during a login. The user token is sent with every
request which is user authenticated, but the secret is never ever ever sent to
the server or outside the client at all.

To make a user authenticated request two additional headers (besides the
`X-API-TOKEN`) must be set: `X-USER-TOKEN` and `X-USER-SIGNATURE`.
`X-USER-TOKEN` is simple the token as returned by the login call. The signature
is generated as follows:

the HMAC-SHA1 of the following (all concatonated, with no
separator): The request type (e.g. POST, GET), the request URL (with hostname,
port, and GET parameters included), and the POST data (or empty string if there
is none). The secret returned by the sign-in request should be used as the
secret for the HMAC-SHA1. The result should be base64 encoded.

### User authentication example

Let's say the user signed in and they received back a user token of `abcdef` and
a secret of `012345`. They are trying to make a POST request to:

    http://localhost/api/whatever?doicare=no&howboutnow=nope

with POST data of

    this beat is bananas

The signature would be computed as:

    // hmac-sha1(<secret>, <data>)
    hmac-sha1(`012345`, 'POSThttp://localhost/api/whatever?doicare=no&howboutnow=nopethis beat is bananas')
    // zoDwmDig84OWyrQB95JJHmw5Oos=

And the full request would look like:

    POST /api/whatever?doicare=no&howboutnow=nope HTTP/1.1
    Host: localhost
    Accept: */*
    Content-Length: 20
    X-API-TOKEN: sometoken
    X-USER-TOKEN: abcdef
    X-USER-SIGNATURE: zoDwmDig84OWyrQB95JJHmw5Oos=

    this beat is bananas
