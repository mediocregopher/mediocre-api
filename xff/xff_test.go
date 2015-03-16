package xff

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	. "testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func echoRemoteAddr(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, r.RemoteAddr)
}

var testXFF = NewXFF(http.HandlerFunc(echoRemoteAddr))

func testAddr(t *T, addrExpect, addrIn string, forwards ...string) {
	r, err := http.NewRequest("GET", "/", nil)
	require.Nil(t, err)
	r.RemoteAddr = addrIn
	if len(forwards) > 0 {
		r.Header.Set("X-Forwarded-For", strings.Join(forwards, ", "))
	}
	w := httptest.NewRecorder()
	testXFF.ServeHTTP(w, r)
	assert.Equal(t, addrExpect, w.Body.String(), "addrIn: %q forwards: %v", addrIn, forwards)
}

func TestXFF(t *T) {
	// Some basic sanity checks
	testAddr(t, "8.8.8.8:2000", "8.8.8.8:2000")
	testAddr(t, "[::ffff:8.8.8.8]:2000", "[::ffff:8.8.8.8]:2000")

	// IPv4
	testAddr(t, "8.8.8.8:2000", "1.1.1.1:2000",
		"8.8.8.8")
	testAddr(t, "1.1.1.1:2000", "1.1.1.1:2000",
		"127.0.0.1")
	testAddr(t, "1.1.1.1:2000", "1.1.1.1:2000",
		"127.0.0.1", "192.168.1.1")
	testAddr(t, "8.8.8.8:2000", "1.1.1.1:2000",
		"127.0.0.1", "192.168.1.1", "8.8.8.8")
	testAddr(t, "8.8.8.8:2000", "1.1.1.1:2000",
		"127.0.0.1", "192.168.1.1", "8.8.8.8", "9.9.9.9")

	// IPv6
	testAddr(t, "[1::1]:2000", "1.1.1.1:2000",
		"1::1")
	testAddr(t, "8.8.8.8:2000", "1.1.1.1:2000",
		"::ffff:8.8.8.8")
	testAddr(t, "1.1.1.1:2000", "1.1.1.1:2000",
		"fd00::1")
	testAddr(t, "1.1.1.1:2000", "1.1.1.1:2000",
		"fd00::1", "::1")
	testAddr(t, "[1::1]:2000", "1.1.1.1:2000",
		"fd00::1", "::1", "1::1")
	testAddr(t, "[1::1]:2000", "1.1.1.1:2000",
		"fd00::1", "::1", "1::1", "2::2")
}
