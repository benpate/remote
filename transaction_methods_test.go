package remote

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMethodAndURL(t *testing.T) {

	tx := New().Method("CUSTOM").URL("http://example.com/path")

	require.Equal(t, "CUSTOM", tx.method)
	require.Equal(t, "http://example.com/path", tx.url)
}

func TestClient(t *testing.T) {

	custom := &http.Client{}
	tx := New().Client(custom)

	require.Same(t, custom, tx.client)
}

func TestUserAgent(t *testing.T) {

	tx := Get("http://example.com").UserAgent("MyAgent/1.0")
	require.Equal(t, "MyAgent/1.0", tx.header[UserAgent])
}

func TestResultAndError(t *testing.T) {

	success := map[string]any{}
	failure := map[string]any{}

	tx := Get("http://example.com").Result(&success).Error(&failure)

	require.Equal(t, &success, tx.success)
	require.Equal(t, &failure, tx.failure)
}

func TestWith(t *testing.T) {

	option1 := Option{}
	option2 := Option{}

	tx := Get("http://example.com").With(option1).With(option2)
	require.Equal(t, 2, len(tx.options))
}

func TestChaining_ReturnsSameTransaction(t *testing.T) {

	// Every chaining method should return the same underlying transaction
	tx := New()
	require.Same(t, tx, tx.Method("GET"))
	require.Same(t, tx, tx.URL("http://x.com"))
	require.Same(t, tx, tx.Get("http://x.com"))
	require.Same(t, tx, tx.Post("http://x.com"))
	require.Same(t, tx, tx.Put("http://x.com"))
	require.Same(t, tx, tx.Patch("http://x.com"))
	require.Same(t, tx, tx.Delete("http://x.com"))
	require.Same(t, tx, tx.Header("a", "b"))
	require.Same(t, tx, tx.ContentType("text/plain"))
	require.Same(t, tx, tx.Query("a", "b"))
	require.Same(t, tx, tx.Form("a", "b"))
	require.Same(t, tx, tx.Body("x"))
	require.Same(t, tx, tx.Result(nil))
	require.Same(t, tx, tx.Error(nil))
}

func TestNew_Defaults(t *testing.T) {

	tx := New()

	require.NotNil(t, tx.client)
	require.NotNil(t, tx.header)
	require.NotNil(t, tx.query)
	require.NotNil(t, tx.form)
	require.NotNil(t, tx.options)
	require.Equal(t, "", tx.method)
	require.Equal(t, "", tx.url)
}

func TestDefaultClient(t *testing.T) {
	client := DefaultClient()
	require.NotNil(t, client)
	require.Greater(t, int64(client.Timeout), int64(0))
}

func TestOptions_Function(t *testing.T) {

	// Options filters a list of `any` values, keeping only remote.Option values
	opt1 := Option{}
	opt2 := Option{}

	result := Options(opt1, "not an option", 42, opt2)
	require.Equal(t, 2, len(result))
}

func TestOptions_Empty(t *testing.T) {
	result := Options()
	require.NotNil(t, result)
	require.Equal(t, 0, len(result))
}
