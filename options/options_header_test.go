package options

import (
	"testing"

	"github.com/benpate/remote"
	"github.com/stretchr/testify/require"
)

// applyBeforeRequest runs an option's BeforeRequest hook against a fresh
// transaction and returns the resulting header map.
func applyBeforeRequest(t *testing.T, option remote.Option) map[string]string {
	t.Helper()

	txn := remote.New()
	require.NotNil(t, option.BeforeRequest)
	require.NoError(t, option.BeforeRequest(txn))

	header, ok := txn.MarshalMap()["header"].(map[string]string)
	require.True(t, ok)
	return header
}

func TestAccept(t *testing.T) {
	header := applyBeforeRequest(t, Accept("application/json"))
	require.Equal(t, "application/json", header["Accept"])
}

func TestAuthorization(t *testing.T) {
	header := applyBeforeRequest(t, Authorization("Bearer xyz"))
	require.Equal(t, "Bearer xyz", header["Authorization"])
}

func TestBasicAuth(t *testing.T) {
	header := applyBeforeRequest(t, BasicAuth("user", "pass"))
	// base64("user:pass") == "dXNlcjpwYXNz"
	require.Equal(t, "Basic dXNlcjpwYXNz", header["Authorization"])
}

func TestBearerAuth(t *testing.T) {
	header := applyBeforeRequest(t, BearerAuth("token123"))
	require.Equal(t, "Bearer token123", header["Authorization"])
}

func TestUserAgent(t *testing.T) {
	header := applyBeforeRequest(t, UserAgent("MyAgent/2.0"))
	require.Equal(t, "MyAgent/2.0", header["User-Agent"])
}
