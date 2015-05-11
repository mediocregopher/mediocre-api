package fwd

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	. "testing"

	"github.com/stretchr/testify/assert"
)

var testHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Method", r.Method)
	xpath := r.URL.Path
	if r.URL.RawQuery != "" {
		xpath += "?" + r.URL.RawQuery
	}
	w.Header().Set("X-Path", xpath)
	w.Header().Set("X-Whatever", "foo")
	io.Copy(w, r.Body)
})

var testServer, testURL = func() (*httptest.Server, *url.URL) {
	s := httptest.NewServer(testHandler)
	u, err := url.Parse(s.URL + "/endpoint")
	if err != nil {
		panic(err)
	}
	return s, u
}()

func testErrHandler(r *http.Request, err error) {
	panic(err)
}

func TestAbs(t *T) {
	// Our handler sets the X-Whatever header on the response and then passes
	// off to Abs, which also ends up adding to X-Whatever. This will confirm
	// that headers are added to and not overwritten
	//handler := Abs(testURL.String(), testErrHandler)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Whatever", "bar")
		Abs(testURL.String(), testErrHandler).ServeHTTP(w, r)
	})

	body := bytes.NewBufferString("OHAI")
	req, err := http.NewRequest("GET", "http://example.com/doesntmatter", body)
	assert.Nil(t, err)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, []string{"GET"}, w.HeaderMap["X-Method"])
	assert.Equal(t, []string{testURL.Path}, w.HeaderMap["X-Path"])
	assert.Equal(t, []string{"bar", "foo"}, w.HeaderMap["X-Whatever"])
	assert.Equal(t, "OHAI", w.Body.String())
}

func TestRel(t *T) {
	// Our handler sets the X-Whatever header on the response and then passes
	// off to Rel, which also ends up adding to X-Whatever. This will confirm
	// that headers are added to and not overwritten
	//handler := Abs(testURL.String(), testErrHandler)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Whatever", "bar")
		endpoint := testURL.Scheme + "://" + testURL.Host
		Rel(endpoint, "/rel", testErrHandler).ServeHTTP(w, r)
	})

	body := bytes.NewBufferString("OHAI")
	req, err := http.NewRequest("GET", "http://example.com/somepath", body)
	assert.Nil(t, err)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, []string{"GET"}, w.HeaderMap["X-Method"])
	assert.Equal(t, []string{"/rel/somepath"}, w.HeaderMap["X-Path"])
	assert.Equal(t, []string{"bar", "foo"}, w.HeaderMap["X-Whatever"])
	assert.Equal(t, "OHAI", w.Body.String())

	// Make sure the request gets its GET arguments passed along
	body = bytes.NewBufferString("OHAI")
	req, err = http.NewRequest("GET", "http://example.com/somepath?foo=bar", body)
	assert.Nil(t, err)

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, []string{"GET"}, w.HeaderMap["X-Method"])
	assert.Equal(t, []string{"/rel/somepath?foo=bar"}, w.HeaderMap["X-Path"])
	assert.Equal(t, []string{"bar", "foo"}, w.HeaderMap["X-Whatever"])
	assert.Equal(t, "OHAI", w.Body.String())
}
