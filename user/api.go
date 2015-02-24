package user

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mediocregopher/mediocre-api/auth"
	"github.com/mediocregopher/mediocre-api/common"
)

func NewMux(o *auth.APIOpts, c common.Cmder) http.Handler {
	m := http.NewServeMux()
	a := auth.NewAPI(m, o)
	s := New(c)

	m.HandleFunc("/new-user", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(400)
			fmt.Fprintln(w, "must POST")
			return
		}

		j := struct {
			Username, Email, Password string
		}{}

		if err := json.NewDecoder(r.Body).Decode(&j); err != nil {
			w.WriteHeader(400)
			fmt.Fprintln(w, err)
			return
		}

		if j.Email == "" || len(j.Email) > 255 {
			w.WriteHeader(400)
			fmt.Fprintln(w, "invalid email")
			return
		} else if j.Password == "" || len(j.Password) > 255 {
			w.WriteHeader(400)
			fmt.Fprintln(w, "invalid password")
			return
		} else if j.Username == "" {
			j.Username = j.Email
		}

		err := s.Create(j.Username, j.Email, j.Password)
		if eerr, ok := err.(ExpectedErr); ok {
			w.WriteHeader(400)
			fmt.Fprintln(w, eerr)
			return
		} else if err != nil {
			w.WriteHeader(500)
			fmt.Fprintln(w, "unknown server-side error")
			return
		}
	})

	return a
}
