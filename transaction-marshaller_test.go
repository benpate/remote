package remote

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarshalMap_Get(t *testing.T) {

	tx := Get("http://example.com").Query("a", "1").Header("X-Custom", "value")

	result := tx.MarshalMap()

	require.Equal(t, "GET", result["method"])
	require.Equal(t, "http://example.com", result["url"])
	require.Equal(t, "", result["body"]) // GET requests carry no body
}

func TestMarshalMap_PostBody(t *testing.T) {

	tx := Post("http://example.com").Body("payload")

	result := tx.MarshalMap()
	require.Equal(t, "POST", result["method"])
	require.Equal(t, "payload", result["body"])
}

func TestMarshalJSON_RoundTrip(t *testing.T) {

	tx := Post("http://example.com").Body("hello").Header("X-Test", "1")

	data, err := json.Marshal(tx)
	require.NoError(t, err)

	restored := New()
	require.NoError(t, json.Unmarshal(data, restored))

	require.Equal(t, "POST", restored.method)
	require.Equal(t, "http://example.com", restored.url)
	require.Equal(t, "hello", restored.body)
	require.Equal(t, "1", restored.header["X-Test"])
}

func TestUnmarshalJSON_Error(t *testing.T) {
	tx := New()
	require.Error(t, json.Unmarshal([]byte("not json"), tx))
}

func TestUnmarshalMap_Full(t *testing.T) {

	tx := New()
	err := tx.UnmarshalMap(map[string]any{
		"method": "PUT",
		"url":    "http://target.com",
		"body":   "the body",
		"header": map[string]string{"Accept": "text/plain"},
		"date":   "2026-01-02",
	})

	require.NoError(t, err)
	require.Equal(t, "PUT", tx.method)
	require.Equal(t, "http://target.com", tx.url)
	require.Equal(t, "the body", tx.body)
	require.Equal(t, "text/plain", tx.header["Accept"])
	require.Equal(t, "2026-01-02", tx.header["Date"])
}
