package main

import (
	"log"
	"net/http"

	"github.com/mediocregopher/lever"
	"github.com/mediocregopher/mediocre-api/auth"
	"github.com/mediocregopher/mediocre-api/common/apihelper"
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
	l.Parse()

	secret, _ := l.ParamStr("--secret")
	if secret == "" {
		log.Fatal("--secret is required")
	}
	addr, _ := l.ParamStr("--listen-addr")

	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, shieldMux(secret)))
}

func shieldMux(secret string) http.Handler {
	a := auth.NewAPI()
	a.Secret = []byte(secret)

	mux := http.NewServeMux()

	mux.Handle("/shield/token", a.WrapHandlerFunc(
		auth.IPRateLimited,
		func(w http.ResponseWriter, r *http.Request) {
			token := a.NewAPIToken()
			apihelper.JSONSuccess(w, &struct{ Token string }{token})
		},
	))

	return mux
}
