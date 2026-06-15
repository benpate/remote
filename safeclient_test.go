package remote

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultClient_Timeout(t *testing.T) {
	require.Equal(t, time.Minute, DefaultClient().Timeout)
}

func TestSafeClient_Timeout(t *testing.T) {
	require.Equal(t, time.Minute, SafeClient().Timeout)
}

func TestSafeClient_BlocksInternalAddress(t *testing.T) {
	// An httptest server listens on loopback; SafeClient's dialer must refuse to
	// connect to it. This is the core SSRF protection, with no external network.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("should never be reached"))
	}))
	t.Cleanup(server.Close)

	err := Get(server.URL).Client(SafeClient()).Send()
	require.Error(t, err)
}

func TestSafeClient_AllowsPublicAddress(t *testing.T) {
	// With a client that permits loopback, the same request succeeds — proving
	// the block comes from the IP policy, not from the client being broken.
	body := []byte("hello from a local server")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	var result string
	err := Get(server.URL).
		Client(newSafeClient(func(net.IP) bool { return true })).
		Result(&result).
		Send()

	require.Nil(t, err)
	require.Equal(t, string(body), result)
}
