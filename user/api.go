package user

import (
	"log"
	"net/http"

	"github.com/asaskevich/govalidator"
	"github.com/mediocregopher/mediocre-api/auth"
	"github.com/mediocregopher/mediocre-api/common"
	"github.com/mediocregopher/mediocre-api/common/apihelper"
	"github.com/mediocregopher/mediocre-api/pickyjson"
	"github.com/mediocregopher/mediocre-api/rend"
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

// If err is an expected error return it to the client as a 400 error. Otherwise
// if it's not nil return an unknown server-side error
func finalErr(w http.ResponseWriter, r *http.Request, err error) {
	if eerr, ok := err.(ExpectedErr); ok {
		http.Error(w, eerr.Error(), 400)
		return
	} else if err != nil {
		log.Printf("%s %s -> %s", r.Method, r.URL, err)
		http.Error(w, "unknown server-side error", 500)
		return
	}
}

// NewMux returns a new http.Handler (in reality a http.ServeMux wrapped with an
// auth.API) which has the basic suite of user creation/modification endpoints
func NewMux(o *auth.APIOpts, c common.Cmder) http.Handler {
	m := http.NewServeMux()
	a := auth.NewAPI(m, o)
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
		finalErr(w, r, err)
	})

	// All other requests will match this, meaning one should be of the form
	// /<username>/action
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var user, action string
		remaining := rend.URL(r.URL, &user, &action)
		if remaining != "" || user == "" || action == "" {
			http.Error(w, "unknown endpoint", 404)
		}

		switch action {
		case "token":
			handleToken(s, a, w, r, user)
		}
	})

	return a
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
		finalErr(w, r, err)
		return
	}

	token := a.NewUserToken(user)

	apihelper.JSONSuccess(w, &struct{ Token string }{token})
}
