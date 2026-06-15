package remote

import (
	"net/http"
	"net/http/httptest"
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
	err := Get(server.URL).AllowHosts("127.0.0.1").Result(&result).Send()

	require.Nil(t, err)
	require.Equal(t, string(body), result)
}

func TestAllowHosts_RejectsUnlistedHost(t *testing.T) {
	// The request URL host is not in the allow-list, so Send fails before
	// contacting the server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

func TestAllowHosts_EmptyListAllowsAny(t *testing.T) {
	body := []byte("no restriction")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	var result string
	err := Get(server.URL).Result(&result).Send()

	require.Nil(t, err)
	require.Equal(t, string(body), result)
}
