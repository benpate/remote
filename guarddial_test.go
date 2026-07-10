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

func TestFilterPublicIPs(t *testing.T) {

	ipAddrs := func(values ...string) []net.IPAddr {
		result := make([]net.IPAddr, 0, len(values))
		for _, value := range values {
			result = append(result, net.IPAddr{IP: net.ParseIP(value)})
		}
		return result
	}

	ips := func(values ...string) []net.IP {
		result := make([]net.IP, 0, len(values))
		for _, value := range values {
			result = append(result, net.ParseIP(value))
		}
		return result
	}

	// The bug this fixes: a host that resolves to a mix of public and non-public
	// addresses keeps only the public ones instead of rejecting the whole host.
	// "fe80::216:3eff:fe2c:7747" is the leaked IPv6 link-local address from the
	// real-world report.
	t.Run("drops non-public, keeps public", func(t *testing.T) {
		result := filterPublicIPs(ipAddrs("93.184.216.34", "fe80::216:3eff:fe2c:7747"))
		require.Equal(t, ips("93.184.216.34"), result)
	})

	// All-public addresses are all retained, order preserved.
	t.Run("keeps all public", func(t *testing.T) {
		result := filterPublicIPs(ipAddrs("8.8.8.8", "1.1.1.1", "2606:4700:4700::1111"))
		require.Equal(t, ips("8.8.8.8", "1.1.1.1", "2606:4700:4700::1111"), result)
	})

	// A host that resolves entirely to non-public space yields an empty slice
	// (the caller turns this into a "blocked" error).
	t.Run("all non-public yields empty", func(t *testing.T) {
		result := filterPublicIPs(ipAddrs("127.0.0.1", "10.0.0.1", "fe80::1", "::1"))
		require.Empty(t, result)
	})

	// No addresses in, empty (non-nil) slice out.
	t.Run("empty input", func(t *testing.T) {
		result := filterPublicIPs(nil)
		require.Empty(t, result)
		require.NotNil(t, result)
	})
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
