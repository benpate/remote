package remote

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllowPrivateIPs_BlockedByDefault(t *testing.T) {
	// The httptest server listens on loopback; by default the dialer must refuse
	// to connect to it (no opt-in required).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("should never be reached"))
	}))
	t.Cleanup(server.Close)

	err := Get(server.URL).Send()
	require.Error(t, err)
}

func TestAllowPrivateIPs_True_Allows(t *testing.T) {
	// Explicitly allowing private IPs lets a loopback request through.
	body := []byte("allowed")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	var result string
	require.Nil(t, Get(server.URL).AllowPrivateIPs(true).Result(&result).Send())
	require.Equal(t, string(body), result)
}

func TestAllowPrivateIPs_False_Blocks(t *testing.T) {
	// Explicitly setting FALSE behaves like the default (blocked).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("should never be reached"))
	}))
	t.Cleanup(server.Close)

	err := Get(server.URL).AllowPrivateIPs(false).Send()
	require.Error(t, err)
}

func TestAllowPrivateIPs_GuardsCustomClientByDefault(t *testing.T) {
	// The default guard applies even to a custom client.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("should never be reached"))
	}))
	t.Cleanup(server.Close)

	custom := &http.Client{Transport: &http.Transport{}}
	err := Get(server.URL).Client(custom).Send()
	require.Error(t, err)
}
