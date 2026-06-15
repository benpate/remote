package remote

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildClient_Timeout(t *testing.T) {
	require.Equal(t, time.Minute, New().buildClient().Timeout)
}

func TestBuildClient_WrapsGuardedBaseByDefault(t *testing.T) {
	// By default, the middleware receives the SSRF-hardened base transport as
	// "next" — so the guard always sits underneath any caller middleware.
	var gotNext http.RoundTripper

	New().WithRoundTripper(func(next http.RoundTripper) http.RoundTripper {
		gotNext = next
		return next
	}).buildClient()

	require.True(t, gotNext == safeTransport)
}

func TestBuildClient_WrapsUnguardedBaseWhenAllowed(t *testing.T) {
	// When private IPs are allowed, the base handed to middleware is the plain
	// (unguarded) shared transport.
	var gotNext http.RoundTripper

	New().AllowPrivateIPs(true).WithRoundTripper(func(next http.RoundTripper) http.RoundTripper {
		gotNext = next
		return next
	}).buildClient()

	require.True(t, gotNext == http.DefaultTransport)
}

func TestBuildClient_NoMiddlewareUsesBaseDirectly(t *testing.T) {
	require.True(t, New().buildClient().Transport == safeTransport)
}
