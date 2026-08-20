package remote

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllowHosts_PermitsListedHost(t *testing.T) {
	// The httptest server listens on 127.0.0.1; allow-listing that host lets the
	// request through (the default client permits loopback).
	body := []byte("allowed")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	var result string
	err := Get(server.URL).AllowPrivateIPs(true).AllowHosts("127.0.0.1").Result(&result).Send()

	require.Nil(t, err)
	require.Equal(t, string(body), result)
}

func TestAllowHosts_RejectsUnlistedHost(t *testing.T) {
	// The request URL host is not in the allow-list, so Send fails before
	// contacting the server.
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("server should never be contacted")
	}))
	t.Cleanup(server.Close)

	err := Get(server.URL).AllowHosts("example.com").Send()
	require.Error(t, err)
}

func TestAllowHosts_CaseInsensitive(t *testing.T) {
	// Matching folds case on both the URL host and the allow-list entries.
	require.NoError(t, Get("https://Example.COM/path").AllowHosts("example.com").validateAllowedHosts())
	require.NoError(t, Get("https://example.com/path").AllowHosts("Example.Com").validateAllowedHosts())

	// A host that is not on the list is rejected.
	require.Error(t, Get("https://other.com/path").AllowHosts("example.com").validateAllowedHosts())
}

func TestAllowHosts_RejectsRedirectToUnlistedHost(t *testing.T) {
	// A redirect target whose host is not in the allow-list must be blocked, even
	// when the initial (allow-listed) host issues the redirect. The redirect points
	// at "localhost" (which also resolves to loopback) while only "127.0.0.1" is
	// allow-listed, so this exercises the allow-list rather than the IP guard.
	redirectFollowed := false
	trusted := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirected" {
			redirectFollowed = true
			return
		}
		_, port, _ := net.SplitHostPort(r.Host)
		http.Redirect(w, r, "http://localhost:"+port+"/redirected", http.StatusFound)
	}))
	t.Cleanup(trusted.Close)

	err := Get(trusted.URL).AllowPrivateIPs(true).AllowHosts("127.0.0.1").Send()
	require.Error(t, err)
	require.False(t, redirectFollowed, "request must not follow a redirect to an unlisted host")
}

func TestAllowHosts_EmptyListAllowsAny(t *testing.T) {
	body := []byte("no restriction")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	var result string
	err := Get(server.URL).AllowPrivateIPs(true).Result(&result).Send()

	require.Nil(t, err)
	require.Equal(t, string(body), result)
}

func TestAllowHosts_BareEntryMatchesAnyPort(t *testing.T) {
	// An entry with no port places no restriction on the port.
	require.NoError(t, Get("https://example.com:8443/path").AllowHosts("example.com").validateAllowedHosts())
	require.NoError(t, Get("https://example.com/path").AllowHosts("example.com").validateAllowedHosts())
}

func TestAllowHosts_EntryWithPortMatchesExactly(t *testing.T) {
	// An entry that names a port matches that port, and only that port.
	require.NoError(t, Get("https://example.com:8443/path").AllowHosts("example.com:8443").validateAllowedHosts())
	require.Error(t, Get("https://example.com:9999/path").AllowHosts("example.com:8443").validateAllowedHosts())
	require.Error(t, Get("https://example.com/path").AllowHosts("example.com:8443").validateAllowedHosts())
}

func TestAllowHosts_EntryWithDefaultPort(t *testing.T) {
	// A URL that omits the scheme's default port still matches an entry that
	// spells it out, since the two name the same endpoint.
	require.NoError(t, Get("https://example.com/path").AllowHosts("example.com:443").validateAllowedHosts())
	require.NoError(t, Get("http://example.com/path").AllowHosts("example.com:80").validateAllowedHosts())

	// ...but the ports still have to agree.
	require.Error(t, Get("http://example.com/path").AllowHosts("example.com:443").validateAllowedHosts())
}

func TestAllowHosts_PortMatchingIsCaseInsensitive(t *testing.T) {
	require.NoError(t, Get("https://Example.COM:8443/path").AllowHosts("EXAMPLE.com:8443").validateAllowedHosts())
}

func TestAllowHosts_IPv6WithPort(t *testing.T) {
	// An IPv6 entry is bracketed, exactly as it appears in the URL.
	require.NoError(t, Get("http://[::1]:8080/path").AllowHosts("[::1]:8080").validateAllowedHosts())
	require.Error(t, Get("http://[::1]:9999/path").AllowHosts("[::1]:8080").validateAllowedHosts())

	// A bare IPv6 entry is unbracketed, because that is what URL.Hostname returns.
	require.NoError(t, Get("http://[::1]:8080/path").AllowHosts("::1").validateAllowedHosts())
}

func TestAllowHosts_UnknownSchemeWithPortEntry(t *testing.T) {
	// A scheme with no default port cannot satisfy a port-qualified entry.
	require.Error(t, Get("ftp://example.com/path").AllowHosts("example.com:21").validateAllowedHosts())

	// A bare entry still matches, since it does not ask about the port.
	require.NoError(t, Get("ftp://example.com/path").AllowHosts("example.com").validateAllowedHosts())
}

func TestAllowHosts_RejectsRedirectToDifferentPort(t *testing.T) {
	// The allow-list is re-checked on every hop, ports included.
	redirectFollowed := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirected" {
			redirectFollowed = true
			return
		}
		http.Redirect(w, r, "http://127.0.0.1:1/redirected", http.StatusFound)
	}))
	t.Cleanup(server.Close)

	_, port, _ := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))

	err := Get(server.URL).AllowPrivateIPs(true).AllowHosts("127.0.0.1:" + port).Send()
	require.Error(t, err)
	require.False(t, redirectFollowed, "request must not follow a redirect to a different port")
}
