package remote

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// newResponseTransaction builds a Transaction with a synthetic http.Response
// whose body and Content-Type are set for testing the response accessors.
func newResponseTransaction(statusCode int, contentType string, body string) *Transaction {
	tx := New()
	header := http.Header{}
	if contentType != "" {
		header.Set(ContentType, contentType)
	}
	tx.response = &http.Response{
		StatusCode: statusCode,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	return tx
}

func TestResponse_Accessors(t *testing.T) {

	tx := newResponseTransaction(201, ContentTypeJSON, `{"a":1}`)

	require.NotNil(t, tx.Response())
	require.Equal(t, 201, tx.ResponseStatusCode())
	require.Equal(t, ContentTypeJSON, tx.ResponseContentType())
	require.Equal(t, ContentTypeJSON, tx.ResponseHeader().Get(ContentType))
}

func TestResponse_NilGuards(t *testing.T) {

	tx := New() // no response set

	require.Nil(t, tx.Response())
	require.Equal(t, 0, tx.ResponseStatusCode())
	require.Equal(t, "", tx.ResponseContentType())
	require.Equal(t, http.Header{}, tx.ResponseHeader())
	require.Equal(t, 0, tx.statusCode())
}

func TestResponseBody_NilResponse(t *testing.T) {
	tx := New()
	_, err := tx.ResponseBody()
	require.Error(t, err)
}

func TestResponseBody_Rereadable(t *testing.T) {

	tx := newResponseTransaction(200, ContentTypeJSON, "some body")

	// ResponseBody can be called multiple times because it replaces the reader
	body1, err := tx.ResponseBody()
	require.NoError(t, err)
	require.Equal(t, "some body", string(body1))

	body2, err := tx.ResponseBody()
	require.NoError(t, err)
	require.Equal(t, "some body", string(body2))
}

func TestResponseBodyReader(t *testing.T) {

	tx := newResponseTransaction(200, ContentTypePlain, "reader content")

	reader := tx.ResponseBodyReader()
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, "reader content", string(data))
}

func TestResponseBodyReader_NilResponse(t *testing.T) {

	tx := New() // no response -> reader is empty, not nil
	reader := tx.ResponseBodyReader()
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, 0, len(data))
}

// --- decodeResponseBody ---

func TestDecodeResponseBody_NilResult(t *testing.T) {
	tx := newResponseTransaction(200, ContentTypeJSON, `{"a":1}`)
	require.NoError(t, tx.decodeResponseBody([]byte(`{"a":1}`), nil))
}

func TestDecodeResponseBody_Writer(t *testing.T) {
	tx := newResponseTransaction(200, ContentTypePlain, "")
	var buffer bytes.Buffer
	require.NoError(t, tx.decodeResponseBody([]byte("written"), &buffer))
	require.Equal(t, "written", buffer.String())
}

func TestDecodeResponseBody_ByteSlice(t *testing.T) {
	tx := newResponseTransaction(200, ContentTypePlain, "")
	var result []byte
	require.NoError(t, tx.decodeResponseBody([]byte("bytes"), &result))
	require.Equal(t, []byte("bytes"), result)
}

func TestDecodeResponseBody_String(t *testing.T) {
	tx := newResponseTransaction(200, ContentTypePlain, "")
	var result string
	require.NoError(t, tx.decodeResponseBody([]byte("a string"), &result))
	require.Equal(t, "a string", result)
}

func TestDecodeResponseBody_JSON(t *testing.T) {
	tx := newResponseTransaction(200, ContentTypeJSON, "")
	result := map[string]any{}
	require.NoError(t, tx.decodeResponseBody([]byte(`{"name":"value"}`), &result))
	require.Equal(t, "value", result["name"])
}

func TestDecodeResponseBody_JSONWithCharset(t *testing.T) {
	// Content-Type suffixes like "; charset=utf-8" must be stripped
	tx := newResponseTransaction(200, "application/json; charset=utf-8", "")
	result := map[string]any{}
	require.NoError(t, tx.decodeResponseBody([]byte(`{"name":"value"}`), &result))
	require.Equal(t, "value", result["name"])
}

func TestDecodeResponseBody_JSONError(t *testing.T) {
	tx := newResponseTransaction(200, ContentTypeJSON, "")
	result := map[string]any{}
	require.Error(t, tx.decodeResponseBody([]byte("not json"), &result))
}

func TestDecodeResponseBody_XML(t *testing.T) {
	tx := newResponseTransaction(200, ContentTypeXML, "")

	type doc struct {
		Name string `xml:"name"`
	}
	result := doc{}
	require.NoError(t, tx.decodeResponseBody([]byte(`<doc><name>value</name></doc>`), &result))
	require.Equal(t, "value", result.Name)
}

func TestDecodeResponseBody_XMLError(t *testing.T) {
	tx := newResponseTransaction(200, ContentTypeXML, "")
	result := struct {
		Name string `xml:"name"`
	}{}
	require.Error(t, tx.decodeResponseBody([]byte("<unclosed>"), &result))
}

func TestDecodeResponseBody_HTMLIntoStruct(t *testing.T) {
	// HTML cannot be decoded into a struct (only io.Writer/*string/*[]byte)
	tx := newResponseTransaction(200, ContentTypeHTML, "")
	result := map[string]any{}
	require.Error(t, tx.decodeResponseBody([]byte("<html></html>"), &result))
}

func TestDecodeResponseBody_UnsupportedContentType(t *testing.T) {
	tx := newResponseTransaction(200, "application/octet-stream", "")
	result := map[string]any{}
	require.Error(t, tx.decodeResponseBody([]byte("data"), &result))
}
