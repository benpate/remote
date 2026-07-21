package remote

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewHTTPClient_UsesGuardedTransportByDefault(t *testing.T) {
	// With private IPs disallowed (the production default), the client is backed
	// by the shared SSRF-hardened transport.
	client := NewHTTPClient(false)
	require.True(t, client.Transport == safeTransport)
	require.Equal(t, time.Minute, client.Timeout)
	require.NotNil(t, client.CheckRedirect)
}

func TestNewHTTPClient_UsesDefaultTransportWhenPrivateAllowed(t *testing.T) {
	// When private IPs are allowed (dev / self-federation), the client uses the
	// plain transport so it can reach local addresses.
	client := NewHTTPClient(true)
	require.True(t, client.Transport == http.DefaultTransport)
}

func TestNewHTTPClient_BlocksPrivateAddress(t *testing.T) {
	// End-to-end: a client from NewHTTPClient(false) refuses to connect to a
	// non-public host, proving the guard rides along for non-remote consumers
	// (e.g. oauth2) that only accept an *http.Client.
	client := NewHTTPClient(false)

	for _, url := range []string{"http://127.0.0.1:8888/", "http://10.0.0.1/", "http://169.254.169.254/latest/meta-data/"} {
		response, err := client.Get(url) // nolint:bodyclose // request is blocked; there is no body
		require.Error(t, err, "url=%s", url)
		require.Nil(t, response, "url=%s", url)
	}
}

func TestNewHTTPClient_AllowsPublicAddress(t *testing.T) {
	// A public host passes the guard. The connection may still fail for network
	// reasons, but it must NOT be rejected by the private-IP guard.
	client := NewHTTPClient(false)

	response, err := client.Get("https://example.com/")

	if err != nil {
		require.NotContains(t, err.Error(), "non-public address", "public host must not be blocked by the guard")
		return
	}

	require.NotNil(t, response)

	if response == nil {
		return
	}

	defer func() { _ = response.Body.Close() }()
	require.NotZero(t, response.StatusCode)
}

func TestGuardedRedirect_CapsChain(t *testing.T) {
	// The standalone redirect policy rejects a chain that reaches maxRedirects.
	via := make([]*http.Request, maxRedirects)
	err := guardedRedirect(&http.Request{}, via)
	require.Error(t, err)

	// A shorter chain is allowed.
	require.NoError(t, guardedRedirect(&http.Request{}, make([]*http.Request, maxRedirects-1)))
}
