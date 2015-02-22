package rend

import (
	"net/url"
	. "testing"

	"github.com/stretchr/testify/assert"
)

func mustURL(path string) *url.URL {
	return &url.URL{
		Path: path,
	}
}

func TestURL(t *T) {
	{
		var a, b string
		rest := URL(mustURL("/foo/bar"), &a, &b)
		assert.Equal(t, "foo", a)
		assert.Equal(t, "bar", b)
		assert.Equal(t, "", rest)
	}
	{
		var a, b string
		rest := URL(mustURL("foo/bar"), &a, &b)
		assert.Equal(t, "foo", a)
		assert.Equal(t, "bar", b)
		assert.Equal(t, "", rest)
	}
	{
		var b string
		rest := URL(mustURL("/foo/bar"), nil, &b)
		assert.Equal(t, "bar", b)
		assert.Equal(t, "", rest)
	}
	{
		var a, b string
		rest := URL(mustURL("foo/bar/baz/buz"), &a, &b)
		assert.Equal(t, "foo", a)
		assert.Equal(t, "bar", b)
		assert.Equal(t, "baz/buz", rest)
	}
	{
		var a, b, c string
		rest := URL(mustURL("foo/bar"), &a, &b, &c)
		assert.Equal(t, "foo", a)
		assert.Equal(t, "bar", b)
		assert.Equal(t, "", c)
		assert.Equal(t, "", rest)
	}
}
