// Package auth wraps around an http multiplexer (e.g. http.ServeMux), providing
// automatic rate-limiting and user authentication. See the package README for
// more documentation and examples
package auth

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mediocregopher/mediocre-api/auth/apitok"
	"github.com/mediocregopher/mediocre-api/auth/usertok"
)

// Various error responses this package may return (these will all be appended
// with a newline in the final output)
var (
	APITokenMissing     = "api token missing"
	APITokenInvalid     = "api token invalid"
	APITokenRateLimited = "chill bro..."
	IPAddrRateLimited   = "chill bro...."
	UserTokenMissing    = "user token missing"
	UserTokenInvalid    = "user token invalid"
	SecretNotSet        = "secret not set on server"
	UnknownProblem      = "unknown problem"
)

// Various http headers which this package will look for
const (
	APITokenHeader  = "X-API-TOKEN"
	UserTokenHeader = "X-USER-TOKEN"
)

// A Muxer is set up with strings (e.g. "/foo", "/") which it will subsequently
// match incoming requests to and under normal circumstances execute the
// http.Handler for those requests. http.ServeMux is an example of a Muxer.
type Muxer interface {

	// Handler returns the handler to use for the given request. If the path is
	// not in its canonical form, the handler will be an internally-generated
	// handler that redirects to the canonical path.
	//
	// Handler also returns the registered pattern that matches the request or,
	// in the case of internally-generated redirects, the pattern that will
	// match after following the redirect.
	//
	// If there is no registered handler that applies to the request, Handler
	// returns a “page not found” handler and an empty pattern.
	Handler(*http.Request) (http.Handler, string)
}

// HandlerFlag is used to set options on a particular handler (see
// SetHandlerFlags)
type HandlerFlag int

const (
	// IPRateLimited sets the endpoint as being rate-limited by IP instead of by
	// api token.  Should be used sparingly (preferably only on the endpoint
	// which doles out the api tokens)
	IPRateLimited HandlerFlag = 1 << iota

	// NoAPITokenRequired sets the endpoint as not requiring an api token to be
	// used. Obviously api token rate limiting will not be used on the endpoint
	// either
	NoAPITokenRequired

	// RequireUserAuthGet sets the endpoint as requiring a valid user token for
	// GET requests
	RequireUserAuthGet

	// RequireUserAuthPost sets the endpoint as requiring a valid user token for
	// POST requests
	RequireUserAuthPost

	// RequireUserAuthPut sets the endpoint as requiring a valid user token for
	// PUT requests
	RequireUserAuthPut

	// RequireUserAuthHead sets the endpoint as requiring a valid user token for
	// HEAD requests
	RequireUserAuthHead

	// RequireUserAuthDelete sets the endpoint as requiring a valid user token
	// for DELETE requests
	RequireUserAuthDelete

	// RequireUserAuthAlways sets the endpoint as requiring a valid user token
	// no matter what the request type is
	RequireUserAuthAlways = RequireUserAuthGet | RequireUserAuthPost | RequireUserAuthPut | RequireUserAuthHead | RequireUserAuthDelete
)

type handlerOpt struct {
	flags HandlerFlag
}

var blankHandlerOpt handlerOpt

// APIOpts describe all the user set-able modifications which can be made to the
// behavior of API. A default APIOpts is returned by NewAPIOpts, and fields on
// it can be modified before being passed into NewAPI
type APIOpts struct {

	// Contains the rate limiting implementation. The fields on the RateLimiter
	// can be changed prior to API actually serving requests
	RateLimiter *apitok.RateLimiter

	// The secret used when signing data for rate limiting and user
	// authentication tokens. If this is null rate-limiting will be disabled and
	// any endpoints needing user authentication will return a 500 error
	Secret []byte
}

// NewAPIOpts returns the default APIOpts struct, the fields on which can be
// changed in order to modify behavior before being passed into NewAPI
func NewAPIOpts() *APIOpts {
	return &APIOpts{
		RateLimiter: apitok.NewRateLimiter(),
	}
}

// API is an http.Handler which wraps a given Muxer, providing automatic
// rate-limiting and user authentication
type API struct {
	mux         Muxer
	handlerOpts map[string]*handlerOpt
	o           *APIOpts
}

// NewAPI takes in a Muxer and returns an API which wraps around it. If the
// given APIOpts is nil the default will be used
func NewAPI(mux Muxer, o *APIOpts) *API {
	if o == nil {
		o = NewAPIOpts()
	}
	return &API{
		mux:         mux,
		handlerOpts: map[string]*handlerOpt{},
		o:           o,
	}
}

// NewAPIToken generates a new api token which will work with the secret this
// API is using.  Will return empty string if Secret isn't set
func (a *API) NewAPIToken() string {
	if a.o.Secret == nil {
		return ""
	}
	return apitok.New(a.o.Secret)
}

// GetAPIToken returns the api token as sent by the client. Will return empty
// string if the client has not set one
func (a *API) GetAPIToken(r *http.Request) string {
	return r.Header.Get(APITokenHeader)
}

// NewUserToken generates a new user token for the given user identifier (which
// can later be retrieved from the token using GetUser). Will return empty
// string if Secret isn't set
func (a *API) NewUserToken(user string) string {
	if a.o.Secret == nil {
		return ""
	}
	return usertok.New(user, a.o.Secret)
}

// GetUser returns the user identifier held by the user token from the given
// request. Returns empty string if the user token header isn't set or invalid,
// or if Secret isn't set
func (a *API) GetUser(r *http.Request) string {
	if a.o.Secret == nil {
		return ""
	}
	userTok := r.Header.Get(UserTokenHeader)
	if userTok == "" {
		return ""
	}
	return usertok.ExtractUser(userTok, a.o.Secret)
}

func (a *API) getHandlerOpt(pattern string) *handlerOpt {
	if _, ok := a.handlerOpts[pattern]; !ok {
		a.handlerOpts[pattern] = &handlerOpt{}
	}
	return a.handlerOpts[pattern]
}

// SetHandlerFlags sets option flags on the given endpoint pattern
func (a *API) SetHandlerFlags(pattern string, flags HandlerFlag) {
	o := a.getHandlerOpt(pattern)
	o.flags = flags
}

// Implements ServeHTTP for the http.Handler interface. This will ensure the
// request meets all the requirements it needs to (rate-limit, user tokens,
// etc...) then gets the real handler from calling Handler on mux and passes the
// request off to that
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler, pattern := a.mux.Handler(r)
	o, ok := a.handlerOpts[pattern]
	if !ok {
		o = &blankHandlerOpt
	}

	// This could be the X-API-TOKEN or the IP, depending a flag in handlerOpts.
	// If it's left empty we won't bother calling Use on it at the end of the
	// query
	var token string

	if o.flags&IPRateLimited != 0 {
		switch a.o.RateLimiter.CanUseRaw(r.RemoteAddr) {
		case apitok.Success:
			token = r.RemoteAddr
		case apitok.RateLimited:
			w.WriteHeader(420)
			fmt.Fprintln(w, IPAddrRateLimited)
			return
		default:
			w.WriteHeader(400)
			fmt.Fprintln(w, UnknownProblem)
			return
		}

		// We only rate limit by api token if we aren't rate limiting by ip
	} else if o.flags&NoAPITokenRequired == 0 && a.o.Secret != nil {
		apiToken := a.GetAPIToken(r)
		if apiToken == "" {
			w.WriteHeader(400)
			fmt.Fprintln(w, APITokenMissing)
			return
		}
		switch a.o.RateLimiter.CanUse(apiToken, a.o.Secret) {
		case apitok.Success:
			token = apiToken
		case apitok.TokenInvalid:
			w.WriteHeader(400)
			fmt.Fprintln(w, APITokenInvalid)
			return
		case apitok.RateLimited:
			w.WriteHeader(420)
			fmt.Fprintln(w, IPAddrRateLimited)
			return
		default:
			w.WriteHeader(400)
			fmt.Fprintln(w, UnknownProblem)
			return
		}
	}

	if a.requiresUserAuth(r, o.flags) {
		if a.o.Secret == nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, SecretNotSet)
			return
		}

		userTok := r.Header.Get(UserTokenHeader)
		if userTok == "" {
			w.WriteHeader(400)
			fmt.Fprintln(w, UserTokenMissing)
			return
		}

		if usertok.ExtractUser(userTok, a.o.Secret) == "" {
			w.WriteHeader(400)
			fmt.Fprintln(w, UserTokenInvalid)
			return
		}
	}

	start := time.Now()
	handler.ServeHTTP(w, r)

	if token != "" {
		elapsed := time.Since(start)
		a.o.RateLimiter.Use(token, elapsed)
	}
}

func (a *API) requiresUserAuth(r *http.Request, flags HandlerFlag) bool {
	if flags&RequireUserAuthAlways == RequireUserAuthAlways {
		return true
	}
	var checkFlag HandlerFlag
	switch r.Method {
	case "GET":
		checkFlag = RequireUserAuthGet
	case "POST":
		checkFlag = RequireUserAuthPost
	case "PUT":
		checkFlag = RequireUserAuthPut
	case "HEAD":
		checkFlag = RequireUserAuthHead
	case "DELETE":
		checkFlag = RequireUserAuthDelete
	default:
		return false
	}

	return flags&checkFlag != 0
}
