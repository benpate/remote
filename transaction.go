// Package remote provides a simple and clean API for making HTTP requests to remote servers.
package remote

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/benpate/derp"
)

// Transaction represents a single HTTP request/response to a remote HTTP server.
type Transaction struct {
	Client         *http.Client      // HTTP client to use to execute the request.  This may be overridden or updated by the calling program.
	Method         string            // HTTP method to use when sending the request
	URLValue       string            // URL of the remote server to call
	HeaderValues   map[string]string // HTTP Header values to send in the request
	QueryString    url.Values        // Query String to append to the URL
	FormData       url.Values        // (if set) Form data to pass to the remote server as x-www-form-urlencoded
	BodyObject     any               // Other data to send in the body.  Encoding determined by header["Content-Type"]
	SuccessObject  any               // Object to parse the response into -- IF the status code is successful
	FailureObject  any               // Object to parse the response into -- IF the status code is NOT successful
	Middleware     []Middleware      // Middleware to execute on the request/response
	RequestObject  *http.Request     // HTTP request that is delivered to the remote server
	ResponseObject *http.Response    // HTTP response that is returned from the remote server
}

// Header sets a designated header value in the HTTP request.
func (t *Transaction) Header(name string, value string) *Transaction {
	t.HeaderValues[name] = value
	return t
}

// Accept sets the Content-Type header of the HTTP request.
func (t *Transaction) Accept(contentTypes ...string) *Transaction {

	switch len(contentTypes) {
	case 0:
		return t.Header("Accept", "*/*")

	case 1:
		return t.Header("Accept", contentTypes[0])

	}

	// Build the Accept header with priorities
	accept := ""
	q := 1.0
	for _, contentType := range contentTypes {
		accept += contentType + ";q=" + strconv.FormatFloat(q, 'f', 1, 64) + ", "
		q -= 0.1
	}

	return t.Header("Accept", strings.TrimRight(accept, ", "))
}

// ContentType sets the Content-Type header of the HTTP request.
func (t *Transaction) ContentType(value string) *Transaction {
	return t.Header(ContentType, value)
}

// Query sets a name/value pair in the URL query string.
func (t *Transaction) Query(name string, value string) *Transaction {
	t.QueryString.Set(name, value)
	return t
}

// Form adds a name/value pair to the form data to be sent to the remote server.
func (t *Transaction) Form(name string, value string) *Transaction {
	t.FormData.Set(name, value)
	return t.ContentType(ContentTypeForm)
}

// Body sets the request body, to be encoded as plain text
func (t *Transaction) Body(value string) *Transaction {
	t.BodyObject = value

	if t.IsContentTypeEmpty() {
		t.ContentType(ContentTypePlain)
	}
	return t
}

// JSON sets the request body, to be encoded as JSON.
func (t *Transaction) JSON(value any) *Transaction {
	t.BodyObject = value

	if t.IsContentTypeEmpty() {
		t.ContentType(ContentTypeJSON)
	}
	return t
}

// XML sets the request body, to be encoded as XML.
func (t *Transaction) XML(value any) *Transaction {
	t.BodyObject = value
	if t.IsContentTypeEmpty() {
		t.ContentType(ContentTypeXML)
	}
	return t
}

func (t *Transaction) IsContentTypeEmpty() bool {
	return t.HeaderValues[ContentType] == ""
}

// Use lets you add middleware to the transaction. Middleware is able to modify
// transaction data before and after it is sent to the remote server.
func (t *Transaction) Use(middleware ...Middleware) *Transaction {
	t.Middleware = append(t.Middleware, middleware...)
	return t
}

// Response sets the objects for parsing HTTP success and failure responses
func (t *Transaction) Response(success any, failure any) *Transaction {
	t.SuccessObject = success
	t.FailureObject = failure
	return t
}

// Send executes the transaction, sending the request to the remote server.
func (t *Transaction) Send() error {

	var err error
	var bodyReader io.Reader

	// Execute middleware.Config
	if err := t.doMiddlewareConfig(); err != nil {
		return err
	}

	// GET methods don't have an HTTP Body.  For all other methods,
	// it's time to defined the body content.
	if t.Method != "GET" {

		bodyReader, err = t.getRequestBody()

		if err != nil {
			err = derp.Wrap(err, "remote.Transaction.Send", "Error Creating Request Body", t.BodyObject, t.ErrorReport())
			derp.SetErrorCode(err, derp.CodeInternalError)
			return err
		}
	}

	// Create the HTTP client request
	t.RequestObject, err = http.NewRequest(t.Method, t.getURL(), bodyReader)

	if err != nil {
		err = derp.Wrap(err, "remote.Transaction.Send", "Error creating HTTP request", t.ErrorReport())
		derp.SetErrorCode(err, derp.CodeInternalError)
		return err
	}

	// Add headers to httpRequest
	for key, value := range t.HeaderValues {
		t.RequestObject.Header.Add(key, value)
	}

	// Execute middleware.Request
	if err := t.doMiddlewareRequest(t.RequestObject); err != nil {
		return derp.Wrap(err, "remote.Transaction.Send", "Error executing middleware.Request", t.ErrorReport())
	}

	// Executing request using HTTP client
	t.ResponseObject, err = t.Client.Do(t.RequestObject)

	if err != nil {
		err = derp.Wrap(err, "remote.Transaction.Send", "Error executing HTTP request", t.ErrorReport())
		derp.SetErrorCode(err, derp.CodeInternalError)
		return err
	}

	// Packing into t.ResponseObject
	body, err := io.ReadAll(t.ResponseObject.Body)

	if err != nil {
		err = derp.Wrap(err, "remote.Transaction.Send", "Error reading response body", t.ErrorReport(), t.ResponseObject)
		derp.SetErrorCode(err, t.ResponseObject.StatusCode)
		return err
	}

	// Execute middleware.Response
	if err := t.doMiddlewareResponse(t.ResponseObject, &body); err != nil {
		return derp.Wrap(err, "remote.Transaction.Send", "Error executing middleware.Response", t.ErrorReport())
	}

	// If Response Code is NOT "OK", then handle the error
	if (t.ResponseObject.StatusCode < 200) || (t.ResponseObject.StatusCode > 299) {

		// If we ALSO have an error object, then try to process the response body into that
		if t.FailureObject != nil {
			if err := t.readResponseBody(body, t.FailureObject); err != nil {
				err = derp.Wrap(err, "remote.Transaction.Send", "Unable to parse error response", err, body)
				derp.SetErrorCode(err, t.ResponseObject.StatusCode)
				return err
			}
		}

		return derp.New(t.ResponseObject.StatusCode, "remote.Transaction.Send", "Error returned by remote service", t.ErrorReport())
	}

	// Fall through to here means that this is a successful response.
	// Try to read the response body
	if err := t.readResponseBody(body, t.SuccessObject); err != nil {
		return derp.NewInternalError("remote.Transaction.Send", "Error processing response body", err, t.ErrorReport())
	}

	// Silence means success.
	return nil
}

func (t *Transaction) getURL() string {
	result := t.URLValue

	if len(t.QueryString) > 0 {
		result += "?" + t.QueryString.Encode()
	}

	return result
}

func (t *Transaction) getRequestBody() (io.Reader, error) {

	// If we already have a reader for the Body, then just return that.
	switch t.BodyObject.(type) {

	case io.Reader:
		return t.BodyObject.(io.Reader), nil

	case string:
		return strings.NewReader(t.BodyObject.(string)), nil

	case []byte:
		return bytes.NewReader(t.BodyObject.([]byte)), nil
	}

	contentType := t.HeaderValues[ContentType]

	// Otherwise, use the correct Marshaller, based on the ContentType of the request
	switch contentType {

	case "", ContentTypePlain:
		return strings.NewReader(""), nil

	case ContentTypeForm:
		return strings.NewReader(t.FormData.Encode()), nil

	case ContentTypeJSON,
		ContentTypeActivityPub,
		ContentTypeJSONResourceDescriptor,
		ContentTypeJSONFeed,
		contentTypeNonStandardJSONText:

		j, err := json.Marshal(t.BodyObject)

		if err != nil {
			return nil, derp.NewInternalError("remote.Transaction.getRequestBody", "Error Marshalling JSON", err, t.ErrorReport(), t.BodyObject)
		}

		return bytes.NewReader(j), nil
	}

	// Fall through to here means that we have an unrecognized content type.  Return an error.
	return strings.NewReader(""), derp.NewInternalError("remote.Transaction.getRequestBody", "Unsupported Content-Type", contentType, t.ErrorReport())
}

// readResponseBody unmarshalls the response body into the result
func (t *Transaction) readResponseBody(body []byte, result any) error {

	// If we don't actually have a result (common for error documents) then there's nothing to do.
	if result == nil {
		return nil
	}

	// If we have a reader/string/byte array, then just read the body straight into it.
	switch result := result.(type) {

	case io.Writer:
		if _, err := result.Write(body); err != nil {
			return derp.Wrap(err, "remote.Transaction.readResponseBody", "Error writing response body to io.Writer", result)
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
	contentType := t.ResponseObject.Header.Get(ContentType) // Get the content type from the header
	contentType = strings.Split(contentType, ";")[0]        // Strip out suffixes, such as "; charset=utf-8"

	switch contentType {

	case ContentTypePlain, ContentTypeHTML:
		return derp.NewInternalError("remote.Transaction.readResponseBody", "HTML must be read into an io.Writer, *string, or *byte[]", result)

	case ContentTypeXML, contentTypeNonStandardXMLText:

		// Parse the result and return to the caller.
		if err := xml.Unmarshal(body, result); err != nil {
			return derp.NewInternalError("remote.Transaction.readResponseBody", "Error Unmarshalling XML Response", err, string(body), result, t.ErrorReport())
		}

		return nil

	case
		ContentTypeJSON,
		ContentTypeActivityPub,
		ContentTypeJSONResourceDescriptor,
		ContentTypeJSONFeed,
		contentTypeNonStandardJSONText:

		// Parse the result and return to the caller.
		if err := json.Unmarshal(body, result); err != nil {
			return derp.NewInternalError("remote.Transaction.readResponseBody", "Error Unmarshalling JSON Response", err, string(body), result, t.ErrorReport())
		}

		return nil
	}

	// If we're here, it means we don't know how to unmarshal the response body.
	return derp.NewInternalError("remote.Transaction.readResponseBody", "Unsupported Content-Type", contentType, t.ErrorReport())
}

// ErrorReport generates a data dump of the current state of the HTTP transaction.
// This is used when reporting errors via derp, to provide insights into what went wrong.
func (t *Transaction) ErrorReport() ErrorReport {

	result := ErrorReport{}

	body := strings.Builder{}
	bodyReader, _ := t.getRequestBody()
	io.Copy(&body, bodyReader) // nolint:errcheck

	result.URL = t.getURL()
	result.Request.Method = t.Method
	result.Request.Header = t.HeaderValues
	result.Request.Body = body.String()

	if t.ResponseObject != nil {
		result.Response.StatusCode = t.ResponseObject.StatusCode
		result.Response.Status = t.ResponseObject.Status
		result.Response.Header = t.ResponseObject.Header
	}

	return result
}
