package remote

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBearCap_Valid(t *testing.T) {

	tx := Get("bear:?t=token123&u=http://target.com/path")
	require.NoError(t, tx.assembleBearCap())

	require.Equal(t, "http://target.com/path", tx.url)
	require.Equal(t, "Bearer token123", tx.header["Authorization"])
}

func TestBearCap_NotBearCap(t *testing.T) {

	// A normal URL is left untouched
	tx := Get("http://example.com")
	require.NoError(t, tx.assembleBearCap())
	require.Equal(t, "http://example.com", tx.url)
	require.Equal(t, "", tx.header["Authorization"])
}

func TestBearCap_MissingURL(t *testing.T) {
	tx := Get("bear:?t=token123")
	require.Error(t, tx.assembleBearCap())
}

func TestBearCap_MissingToken(t *testing.T) {
	tx := Get("bear:?u=http://target.com")
	require.Error(t, tx.assembleBearCap())
}

func TestBearCap_InvalidQuery(t *testing.T) {
	// A malformed query string fails to parse
	tx := Get("bear:?%zz")
	require.Error(t, tx.assembleBearCap())
}
