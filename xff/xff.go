// Package xff implements a middleware http.Handler which parses any
// X-Forwarded-For headers it sees in the *http.Request and sets the RemoteAddr
// to the correct value based on them
//
//	s := http.NewServeMux()
//	// Set Handle and HandleFuncs here
//	x := xff.NewXFF(s)
//	// All *http.Request instances in the handlers for s will have the correct
//	// RemoteAddr field value now
//	http.ListenAndServe(":8080", x)
//
package xff

import (
	"net"
	"net/http"
	"strings"
)

// Unfortunately go doesn't provide a way to distinguish a v4 net.IP from a v6,
// so we have to just try all private CIDRs no matter the type
var internalCIDRs = []*net.IPNet{
	mustGetCIDRNetwork("10.0.0.0/8"),
	mustGetCIDRNetwork("172.16.0.0/12"),
	mustGetCIDRNetwork("192.168.0.0/16"),
	mustGetCIDRNetwork("169.254.0.0/16"),
	mustGetCIDRNetwork("169.254.0.0/16"),
	mustGetCIDRNetwork("fd00::/8"),
}

func ipIsPrivate(ip net.IP) bool {
	for _, cidr := range internalCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// XFF implements the https.Handler interface. It will parse X-Forwarded-For
// headers in any request it sees, fixing the RemoteAddr accordingly, and then
// pass off the request to the handler it was initialized with
type XFF struct {
	h http.Handler
}

// NewXFF intializes an XFF which can be then passed off to anything which
// acceps an http.Handler, such as http.ListenAndServe
func NewXFF(h http.Handler) *XFF {
	return &XFF{h}
}

// ServeHTTP implements the http.Handler interface
func (x *XFF) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer x.h.ServeHTTP(w, r)
	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		return
	}

	ips := strings.Split(xff, ",")
	finalIP := ""
	for i := range ips {
		ip := parseIP(ips[i])
		if ip == nil || ip.IsLoopback() || ipIsPrivate(ip) {
			continue
		}
		finalIP = ip.String()
		break
	}

	if finalIP == "" {
		return
	}
	finalNeedsBrackets := (strings.Index(finalIP, ":") >= 0)

	port := r.RemoteAddr[strings.LastIndex(r.RemoteAddr, ":")+1:]
	if finalNeedsBrackets {
		r.RemoteAddr = "[" + finalIP + "]:" + port
	} else {
		r.RemoteAddr = finalIP + ":" + port
	}
}

// Used because we may want to strip the brackets from an input ip, if there are
// any
func parseIP(ipRaw string) net.IP {
	ipRaw = strings.TrimSpace(ipRaw)
	if len(ipRaw) == 0 {
		return nil
	}

	if ipRaw[0] == '[' {
		ipRaw = ipRaw[1 : len(ipRaw)-1]
	}
	return net.ParseIP(ipRaw)
}

func mustGetCIDRNetwork(cidr string) *net.IPNet {
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	return n
}
