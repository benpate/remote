package remote

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// TestUndecodableResponse_HTML confirms that a server answering 200 text/html to a request for
// an Actor produces a 502 whose message names the media type that arrived.
func TestUndecodableResponse_HTML(t *testing.T) {

	err := fetchIntoMap(serve(t, http.StatusOK, "text/html; charset=utf-8", "<h1>This domain is for sale</h1>"))
	require.Error(t, err)

	// RULE: The upstream answered with something unusable, which derp names 502.  A 500 would
	// place the fault in this client.
	require.Equal(t, http.StatusBadGateway, derp.ErrorCode(err))
	require.True(t, derp.IsBadGateway(err))
	require.False(t, derp.IsInternalServerError(err))

	// Callers can ask for this condition by name, instead of reconstructing it
	require.True(t, IsUndecodableResponse(err))

	// RULE: The explanation says what arrived.  The status line is not a reason.
	rootMessage := derp.RootMessage(err)
	require.Contains(t, rootMessage, "text/html")
	require.NotEqual(t, "200 OK", rootMessage)

	// The evidence survives: what the server sent is still in the chain
	var httpError derp.HTTPError
	require.True(t, errors.As(err, &httpError))
	require.Equal(t, http.StatusOK, httpError.Response.StatusCode)
	require.Contains(t, httpError.Response.Body, "This domain is for sale")
}

// TestUndecodableResponse_UnknownContentType confirms that a media type with no decoder produces
// the same 502, carrying the request and response that caused it.
func TestUndecodableResponse_UnknownContentType(t *testing.T) {

	err := fetchIntoMap(serve(t, http.StatusOK, "application/octet-stream", "binary"))
	require.Error(t, err)

	require.Equal(t, http.StatusBadGateway, derp.ErrorCode(err))
	require.True(t, IsUndecodableResponse(err))
	require.Contains(t, derp.RootMessage(err), "application/octet-stream")

	// The transaction is recorded on the error
	var httpError derp.HTTPError
	require.True(t, errors.As(err, &httpError))
	require.Equal(t, http.StatusOK, httpError.Response.StatusCode)
	require.NotEmpty(t, httpError.Request.URL)
}

// TestUndecodableResponse_NoContentType confirms that a server declaring no media type still
// produces a complete sentence.
func TestUndecodableResponse_NoContentType(t *testing.T) {

	err := fetchIntoMap(serve(t, http.StatusOK, "", "who knows"))
	require.Error(t, err)

	require.True(t, IsUndecodableResponse(err))

	// RULE: An empty media type must not read as "Server answered , which ..."
	rootMessage := derp.RootMessage(err)
	require.Contains(t, rootMessage, "no Content-Type")
	require.NotContains(t, rootMessage, "answered ,")
}

// TestUndecodableResponse_ContentTypeIsBounded confirms that the server's own header cannot
// lengthen the message past a fixed bound, or leave invalid UTF-8 in it.
func TestUndecodableResponse_ContentTypeIsBounded(t *testing.T) {

	// RULE: RootMessage feeds user-facing text and the triage signature, so an unbounded header
	// would let a hostile server create a new error class with every request.
	err := fetchIntoMap(serve(t, http.StatusOK, "application/"+strings.Repeat("x", 10_000), "body"))
	require.Error(t, err)
	require.True(t, IsUndecodableResponse(err))
	require.Less(t, len(derp.RootMessage(err)), contentTypeMaxLength+100)

	// RULE: The cut never lands mid-character.  A header of multi-byte runes straddling the limit
	// would otherwise leave invalid UTF-8 in a message bound for Mongo and a user-facing page.
	for offset := range 4 {
		multiByte := "application/" + strings.Repeat("x", offset) + strings.Repeat("日", 100)
		err := fetchIntoMap(serve(t, http.StatusOK, multiByte, "body"))
		require.True(t, utf8.ValidString(derp.RootMessage(err)), "offset %d produced invalid UTF-8", offset)
	}
}

// TestUndecodableResponse_BodyIsCappedNotCopied confirms that a response body reaches the error
// log once, as derp's capped copy, and never as an uncapped detail.
func TestUndecodableResponse_BodyIsCappedNotCopied(t *testing.T) {

	// A body copied into the details as a string and as a []byte (base64 in JSON) would reach
	// the log uncapped and about 2.3 times over, so the page here is far larger than the cap.
	const pageSize = 256 * 1024
	page := "<html>" + strings.Repeat("x", pageSize) + "</html>"

	for _, contentType := range []string{"text/html", "application/json", "application/xml"} {

		err := fetchIntoMap(serve(t, http.StatusOK, contentType, page))
		require.Error(t, err, contentType)

		// The error as derp-mongo would store it
		serialized, marshalError := json.Marshal(err)
		require.NoError(t, marshalError, contentType)

		// RULE: One capped copy is 4KB.  Allow generously for headers and structure, and fail on
		// anything that looks like the page got through.
		require.Less(t, len(serialized), 32*1024, "%s: %d bytes serialized for a %d byte page", contentType, len(serialized), len(page))

		// The capped evidence is still there, so nothing was lost
		var httpError derp.HTTPError
		require.True(t, errors.As(err, &httpError), contentType)
		require.True(t, httpError.Response.BodyTruncated, contentType)
		require.NotEmpty(t, httpError.Response.Body, contentType)
	}
}

// TestUndecodableResponse_OnlyThatCase confirms that IsUndecodableResponse and the 502 apply to
// an undecodable 2xx answer and to no other failure.
func TestUndecodableResponse_OnlyThatCase(t *testing.T) {

	// A 502 sent by the far end means the origin behind a proxy is down, and nothing arrived
	// to decode, so it is a Bad Gateway that IsUndecodableResponse rejects.
	realBadGateway := fetchIntoMap(serve(t, http.StatusBadGateway, "text/html", "origin unreachable"))
	require.True(t, derp.IsBadGateway(realBadGateway))
	require.False(t, IsUndecodableResponse(realBadGateway))

	// RULE: An ordinary status failure is untouched.  For a 404 the status line IS the reason.
	notFound := fetchIntoMap(serve(t, http.StatusNotFound, "text/html", "nope"))
	require.Equal(t, http.StatusNotFound, derp.ErrorCode(notFound))
	require.Equal(t, "404 Not Found", derp.RootMessage(notFound))
	require.False(t, IsUndecodableResponse(notFound))

	// RULE: Malformed JSON keeps its 500.  It carries a real parser error as its cause, and
	// whether it should also be a 502 is a separate decision.
	malformed := fetchIntoMap(serve(t, http.StatusOK, "application/json", "{not json"))
	require.Equal(t, http.StatusInternalServerError, derp.ErrorCode(malformed))
	require.False(t, IsUndecodableResponse(malformed))

	// No error at all, and a foreign error, are never this case
	require.False(t, IsUndecodableResponse(nil))
	require.False(t, IsUndecodableResponse(errors.New("dial tcp: lookup gone.example: no such host")))

	// A 200 that decodes cleanly is not an error in the first place
	require.NoError(t, fetchIntoMap(serve(t, http.StatusOK, "application/activity+json", `{"type":"Person"}`)))
}

/******************************************
 * Helpers
 ******************************************/

// fetchIntoMap requests a URL the way an ActivityPub client does: asking for JSON, into a map.
func fetchIntoMap(url string) error {
	result := map[string]any{}
	return Get(url).Accept(ContentTypeActivityPub).AllowPrivateIPs(true).Result(&result).Send()
}

// serve starts a loopback server that answers every request with one status, media type, and body.
// An empty contentType sends no Content-Type header at all.
func serve(t *testing.T, status int, contentType string, body string) string {

	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {

		// RULE: A nil entry suppresses net/http's content sniffing, which would otherwise
		// invent "text/html; charset=utf-8" for any body that looks like markup.
		w.Header()["Content-Type"] = nil

		if contentType != "" {
			w.Header().Set(ContentType, contentType)
		}

		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))

	t.Cleanup(server.Close)
	return server.URL
}
