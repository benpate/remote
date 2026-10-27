package remote

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"math"
	"net/http"
	"strings"

	"github.com/benpate/derp"
)

// Response returns the original HTTP response object.
func (t *Transaction) Response() *http.Response {
	return t.response
}

// ResponseHeader returns the HTTP response header.
func (t *Transaction) ResponseHeader() http.Header {

	if t.response == nil {
		return http.Header{}
	}

	if t.response.Header == nil {
		return http.Header{}
	}

	return t.response.Header
}

// ResponseContentType returns the Content-Type header of the response.
func (t *Transaction) ResponseContentType() string {

	if t.response == nil {
		return ""
	}

	if t.response.Header == nil {
		return ""
	}

	return t.response.Header.Get(ContentType)
}

// ResponseStatusCode returns the HTTP status code of the response.
func (t *Transaction) ResponseStatusCode() int {

	if t.response == nil {
		return 0
	}

	return t.response.StatusCode
}

// defaultMaxResponseSize is the default cap (1GB) on how many bytes are read
// from a response body, to prevent an untrusted server from exhausting memory.
const defaultMaxResponseSize = 1 << 30

// ResponseBody returns the original response body, as a byte array. It replaces
// the body reader as it goes, so it can be called repeatedly without error. The
// result is a copy, which the caller owns outright.
func (t *Transaction) ResponseBody() ([]byte, error) {

	body, err := t.responseBody()

	if err != nil {
		return []byte{}, err
	}

	// Copy, so that a caller writing to the result cannot change what a later call
	// returns, or what the transaction itself decodes.
	return bytes.Clone(body), nil
}

// responseBody reads and buffers the response body, returning the transaction's
// own slice rather than a copy of it.
func (t *Transaction) responseBody() ([]byte, error) {

	const location = "remote.Transaction.responseBody"

	// Internal callers use this to avoid duplicating a body that they only read.
	// Anything handed outside the package goes through ResponseBody instead.

	// Guard against NPE
	if t.response == nil {
		return []byte{}, derp.Internal(location, "Response object is nil")
	}

	// Determine the maximum number of bytes to read (falling back to the default).
	maxSize := t.maxResponseSize
	if maxSize <= 0 {
		maxSize = defaultMaxResponseSize
	}

	// RULE: Keep the limit one below MaxInt64, so the "+1" below cannot overflow
	// into a negative count. io.LimitReader reads nothing at all when N is
	// negative, which would turn "no practical limit" into an empty body.
	maxSize = min(maxSize, math.MaxInt64-1)

	// Read up to maxSize+1 bytes, so a body that exceeds the limit can be detected.
	originalBytes, err := io.ReadAll(io.LimitReader(t.response.Body, maxSize+1))

	if err != nil {
		return []byte{}, derp.Wrap(err, location, "Unable to read response body")
	}

	if int64(len(originalBytes)) > maxSize {
		return []byte{}, derp.Internal(location, "Response body exceeds maximum size", maxSize)
	}

	// Replace the (now used up) Body reader
	t.response.Body = io.NopCloser(bytes.NewReader(originalBytes))

	// Return success
	return originalBytes, nil
}

// ResponseBodyReader returns an io.Reader for the response body.
func (t *Transaction) ResponseBodyReader() io.Reader {

	// The internal buffer is safe to share here, because a bytes.Reader only reads.
	if body, err := t.responseBody(); err == nil {
		return bytes.NewReader(body)
	}

	return bytes.NewReader([]byte{})
}

// decodeResponseBody unmarshals the response body into the result object.
func (t *Transaction) decodeResponseBody(body []byte, result any) error {

	const location = "remote.Transaction.decodeResponseBody"

	// If we don't actually have a result (common for error documents) then there's nothing to do.
	if result == nil {
		return nil
	}

	// If we have a reader/string/byte array, then just read the body straight into it.
	switch result := result.(type) {

	case io.Writer:
		if _, err := result.Write(body); err != nil {
			err = derp.WrapHTTPError(err, t.request, t.response)
			err = derp.Wrap(err, location, "Unable to write response body to io.Writer", result, derp.WithInternalError())
			return err
		}
		return nil

	case *[]byte:
		// Copy, so the caller owns the result outright. Handing back `body` would
		// alias the buffer that ResponseBody kept for re-reads, letting a caller
		// who writes to their own result silently rewrite the response.
		*result = bytes.Clone(body)
		return nil

	case *string:
		*result = string(body)
		return nil
	}

	// Otherwise, try to use the content type to pick an unmarshaller
	contentType := t.response.Header.Get(ContentType) // Get the content type from the header
	contentType, _, _ = strings.Cut(contentType, ";") // Strip out suffixes, such as "; charset=utf-8"

	switch contentType {

	case ContentTypePlain, ContentTypeHTML:
		// Text can only be read into an io.Writer, *string, or *[]byte, which were handled above
		return t.undecodableResponse(location, contentType)

	// RULE: The body is never a detail on these errors.  The HTTPError already keeps a capped copy,
	// and a detail is uncapped -- so a large page reached the error log whole, bypassing derp's limit.
	case
		ContentTypeXML,
		contentTypeNonStandardXMLText,
		ContentTypeRSSXML,
		ContentTypeAtomXML:

		// Parse the result and return to the caller.
		if err := xml.Unmarshal(body, result); err != nil {
			err = derp.WrapHTTPError(err, t.request, t.response)
			err = derp.Wrap(err, location, "Unable to unmarshal XML Response", result, derp.WithInternalError())
			return err
		}

		return nil

	case
		ContentTypeJSON,
		ContentTypeJSONLD,
		ContentTypeActivityPub,
		ContentTypeJSONResourceDescriptor,
		ContentTypeJSONFeed,
		contentTypeNonStandardJSONText:

		// Parse the result and return to the caller.
		if err := json.Unmarshal(body, result); err != nil {
			err = derp.WrapHTTPError(err, t.request, t.response)
			err = derp.Wrap(err, location, "Unable to unmarshal JSON Response", result, derp.WithInternalError())
			return err
		}

		return nil
	}

	// If we're here, it means we don't know how to unmarshal the response body.
	return t.undecodableResponse(location, contentType)
}

// undecodableResponse reports a response that arrived intact, but in a form that cannot be
// decoded into the caller's result.
func (t *Transaction) undecodableResponse(location string, contentType string) error {

	// Name what arrived, so the explanation says more than the status line
	answered := "with no Content-Type"

	if contentType != "" {
		// RULE: Drop any rune the cut landed inside.  This header is written by the server, and
		// a half-character would leave invalid UTF-8 in a message bound for a log and a web page.
		answered = strings.ToValidUTF8(contentType[:min(len(contentType), contentTypeMaxLength)], "")
	}

	httpError := derp.NewHTTPError(t.request, t.response)
	httpError.Message = "Server answered " + answered + ", which cannot be decoded into the requested result"

	// RULE: Report 502 (Bad Gateway).  The upstream sent something unusable, a 500 would place the
	// fault in this client, and callers branch on the difference (see IsUndecodableResponse).
	return derp.Wrap(httpError, location, httpError.Message, derp.WithBadGateway())
}

// IsUndecodableResponse returns TRUE when a server answered successfully, but with a body that
// could not be decoded into the result the caller asked for.
func IsUndecodableResponse(err error) bool {

	// RULE: Only remote's own 502 counts.  A 502 sent by the SERVER comes from a proxy whose
	// origin is down, and the 2xx test below rejects it because its own status is 502.
	if !derp.IsBadGateway(err) {
		return false
	}

	// The response that actually arrived is carried by the HTTPError
	var httpError derp.HTTPError

	if !errors.As(err, &httpError) {
		return false
	}

	status := httpError.Response.StatusCode
	return (status >= http.StatusOK) && (status < http.StatusMultipleChoices)
}
