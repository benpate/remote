// Package remote provides a simple and clean API for making HTTP requests to remote servers.
package remote

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
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
	BodyObject     interface{}       // Other data to send in the body.  Encoding determined by header["Content-Type"]
	SuccessObject  interface{}       // Object to parse the response into -- IF the status code is successful
	FailureObject  interface{}       // Object to parse the response into -- IF the status code is NOT successful
	Middleware     []Middleware      // Middleware to execute on the request/response
	RequestObject  *http.Request     // HTTP request that is delivered to the remote server
	ResponseObject *http.Response    // HTTP response that is returned from the remote server
}

// Header sets a designated header value in the HTTP request.
func (t *Transaction) Header(name string, value string) *Transaction {
	t.HeaderValues[name] = value
	return t
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
	return t.ContentType(ContentTypePlain)
}

// JSON sets the request body, to be encoded as JSON.
func (t *Transaction) JSON(value interface{}) *Transaction {
	t.BodyObject = value
	return t.ContentType(ContentTypeJSON)
}

// XML sets the request body, to be encoded as XML.
func (t *Transaction) XML(value interface{}) *Transaction {
	t.BodyObject = value
	return t.ContentType(ContentTypeXML)
}

// Use lets you add middleware to the transaction. Middleware is able to modify
// transaction data before and after it is sent to the remote server.
func (t *Transaction) Use(middleware ...Middleware) *Transaction {
	t.Middleware = append(t.Middleware, middleware...)
	return t
}

// Response sets the objects for parsing HTTP success and failure responses
func (t *Transaction) Response(success interface{}, failure interface{}) *Transaction {
	t.SuccessObject = success
	t.FailureObject = failure
	return t
}

// Send executes the transaction, sending the request to the remote server.
func (t *Transaction) Send() *derp.Error {

	var err *derp.Error
	var errr error
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
			return derp.New(derp.CodeInternalError, "remote.Result", "Error Creating Request Body", err, t.ErrorReport())
		}
	}

	// Create the HTTP client request
	t.RequestObject, errr = http.NewRequest(t.Method, t.getURL(), bodyReader)

	if errr != nil {
		return derp.New(derp.CodeInternalError, "remote.Result", "Error creating HTTP request", err, t.ErrorReport())
	}

	// Add headers to httpRequest
	for key, value := range t.HeaderValues {
		t.RequestObject.Header.Add(key, value)
	}

	// Execute middleware.Request
	if err := t.doMiddlewareRequest(t.RequestObject); err != nil {
		return err
	}

	// Executing request using HTTP client
	t.ResponseObject, errr = t.Client.Do(t.RequestObject)

	if errr != nil {
		return derp.New(derp.CodeInternalError, "remote.Result", "Error executing HTTP request", errr, t.ErrorReport())
	}

	// Packing into t.ResponseObject
	body, errr := ioutil.ReadAll(t.ResponseObject.Body)

	if errr != nil {
		return derp.New(t.ResponseObject.StatusCode, "remote.Send", "Error Reading Response Body", errr, t.ErrorReport(), t.ResponseObject)
	}

	// Execute middleware.Response
	if err := t.doMiddlewareResponse(t.ResponseObject, &body); err != nil {
		return err
	}

	// If Response Code is NOT "OK", then handle the error
	if (t.ResponseObject.StatusCode < 200) || (t.ResponseObject.StatusCode > 299) {

		// If we ALSO have an error object, then try to process the response body into that
		if t.FailureObject != nil {
			if er := t.readResponseBody(body, t.FailureObject); er != nil {
				return derp.New(derp.CodeInternalError, "remote.Send", "Error Parsing Error Body", er, body)
			}
		}

		return derp.New(t.ResponseObject.StatusCode, "netclient.Do", "Error Result from Remote Service", t.ErrorReport())
	}

	// Fall through to here means that this is a successful response.
	// Try to read the response body
	if err := t.readResponseBody(body, t.SuccessObject); err != nil {
		return derp.New(derp.CodeInternalError, "remote.Send", "Error in readResponseBody()", err, t.ErrorReport())
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

func (t *Transaction) getRequestBody() (io.Reader, *derp.Error) {

	// If we already have a reader for the Body, then just return that.
	switch t.BodyObject.(type) {

	case io.Reader:
		return t.BodyObject.(io.Reader), nil

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

	case ContentTypeJSON:

		j, err := json.Marshal(t.BodyObject)

		if err != nil {
			return nil, derp.New(derp.CodeInternalError, "remote.getJSONReader", "Error Marshalling JSON", err, t.ErrorReport(), t.BodyObject)
		}

		return bytes.NewReader(j), nil
	}

	// Fall through to here means that we have an unrecognized content type.  Return an error.
	return strings.NewReader(""), derp.New(derp.CodeInternalError, "remote.getRequestBodyReader", "Unsupported Content-Type", contentType, t.ErrorReport())
}

// readResponseBody unmarshalls the response body into the result
func (t *Transaction) readResponseBody(body []byte, result interface{}) *derp.Error {

	// TODO: inspect MIME Type and use the appropriate decoder.

	// If we have defined a result variable, then try to unmarshal the results
	// into it.
	if result != nil {

		// If result is a pointer to a string (or slice of bytes) then just populate
		// the response directly into the result
		switch result.(type) {

		case *[]byte, []byte:
			result = body
			return nil

		case *string, string:
			result = string(body)
			return nil
		}

		// Fall through to here means its a more complex data type.  Try to
		// unmarshal based on content type

		// Parse the result and return to the caller.
		if err := json.Unmarshal(body, result); err != nil {
			return derp.New(derp.CodeInternalError, "remote.readResponseBody", "Error Unmarshalling JSON Response", err, string(body), result, t.ErrorReport())
		}
	}

	return nil
}

// ErrorReport generates a data dump of the current state of the HTTP transaction.
// This is used when reporting errors via derp, to provide insights into what went wrong.
func (t *Transaction) ErrorReport() ErrorReport {

	result := ErrorReport{}

	result.URL = t.getURL()
	result.Request.Method = t.Method
	result.Request.Header = t.HeaderValues

	if t.ResponseObject != nil {
		result.Response.StatusCode = t.ResponseObject.StatusCode
		result.Response.Status = t.ResponseObject.Status
		result.Response.Header = t.ResponseObject.Header
	}

	return result
}
