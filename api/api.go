// Wraps around an http multiplexer (e.g. http.ServeMux), providing automatic
// rate-limiting and user authentication. See the package README for more
// documentation and examples
package api

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/mediocregopher/mediocre-api/api/apitok"
	"github.com/mediocregopher/mediocre-api/api/usertok"
)

// Various error responses this package may return (these will all be appended
// with a newline in the final output)
var (
	APITokenMissing     = "api token missing"
	APITokenInvalid     = "api token invalid"
	APITokenRateLimited = "chill bro..."
	IPAddrRateLimited   = "chill bro...."
	UserTokenSigInvalid = "user token or signature invalid"
	SecretNotSet        = "secret not set on server"
	UnknownProblem      = "unknown problem"
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

type HandlerFlag int

const (
	// Sets the endpoint as being rate-limited by IP instead of by api token.
	// Should be used sparingly (preferably only on the endpoint which doles out
	// the api tokens)
	IPRateLimited HandlerFlag = 1 << iota

	// Sets the endpoint as not requiring an api token to be used. Obviously api
	// token rate limiting will not be used on the endpoint either
	NoAPITokenRequired

	// Sets the endpoint as requiring a valid user in order to be used
	RequireUserAuth

	// Sets the endpoint as not including the POST body when checking the user
	// signature
	UserAuthIgnoreBody
)

type handlerOpt struct {
	flags HandlerFlag
}

var blankHandlerOpt handlerOpt

// An http.Handler which wraps a given Muxer, providing automatic rate-limiting
// and user authentication
type API struct {
	mux         Muxer
	handlerOpts map[string]*handlerOpt

	// Contains the rate limiting implementation. The fields on the RateLimiter
	// can be changed prior to API actually serving requests
	RateLimiter *apitok.RateLimiter

	// The secret used when signing data for rate limiting and user
	// authentication tokens. If this is null rate-limiting will be disabled and
	// any endpoints needing user authentication will return a 500 error
	Secret []byte
}

func NewAPI(mux Muxer) *API {
	return &API{
		mux:         mux,
		handlerOpts: map[string]*handlerOpt{},
		RateLimiter: apitok.NewRateLimiter(),
	}
}

// Generates a new api token which will work with the secret this API is using.
// Will return empty string if Secret isn't set
func (a *API) NewAPIToken() string {
	if a.Secret == nil {
		return ""
	}
	return apitok.New(a.Secret)
}

// Returns the api token as sent by the client. Will return empty string if the
// client has not set one
func GetAPIToken(r *http.Request) string {
	return r.Header.Get("X-API-TOKEN")
}

// Generates new user token and secret for the given user identifier (which can
// later be retrieved from the token using GetUser). Will return empty strings
// if Secret isn't set
func (a *API) NewUserTokenSecret(user string) (string, string) {
	if a.Secret == nil {
		return "", ""
	}
	return usertok.New(user, a.Secret)
}

// Returns the user identifier held by the user token from the given request.
// Returns empty string if the user token header isn't set or invalid
func GetUser(r *http.Request) string {
	userTok := r.Header.Get("X-USER-TOKEN")
	if userTok == "" {
		return ""
	}
	return usertok.ExtractUser(userTok)
}

func (a *API) getHandlerOpt(pattern string) *handlerOpt {
	if _, ok := a.handlerOpts[pattern]; !ok {
		a.handlerOpts[pattern] = &handlerOpt{}
	}
	return a.handlerOpts[pattern]
}

// Sets option flags on the given endpoint pattern
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

	if o.flags&IPRateLimited != 0 {
		switch a.RateLimiter.UseRaw(r.RemoteAddr) {
		case apitok.Success:
			// COntinue on with life
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
	} else if o.flags&NoAPITokenRequired == 0 && a.Secret != nil {
		token := GetAPIToken(r)
		if token == "" {
			w.WriteHeader(400)
			fmt.Fprintln(w, APITokenMissing)
			return
		}
		switch a.RateLimiter.Use(token, a.Secret) {
		case apitok.Success:
			// Continue on with life
		case apitok.TokenInvalid, apitok.TokenExpired:
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

	if o.flags&RequireUserAuth != 0 {
		if a.Secret == nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, SecretNotSet)
			return
		}

		userTok := r.Header.Get("X-User-Token")
		userSig := r.Header.Get("X-User-Signature")
		if userTok == "" || userSig == "" {
			w.WriteHeader(400)
			fmt.Fprintf(w, UserTokenSigInvalid)
			return
		}

		includePost := o.flags&UserAuthIgnoreBody == 0
		data := requestSignData(r, includePost)
		if !usertok.Verify(userTok, userSig, data, a.Secret) {
			w.WriteHeader(400)
			fmt.Fprintf(w, UserTokenSigInvalid)
			return
		}
	}

	handler.ServeHTTP(w, r)
}

func requestSignData(r *http.Request, includePost bool) []byte {
	if includePost && r.ContentLength < 0 {
		return nil
	}

	b := make([]byte, 0, r.ContentLength+64)
	b = append(b, r.Method...)
	b = append(b, r.URL.String()...)

	if includePost {
		buf := bytes.NewBuffer(make([]byte, 0, r.ContentLength))
		tee := io.TeeReader(r.Body, buf)
		body, err := ioutil.ReadAll(tee)
		if err != nil {
			return nil
		}
		b = append(b, body...)
		r.Body = ioutil.NopCloser(buf)
	}

	return b
}
