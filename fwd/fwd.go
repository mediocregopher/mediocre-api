// Package fwd implements forwarding requests from one http endpoint to another.
// It's useful when you have some kind of internal api which you wish to expose
// to the world, but you want to place some kind of proxy in front which will
// handle rate-limiting and authentication and such.
//
//	// Forwards a request to another endpoint
//	http.Handle("/foo", fwd.Abs("http://127.0.0.1:8081/other/foo", nil))
//
//	// Forwards a request to another endpoint, appending the URL.Path of the
//	// original to be relative to this one. e.g. "/fuz/fiz" becomes
//	// "http://127.0.0.1:8080/rel/fuz/fiz"
//	http.Handle("/fuz", fwd.Rel("http://127.0.0.1:8081", "/rel", nil))
//
package fwd

import (
	"io"
	"net/http"
	"net/url"
	"path"
)

// Abs returns an http.Handler which receives any incoming requests and
// re-performs them exactly as-is, except on the given URL instead.
//
// errHandler can be passed in to handle any network errors which may occur.
//
// This function panics if absURL cannot be parsed by url.Parse
func Abs(absURL string, errHandler func(*http.Request, error)) http.Handler {
	parsedURL, err := url.Parse(absURL)
	if err != nil {
		panic(err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		doProxy(parsedURL, w, r, errHandler)
	})
}

// Rel returns an http.Handler which receives any incoming requests and
// re-performs them exactly as-is, except it rebases the request's path onto the
// given address and path.
//
// For example, the handler returned by Rel("http://foo.com", "/proxied", nil)
// will receive a request for "http://bar.com/baz" and re-perform it on
// "http://foo.com/proxied/baz"
//
// errHandler can be passed in to handle any network or url parsing errors which
// may occur.
func Rel(
	addr, relPath string, errHandler func(*http.Request, error),
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		absURL := addr + path.Join(relPath, r.URL.Path)
		parsedURL, err := url.Parse(absURL)
		if err != nil {
			errHandler(r, err)
		}

		doProxy(parsedURL, w, r, errHandler)
	})
}

func doProxy(
	u *url.URL, w http.ResponseWriter, r *http.Request,
	errHandler func(*http.Request, error),
) {
	r.URL = u
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		if errHandler != nil {
			errHandler(r, err)
		}
		http.Error(w, "unexpected server-side error", 500)
	}
	defer resp.Body.Close()

	for header, vals := range resp.Header {
		w.Header()[header] = append(w.Header()[header], vals...)
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
