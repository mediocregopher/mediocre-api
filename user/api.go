package user

import (
	"net/http"

	"github.com/asaskevich/govalidator"
	"github.com/gorilla/mux"
	"github.com/mediocregopher/mediocre-api/common"
	"github.com/mediocregopher/mediocre-api/common/apihelper"
	"github.com/mediocregopher/mediocre-api/pickyjson"
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

// NewMux returns a new http.Handler (in reality a http.ServeMux) which has the
// basic suite of user creation/modification endpoints. See the package README
// for more information
func NewMux(c util.Cmder) http.Handler {
	m := mux.NewRouter()
	s := New(c)

	m.Methods("POST").Path("/new-user").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			j := struct {
				Username, Email, Password pickyjson.Str
			}{
				Username: usernameParam.Required(),
				Email:    emailParam.Required(),
				Password: passwordParam.Required(),
			}
			if !apihelper.Prepare(w, r, &j, bodySizeLimit) {
				return
			}

			err := s.Create(j.Username.Str, j.Email.Str, j.Password.Str)
			common.HTTPError(w, r, err)
		},
	)

	m.Methods("POST").Path("/{user}/auth").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			user := mux.Vars(r)["user"]

			j := struct {
				Password pickyjson.Str
			}{
				Password: passwordParam.Required(),
			}
			if !apihelper.Prepare(w, r, &j, bodySizeLimit) {
				return
			}

			// login only succeeds without an error
			if err := s.Authenticate(user, j.Password.Str); err != nil {
				common.HTTPError(w, r, err)
				return
			}
		},
	)

	m.Methods("GET").Path("/{user}").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			user := mux.Vars(r)["user"]

			authUser := r.FormValue("_asUser")
			var filter FieldFlag
			if user == authUser {
				filter |= Private
			}
			ret, err := s.Get(user, filter)

			if err != nil {
				common.HTTPError(w, r, err)
			} else {
				apihelper.JSONSuccess(w, &ret)
			}
		},
	)

	return m
}
