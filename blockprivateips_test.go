package remote

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBlockPrivateIPs_BlocksLoopback(t *testing.T) {
	// The httptest server listens on loopback; with BlockPrivateIPs(true) the
	// dialer must refuse to connect. Equivalent to using SafeClient.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("should never be reached"))
	}))
	t.Cleanup(server.Close)

	err := Get(server.URL).BlockPrivateIPs(true).Send()
	require.Error(t, err)
}

func TestBlockPrivateIPs_DefaultAllows(t *testing.T) {
	// The default (unset) permits any address, so a loopback server succeeds.
	body := []byte("default allows loopback")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	var result string
	require.Nil(t, Get(server.URL).Result(&result).Send())
	require.Equal(t, string(body), result)
}

func TestBlockPrivateIPs_FalseAllows(t *testing.T) {
	// Explicitly setting FALSE behaves like the default.
	body := []byte("explicitly allowed")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	var result string
	require.Nil(t, Get(server.URL).BlockPrivateIPs(false).Result(&result).Send())
	require.Equal(t, string(body), result)
}

func TestBlockPrivateIPs_PreservesCustomClient(t *testing.T) {
	// BlockPrivateIPs guards whatever client is configured: a custom client
	// still gets the loopback block.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("should never be reached"))
	}))
	t.Cleanup(server.Close)

	custom := &http.Client{Transport: &http.Transport{}}
	err := Get(server.URL).Client(custom).BlockPrivateIPs(true).Send()
	require.Error(t, err)
}
