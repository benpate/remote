package remote

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

// errStubDial is returned by the stub inner dialer; reaching it proves the guard
// delegated to (augmented) the inner dialer rather than connecting itself.
var errStubDial = errors.New("stub dial reached")

func TestGuardedDialContext_DelegatesForPublicIP(t *testing.T) {
	// A public IP literal passes the check and is handed to the inner dialer.
	var dialedAddress string
	inner := func(_ context.Context, _ string, address string) (net.Conn, error) {
		dialedAddress = address
		return nil, errStubDial
	}

	guard := guardedDialContext(inner)
	_, err := guard(context.Background(), "tcp", "8.8.8.8:443")

	require.ErrorIs(t, err, errStubDial) // reached the inner dialer
	require.Equal(t, "8.8.8.8:443", dialedAddress)
}

func TestGuardedDialContext_BlocksPrivateIP(t *testing.T) {
	// A non-public IP literal is rejected before the inner dialer is called.
	called := false
	inner := func(_ context.Context, _ string, _ string) (net.Conn, error) {
		called = true
		return nil, errStubDial
	}

	guard := guardedDialContext(inner)

	for _, address := range []string{"127.0.0.1:443", "10.0.0.1:80", "169.254.169.254:80", "[::1]:443"} {
		_, err := guard(context.Background(), "tcp", address)
		require.Error(t, err, "address=%s", address)
		require.False(t, called, "inner dialer must not be called for %s", address)
	}
}

func TestGuardedDialContext_PreservesInnerDialer(t *testing.T) {
	// The inner dialer's own behavior (here, a sentinel error) is preserved,
	// confirming the guard augments rather than replaces it.
	sentinel := errors.New("custom inner dialer ran")
	inner := func(_ context.Context, _ string, _ string) (net.Conn, error) {
		return nil, sentinel
	}

	guard := guardedDialContext(inner)
	_, err := guard(context.Background(), "tcp", "8.8.8.8:443")

	require.ErrorIs(t, err, sentinel)
}
