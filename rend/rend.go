// Package rend is a small utility package for breaking down a URL string into
// it's parts cleanly, as you would need to do for a REST interface
package rend

import (
	"net/url"
	"strings"
)

// URL takes in a URL and rends (splits) the URL's path into all slash separated
// parts, assigning each part to a given pointer's value. If any pointers are
// nil their corresponding part will not be assigned. If there are fewer
// pointers than parts the remaining parts (concatonated by slashes) will be
// returned.
//
//	u, _ := url.Parse("http://host.com/foo/bar/baz/buz")
//	var a, b string
//	rest := rend.URL(u, &a, &b)
//	// a    -> "foo"
//	// b    -> "bar"
//	// rest -> "baz/buz"
//
func URL(u *url.URL, receivers ...*string) string {
	urlPath := u.Path
	if urlPath[0] == '/' {
		urlPath = urlPath[1:]
	}

	parts := strings.SplitN(urlPath, "/", len(receivers)+1)
	if lp := len(parts); len(receivers) > lp {
		receivers = receivers[:lp]
	}

	for i := range receivers {
		if receivers[i] == nil {
			continue
		}
		*(receivers[i]) = parts[i]
	}

	if len(parts) > len(receivers) {
		return parts[len(parts)-1]
	}

	return ""
}
