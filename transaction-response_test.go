package remote

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/benpate/derp"
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

// TestDecodeResponseBody_MediaTypeIsCaseInsensitive confirms that a media type matches its decoder
// regardless of case, and of spaces around it.
func TestDecodeResponseBody_MediaTypeIsCaseInsensitive(t *testing.T) {

	for _, contentType := range []string{
		"Application/JSON",
		"APPLICATION/ACTIVITY+JSON; charset=utf-8",
		"application/json ; charset=utf-8",
		" application/ld+json",
	} {
		tx := newResponseTransaction(200, contentType, "")
		result := map[string]any{}
		require.NoError(t, tx.decodeResponseBody([]byte(`{"name":"value"}`), &result), contentType)
		require.Equal(t, "value", result["name"], contentType)
	}

	// XML matches the same way
	tx := newResponseTransaction(200, "Application/XML", "")
	result := struct {
		Name string `xml:"name"`
	}{}
	require.NoError(t, tx.decodeResponseBody([]byte(`<doc><name>value</name></doc>`), &result))
	require.Equal(t, "value", result.Name)
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

// TestDecodeResponseBody_FailureOmitsResult confirms that a failed decode records only derp's
// capped excerpt of the body, even when the result has already absorbed the rest of it.
func TestDecodeResponseBody_FailureOmitsResult(t *testing.T) {

	// The marker sits past derp's 4KB excerpt, so it can only reach the error by another route
	const marker = "PAST-THE-EXCERPT"
	padding := strings.Repeat("x", 8*1024)

	// requireNoMarker checks the error as derp-mongo would store it, and that the excerpt survives
	requireNoMarker := func(name string, err error) {
		t.Helper()
		require.Error(t, err, name)

		serialized, marshalError := json.Marshal(err)
		require.NoError(t, marshalError, name)
		require.NotContains(t, string(serialized), marker, name)

		var httpError derp.HTTPError
		require.True(t, errors.As(err, &httpError), name)
		require.True(t, httpError.Response.BodyTruncated, name)
	}

	// A JSON type error: Unmarshal fills "name" before it rejects "followers"
	jsonBody := `{"name":"` + padding + marker + `","followers":"lots"}`
	actor := struct {
		Name      string `json:"name"`
		Followers int    `json:"followers"`
	}{}
	err := newResponseTransaction(200, ContentTypeJSON, jsonBody).decodeResponseBody([]byte(jsonBody), &actor)
	require.Contains(t, actor.Name, marker, "precondition: the result holds the body")
	requireNoMarker("json", err)

	// An XML syntax error late in the document: Unmarshal has already filled "title"
	xmlBody := `<feed><title>` + padding + marker + `</title><broken`
	feed := struct {
		Title string `xml:"title"`
	}{}
	err = newResponseTransaction(200, ContentTypeXML, xmlBody).decodeResponseBody([]byte(xmlBody), &feed)
	require.Contains(t, feed.Title, marker, "precondition: the result holds the body")
	requireNoMarker("xml", err)

	// A writer that fails after it has seen the whole body
	writer := &recordingWriter{}
	err = newResponseTransaction(200, ContentTypePlain, jsonBody).decodeResponseBody([]byte(jsonBody), writer)
	require.Contains(t, writer.Received, marker, "precondition: the writer holds the body")
	requireNoMarker("writer", err)
}

// recordingWriter keeps everything written to it, then fails the write
type recordingWriter struct {
	Received string // Every byte written so far, exported so that JSON would carry it
}

// Write records the bytes, then reports a failure
func (w *recordingWriter) Write(p []byte) (int, error) {
	w.Received += string(p)
	return 0, derp.Internal("remote.recordingWriter.Write", "Disk full")
}

func TestDecodeResponseBody_ByteSliceIsACopy(t *testing.T) {

	// The caller owns their result outright. Writing to it must not reach back
	// into the buffer that ResponseBody keeps for re-reads.
	tx := New()
	tx.response = &http.Response{
		Header: http.Header{},
		Body:   io.NopCloser(strings.NewReader("original")),
	}

	body, err := tx.ResponseBody()
	require.NoError(t, err)

	var result []byte
	require.NoError(t, tx.decodeResponseBody(body, &result))
	require.Equal(t, "original", string(result))

	// Scribble on the caller's copy. (copy, not an index, so that a nil result
	// would fail the assertion above rather than panicking here.)
	copy(result, "XXXXXXXX")

	reread, err := tx.ResponseBody()
	require.NoError(t, err)
	require.Equal(t, "original", string(reread))
}

func TestResponseBody_ReturnsACopy(t *testing.T) {

	// The caller owns what ResponseBody hands back. Writing to it must not change
	// what a later call returns.
	tx := New()
	tx.response = &http.Response{
		Header: http.Header{},
		Body:   io.NopCloser(strings.NewReader("original")),
	}

	first, err := tx.ResponseBody()
	require.NoError(t, err)
	require.Equal(t, "original", string(first))

	copy(first, "XXXXXXXX")

	second, err := tx.ResponseBody()
	require.NoError(t, err)
	require.Equal(t, "original", string(second))
}

func TestResponseBodyReader_UnaffectedByCallerWrites(t *testing.T) {

	tx := New()
	tx.response = &http.Response{
		Header: http.Header{},
		Body:   io.NopCloser(strings.NewReader("original")),
	}

	body, err := tx.ResponseBody()
	require.NoError(t, err)
	copy(body, "XXXXXXXX")

	reread, err := io.ReadAll(tx.ResponseBodyReader())
	require.NoError(t, err)
	require.Equal(t, "original", string(reread))
}
