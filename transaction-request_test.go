package remote

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestURL_NoQuery(t *testing.T) {
	tx := Get("http://example.com/path")
	require.Equal(t, "http://example.com/path", tx.RequestURL())
}

func TestRequestURL_WithQuery(t *testing.T) {
	tx := Get("http://example.com/path").Query("a", "1").Query("b", "2")
	require.Equal(t, "http://example.com/path?a=1&b=2", tx.RequestURL())
}

func TestRequestBody_String(t *testing.T) {
	tx := Post("http://example.com").Body("hello world")
	body, err := tx.RequestBody()
	require.NoError(t, err)
	require.Equal(t, []byte("hello world"), body)
}

func TestRequestBody_Bytes(t *testing.T) {
	tx := New()
	tx.body = []byte("raw bytes")
	body, err := tx.RequestBody()
	require.NoError(t, err)
	require.Equal(t, []byte("raw bytes"), body)
}

func TestRequestBody_Reader(t *testing.T) {
	tx := New()
	tx.body = strings.NewReader("from reader")
	body, err := tx.RequestBody()
	require.NoError(t, err)
	require.Equal(t, []byte("from reader"), body)
}

func TestRequestBody_EmptyPlain(t *testing.T) {
	// With no content type, the body is empty
	tx := New()
	body, err := tx.RequestBody()
	require.NoError(t, err)
	require.Equal(t, []byte{}, body)
}

func TestRequestBody_Form(t *testing.T) {
	tx := Post("http://example.com").Form("a", "1").Form("b", "2")
	body, err := tx.RequestBody()
	require.NoError(t, err)
	require.Equal(t, "a=1&b=2", string(body))
}

func TestRequestBody_JSON(t *testing.T) {
	tx := Post("http://example.com").JSON(map[string]int{"x": 1})
	body, err := tx.RequestBody()
	require.NoError(t, err)
	require.JSONEq(t, `{"x":1}`, string(body))
}

func TestRequestBody_JSONError(t *testing.T) {
	// A value that cannot be marshalled to JSON returns an error
	tx := Post("http://example.com").JSON(make(chan int))
	_, err := tx.RequestBody()
	require.Error(t, err)
}

func TestRequestBody_UnsupportedContentType(t *testing.T) {
	// XML content type is not handled by RequestBody's marshaller switch
	tx := Post("http://example.com").ContentType(ContentTypeXML)
	_, err := tx.RequestBody()
	require.Error(t, err)
}

func TestIsContentTypeEmpty(t *testing.T) {
	tx := New()
	require.True(t, tx.isContentTypeEmpty())

	tx.ContentType("application/json")
	require.False(t, tx.isContentTypeEmpty())
}

func TestBody_DoesNotOverrideContentType(t *testing.T) {
	// If a content type is already set, Body() should not override it
	tx := Post("http://example.com").ContentType(ContentTypeJSON).Body("text")
	require.Equal(t, ContentTypeJSON, tx.header[ContentType])
}
