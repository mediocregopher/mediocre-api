package main

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/mediocregopher/lever"
	"github.com/mediocregopher/mediocre-api/auth"
	"github.com/mediocregopher/mediocre-api/common/apihelper"
	"github.com/mediocregopher/mediocre-api/fwd"
)

func main() {
	l := lever.New("shield", nil)
	l.Add(lever.Param{
		Name:        "--listen-addr",
		Description: "Address to listen for api requests on",
		Default:     ":8080",
	})
	l.Add(lever.Param{
		Name:        "--secret",
		Description: "Secret to validate api and user tokens with",
	})
	l.Add(lever.Param{
		Name:        "--user-api-addr",
		Description: "Address the user api is listening on, e.g. \"http://127.0.0.1:8081\". Leave blank to not forward user requests",
	})
	l.Parse()

	secret, _ := l.ParamStr("--secret")
	if secret == "" {
		log.Fatal("--secret is required")
	}
	addr, _ := l.ParamStr("--listen-addr")
	userAddr, _ := l.ParamStr("--user-api-addr")

	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, newShieldMux(secret, userAddr)))
}

func prefixStrip(prefix string) alice.Constructor {
	return func(h http.Handler) http.Handler {
		return http.StripPrefix(prefix, h)
	}
}

func fwdErrorHandler(r *http.Request, err error) {
	log.Print(err)
}

type shield struct {
	a *auth.API
	m *mux.Router
}

func newShieldMux(secret, userAddr string) *shield {
	a := auth.NewAPI()
	a.Secret = []byte(secret)
	a.UserAuthGetParam = "_asUser"

	m := mux.NewRouter()
	base := alice.New()

	m.Handle("/shield/token", base.Append(
		a.Wrapper(auth.IPRateLimited),
	).ThenFunc(
		func(w http.ResponseWriter, r *http.Request) {
			token := a.NewAPIToken()
			apihelper.JSONSuccess(w, &struct{ Token string }{token})
		},
	))

	authDefault := a.Wrapper(auth.Default)

	if userAddr != "" {
		i := strings.Index(userAddr, ":")
		if i < 0 {
			log.Fatalf("bad user address: %s", userAddr)
		}
		userAddrScheme := userAddr[:i]
		userAddrHost := userAddr[i+3:]
		userPrefixStrip := prefixStrip("/user")

		// We need to manually handle this part since the user api doesn't know
		// anything about user tokens. So we ask the user api if the auth was
		// successful and if so create a user token, instead of just forwarding
		// the request
		m.Methods("POST").Path("/user/{user}/auth").Handler(base.Append(
			authDefault,
			userPrefixStrip,
		).ThenFunc(
			func(w http.ResponseWriter, r *http.Request) {
				r.URL.Scheme = userAddrScheme
				r.URL.Host = userAddrHost
				resp, err := http.DefaultClient.Do(r)
				if err != nil {
					fwdErrorHandler(r, err)
					http.Error(w, "unexpected server-side error", 500)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != 200 {
					w.WriteHeader(resp.StatusCode)
					io.Copy(w, resp.Body)
				}

				u := mux.Vars(r)["user"]
				tok := a.NewUserToken(u)
				apihelper.JSONSuccess(w, &struct{ Token string }{Token: tok})
			},
		))

		m.PathPrefix("/user/").Handler(base.Append(
			authDefault,
			userPrefixStrip,
		).Then(
			fwd.Rel(userAddr, "/", fwdErrorHandler),
		))
	}

	return &shield{a: a, m: m}
}

func (s *shield) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.m.ServeHTTP(w, r)
}
