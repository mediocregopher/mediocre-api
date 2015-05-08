package main

import (
	"log"
	"net/http"

	"github.com/asaskevich/govalidator"
	"github.com/gorilla/mux"
	"github.com/mediocregopher/lever"
	"github.com/mediocregopher/mediocre-api/common"
	"github.com/mediocregopher/mediocre-api/common/apihelper"
	"github.com/mediocregopher/mediocre-api/pickyjson"
	"github.com/mediocregopher/mediocre-api/user"
	"github.com/mediocregopher/radix.v2/cluster"
	"github.com/mediocregopher/radix.v2/pool"
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

func main() {
	l := lever.New("shield", nil)
	l.Add(lever.Param{
		Name:        "--listen-addr",
		Description: "Address to listen for api requests on",
		Default:     ":8081",
	})
	l.Add(lever.Param{
		Name:        "--redis-addr",
		Description: "Address redis is listening on",
		Default:     "127.0.0.1:6379",
	})
	l.Add(lever.Param{
		Name:        "--redis-pool-size",
		Description: "Number of connections to make for each redis instance",
		Default:     "10",
	})
	l.Add(lever.Param{
		Name:        "--redis-cluster",
		Description: "Whether or not to treat the redis address as a node in a larger cluster",
		Flag:        true,
	})
	l.Parse()

	addr, _ := l.ParamStr("--listen-addr")
	redisAddr, _ := l.ParamStr("--redis-addr")
	redisPoolSize, _ := l.ParamInt("--redis-pool-size")
	redisCluster := l.ParamFlag("--redis-cluster")

	var cmder util.Cmder
	var err error
	if redisCluster {
		cmder, err = cluster.NewWithOpts(cluster.Opts{
			Addr:     redisAddr,
			PoolSize: redisPoolSize,
		})
	} else {
		cmder, err = pool.New("tcp", redisAddr, redisPoolSize)
	}
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, userMux(cmder)))
}

func requireAuthd(hf http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := mux.Vars(r)["user"]
		if r.FormValue("_asUser") != u {
			common.HTTPError(w, r, user.ErrBadAuth)
			return
		}
		hf(w, r)
	}
}

func userMux(cmder util.Cmder) http.Handler {
	m := mux.NewRouter()
	s := user.New(cmder)

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

	m.Methods("GET").Path("/{user}").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			u := mux.Vars(r)["user"]

			authU := r.FormValue("_asUser")
			var filter user.FieldFlag
			if u == authU {
				filter |= user.Private
			}
			ret, err := s.Get(u, filter)

			if err != nil {
				common.HTTPError(w, r, err)
			} else {
				apihelper.JSONSuccess(w, &ret)
			}
		},
	)

	m.Methods("POST").Path("/{user}").HandlerFunc(
		requireAuthd(
			func(w http.ResponseWriter, r *http.Request) {
				u := mux.Vars(r)["user"]

				j := user.Info{}
				if !apihelper.Prepare(w, r, &j, bodySizeLimit) {
					return
				}

				if err := s.Set(u, j); err != nil {
					common.HTTPError(w, r, err)
					return
				}
			},
		),
	)

	m.Methods("POST").Path("/{user}/password").HandlerFunc(
		requireAuthd(
			func(w http.ResponseWriter, r *http.Request) {
				user := mux.Vars(r)["user"]

				j := struct {
					OldPassword, NewPassword pickyjson.Str
				}{
					OldPassword: passwordParam.Required(),
					NewPassword: passwordParam.Required(),
				}
				if !apihelper.Prepare(w, r, &j, bodySizeLimit) {
					return
				}

				if err := s.Authenticate(user, j.OldPassword.Str); err != nil {
					common.HTTPError(w, r, err)
					return
				}

				if err := s.ChangePassword(user, j.NewPassword.Str); err != nil {
					common.HTTPError(w, r, err)
					return
				}
			},
		),
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

	return m
}
