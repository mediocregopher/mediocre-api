// Package auth wraps around an http.Handler, providing automatic rate-limiting
// and user authentication. See the package README for more documentation and
// examples
package auth

import (
	"bytes"
	"net/http"
	"strings"
	"time"

	"github.com/mediocregopher/mediocre-api/auth/apitok"
	"github.com/mediocregopher/mediocre-api/auth/usertok"
	"github.com/mediocregopher/mediocre-api/common"
)

// Various error responses this package may return (these will all be appended
// with a newline in the final output)
var (
	ErrAPITokenMissing     = common.ExpectedErr{Code: 400, Err: "api token missing"}
	ErrAPITokenInvalid     = common.ExpectedErr{Code: 400, Err: "api token invalid"}
	ErrAPITokenRateLimited = common.ExpectedErr{Code: 420, Err: "chill bro"}
	ErrIPAddrRateLimited   = common.ExpectedErr{Code: 420, Err: "chill bro"}
	ErrUserTokenMissing    = common.ExpectedErr{Code: 400, Err: "user token missing"}
	ErrUserTokenInvalid    = common.ExpectedErr{Code: 400, Err: "user token invalid"}
	ErrSecretNotSet        = common.ExpectedErr{Code: 500, Err: "secret not set on server"}
	ErrUnknownProblem      = common.ExpectedErr{Code: 500, Err: "unknown problem"}
)

// Various http headers which this package will look for
const (
	APITokenHeader  = "X-API-TOKEN"
	UserTokenHeader = "X-USER-TOKEN"
)

// HandlerFlag is used to set options on a particular handler
type HandlerFlag int

const (
	// Default means no flags are set on an endpoint. It will rate-limit the
	// client based on their api token, and that is all.
	Default = 0

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

// API can return an http.Handler which wraps around
// another http.Handler, providing automatic rate-limiting and user
// authentication
type API struct {

	// Contains the rate limiting implementation. The fields on the RateLimiter
	// can be changed prior to actually serving requests (generally before
	// ListenAndServe is called)
	RateLimiter *apitok.RateLimiter

	// The secret used when signing data for rate limiting and user
	// authentication tokens. If this is nil rate-limiting will be disabled and
	// any endpoints needing user authentication will return a 500 error.
	// Defaults to nil
	Secret []byte

	// If set, when a user is successfully authenticated using their user token
	// their username will be added onto the request itself under the GET
	// request named by this field. Defaults to empty string (off).
	//
	// For example, if this was set to "asUser" then all requests which
	// successfully authenticate a user will have "?asUser=<username>" appended
	// to their URL before being forwarded down the handler chain.
	UserAuthGetParam string
}

// NewAPI returns an API with all of its fields initialized to their default
// values. Any of the fields may be modified before actually serving requests
// (generally before ListenAndServe is called)
func NewAPI() *API {
	return &API{
		RateLimiter: apitok.NewRateLimiter(),
	}
}

// NewAPIToken generates a new api token which will work with the Secret this
// API is using. Will return empty string if Secret isn't set
func (a *API) NewAPIToken() string {
	if a.Secret == nil {
		return ""
	}
	return apitok.New(a.Secret)
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
	if a.Secret == nil {
		return ""
	}
	return usertok.New(user, a.Secret)
}

// GetUser returns the user identifier held by the user token from the given
// request. Returns empty string if the user token header isn't set or invalid,
// or if Secret isn't set
func (a *API) GetUser(r *http.Request) string {
	if a.Secret == nil {
		return ""
	}
	userTok := r.Header.Get(UserTokenHeader)
	if userTok == "" {
		return ""
	}
	return usertok.ExtractUser(userTok, a.Secret)
}

// Wrapper returns a function which takes in http.Handlers and wraps them,
// returning a new http.Handler which will execute some logic based on the given
// flags, calling the function's passed in http.Handler if everything checks out
func (a *API) Wrapper(flags HandlerFlag) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// This could be the X-API-TOKEN or the IP, depending on flags If
			// it's left empty we won't bother calling Use on it at the end of
			// the query
			var token string

			secret := a.Secret

			if flags&IPRateLimited != 0 {
				remoteIP := r.RemoteAddr[:strings.LastIndex(r.RemoteAddr, ":")]
				switch a.RateLimiter.CanUseRaw(remoteIP) {
				case apitok.Success:
					token = r.RemoteAddr
				case apitok.RateLimited:
					common.HTTPError(w, r, ErrIPAddrRateLimited)
					return
				default:
					common.HTTPError(w, r, ErrUnknownProblem)
					return
				}

				// We only rate limit by api token if we aren't rate limiting by ip
			} else if flags&NoAPITokenRequired == 0 && secret != nil {
				apiToken := a.GetAPIToken(r)
				if apiToken == "" {
					common.HTTPError(w, r, ErrAPITokenMissing)
					return
				}
				switch a.RateLimiter.CanUse(apiToken, secret) {
				case apitok.Success:
					token = apiToken
				case apitok.TokenInvalid:
					common.HTTPError(w, r, ErrAPITokenInvalid)
					return
				case apitok.RateLimited:
					common.HTTPError(w, r, ErrAPITokenRateLimited)
					return
				default:
					common.HTTPError(w, r, ErrUnknownProblem)
					return
				}
			}

			user, err := a.authdUser(r)
			if a.requiresUserAuth(flags, r) && err != nil {
				common.HTTPError(w, r, err)
				return
			}
			if user != "" && a.UserAuthGetParam != "" {
				values := r.URL.Query()
				values.Add(a.UserAuthGetParam, user)
				r.URL.RawQuery = values.Encode()
			}

			start := time.Now()
			h.ServeHTTP(w, r)

			if token != "" {
				elapsed := time.Since(start)
				a.RateLimiter.Use(token, elapsed)
			}
		})
	}
}

func (a *API) authdUser(r *http.Request) (string, error) {
	secret := a.Secret
	if secret == nil {
		return "", ErrSecretNotSet
	}

	userTok := r.Header.Get(UserTokenHeader)
	if userTok == "" {
		return "", ErrUserTokenMissing
	}

	user := usertok.ExtractUser(userTok, secret)
	if user == "" {
		return "", ErrUserTokenInvalid
	}

	return user, nil
}

func (a *API) requiresUserAuth(flags HandlerFlag, r *http.Request) bool {
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

// NewRequest returns an *http.Request which will have the appropriate headers
// needed for interacting with an endpoint wrapped by the API. The api token
// generated will be different for every request. If user is non-empty a user
// token for that user will be generated and filled in as well. This funtion is
// primarily useful for testing.
func (a *API) NewRequest(method, endpnt, body, user string) *http.Request {
	r, _ := http.NewRequest(method, endpnt, bytes.NewBufferString(body))
	r.Header.Set(APITokenHeader, a.NewAPIToken())
	if user != "" {
		r.Header.Set(UserTokenHeader, a.NewUserToken(user))
	}
	return r
}
