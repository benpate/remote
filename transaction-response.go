package remote

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
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

// ResponseBytes returns the original response body, as a byte array.
// This method replaces the original body reader, meaning that it can be called
// multiple times without error.
func (t *Transaction) ResponseBody() ([]byte, error) {

	// Guard against NPE
	if t.response == nil {
		return []byte{}, derp.InternalError("remote.Transaction.ResponseBytes", "Response object is nil", t.errorReport())
	}

	// Read the original response body
	originalBytes, err := io.ReadAll(t.response.Body)

	if err != nil {
		return []byte{}, derp.Wrap(err, "remote.Transaction.ResponseBytes", "Unable to read response body")
	}

	// Replace the (now used up) Body reader
	t.response.Body = io.NopCloser(bytes.NewReader(originalBytes))

	// Return success
	return originalBytes, nil
}

func (t *Transaction) ResponseBodyReader() io.Reader {

	if body, err := t.ResponseBody(); err == nil {
		return bytes.NewReader(body)
	}

	return bytes.NewReader([]byte{})
}

// readResponseBody unmarshalls the response body into the result
func (t *Transaction) decodeResponseBody(body []byte, result any) error {

	const location = "remote.Transaction.readResponseBody"

	// If we don't actually have a result (common for error documents) then there's nothing to do.
	if result == nil {
		return nil
	}

	// If we have a reader/string/byte array, then just read the body straight into it.
	switch result := result.(type) {

	case io.Writer:
		if _, err := result.Write(body); err != nil {
			return derp.Wrap(err, location, "Unable to write response body to io.Writer", result)
		}
		return nil

	case *[]byte:
		*result = body
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
		return derp.InternalError(location, "HTML must be read into an io.Writer, *string, or *byte[]", result)

	case
		ContentTypeXML,
		contentTypeNonStandardXMLText,
		ContentTypeRSSXML,
		ContentTypeAtomXML:

		// Parse the result and return to the caller.
		if err := xml.Unmarshal(body, result); err != nil {
			return derp.Wrap(err, location, "Unable to unmarshal XML Response", err, string(body), result, t.errorReport(), derp.WithCode(http.StatusInternalServerError))
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
			return derp.Wrap(err, location, "Unable to unmarshal JSON Response", err, string(body), result, t.errorReport(), derp.WithCode(http.StatusInternalServerError))
		}

		return nil
	}

	// If we're here, it means we don't know how to unmarshal the response body.
	return derp.InternalError(location, "Unsupported Content-Type", contentType, t.errorReport())
}
