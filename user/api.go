package user

import (
	"net/http"

	"github.com/asaskevich/govalidator"
	"github.com/mediocregopher/mediocre-api/auth"
	"github.com/mediocregopher/mediocre-api/common"
	"github.com/mediocregopher/mediocre-api/common/apihelper"
	"github.com/mediocregopher/mediocre-api/pickyjson"
	"github.com/mediocregopher/mediocre-api/rend"
	"github.com/mediocregopher/radix.v2/util"
)

// Body size limit for this module is very low, we're not dealing with large
// requests here
const bodySizeLimit = int64(4 * 1024)

var usernameParam = pickyjson.Str{
	MaxLength: 40,
	Func:      govalidator.IsUTFLetterNumeric,
}

var emailParam = pickyjson.Str{
	Map: govalidator.NormalizeEmail,
}

var passwordParam = pickyjson.Str{
	MinLength: 6,
	MaxLength: 255,
}

type userHandlerFunc func(
	s *System, a *auth.API, w http.ResponseWriter, r *http.Request, user string,
)

var userHandlerFuncs = map[string]userHandlerFunc{
	"":      handleGet,
	"token": handleToken,
}

// NewMux returns a new http.Handler (in reality a http.ServeMux) which has the
// basic suite of user creation/modification endpoints. See the package README
// for more information
func NewMux(a *auth.API, c util.Cmder) http.Handler {
	m := http.NewServeMux()
	s := New(c)

	m.HandleFunc("/new-user", func(w http.ResponseWriter, r *http.Request) {
		j := struct {
			Username, Email, Password pickyjson.Str
		}{
			Username: usernameParam.Required(),
			Email:    emailParam.Required(),
			Password: passwordParam.Required(),
		}
		if !apihelper.Prepare(w, r, &j, bodySizeLimit, "POST") {
			return
		}

		err := s.Create(j.Username.Str, j.Email.Str, j.Password.Str)
		common.HTTPError(w, r, err)
	})

	// All other requests will match this, meaning one should be of the form
	// /<username>/action
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var user, action string
		remaining := rend.URL(r.URL, &user, &action)
		if remaining != "" || user == "" {
			http.Error(w, "unknown endpoint", 404)
			return
		}

		f, ok := userHandlerFuncs[action]
		if !ok {
			http.Error(w, "unknown endpoint", 404)
			return
		}
		f(s, a, w, r, user)
	})

	return m
}

// Handles the retrieval of a user token
func handleToken(
	s *System, a *auth.API, w http.ResponseWriter, r *http.Request, user string,
) {
	j := struct {
		Password pickyjson.Str
	}{
		Password: passwordParam.Required(),
	}
	if !apihelper.Prepare(w, r, &j, bodySizeLimit, "GET") {
		return
	}

	// login only succeeds without an error
	_, err := s.Login(user, j.Password.Str)
	if err != nil {
		common.HTTPError(w, r, err)
		return
	}

	token := a.NewUserToken(user)

	apihelper.JSONSuccess(w, &struct{ Token string }{token})
}

// Handles retrieving a user's information by their username
func handleGet(
	s *System, a *auth.API, w http.ResponseWriter, r *http.Request, user string,
) {
	if !apihelper.Prepare(w, r, nil, 0, "GET") {
		return
	}

	authUser := a.GetUser(r)
	var filter FieldFilter
	if user == authUser {
		filter |= Private
	}
	ret, err := s.Get(user, filter)

	if err != nil {
		common.HTTPError(w, r, err)
	} else {
		apihelper.JSONSuccess(w, &ret)
	}
}
