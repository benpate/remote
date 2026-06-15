package remote

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// sizedServer returns an httptest server that writes a body of n bytes.
func sizedServer(t *testing.T, n int) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(make([]byte, n))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestMaxResponseSize_DefaultIsOneGB(t *testing.T) {
	require.Equal(t, int64(1<<30), New().maxResponseSize)
}

func TestMaxResponseSize_Exceeded(t *testing.T) {
	server := sizedServer(t, 1000)

	var result []byte
	err := Get(server.URL).AllowPrivateIPs(true).MaxResponseSize(100).Result(&result).Send()
	require.Error(t, err)
}

func TestMaxResponseSize_WithinLimit(t *testing.T) {
	server := sizedServer(t, 100)

	var result []byte
	err := Get(server.URL).AllowPrivateIPs(true).MaxResponseSize(1000).Result(&result).Send()
	require.Nil(t, err)
	require.Len(t, result, 100)
}

func TestMaxResponseSize_ExactBoundary(t *testing.T) {
	server := sizedServer(t, 100)

	// A body exactly at the limit is allowed.
	var result []byte
	err := Get(server.URL).AllowPrivateIPs(true).MaxResponseSize(100).Result(&result).Send()
	require.Nil(t, err)
	require.Len(t, result, 100)
}

func TestMaxResponseSize_ZeroRestoresDefault(t *testing.T) {
	server := sizedServer(t, 100)

	// A zero (or negative) limit falls back to the 1GB default, so a small body
	// is read normally.
	var result []byte
	err := Get(server.URL).AllowPrivateIPs(true).MaxResponseSize(0).Result(&result).Send()
	require.Nil(t, err)
	require.Len(t, result, 100)
}
