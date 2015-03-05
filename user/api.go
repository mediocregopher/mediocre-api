package user

import (
	"net/http"

	"github.com/asaskevich/govalidator"
	"github.com/mediocregopher/mediocre-api/auth"
	"github.com/mediocregopher/mediocre-api/common"
	"github.com/mediocregopher/mediocre-api/common/apihelper"
	"github.com/mediocregopher/mediocre-api/pickyjson"
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
		if eerr, ok := err.(ExpectedErr); ok {
			http.Error(w, eerr.Error(), 400)
			return
		} else if err != nil {
			http.Error(w, "unknown server-side error", 500)
			return
		}
	})

	return a
}
