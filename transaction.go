// Package remote provides a simple and clean API for making HTTP requests to remote servers.
package remote

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/benpate/derp"
)

// Transaction represents a single HTTP request/response to a remote HTTP server.
type Transaction struct {
	client   *http.Client      // HTTP client to use to execute the request.  This may be overridden or updated by the calling program.
	method   string            // HTTP method to use when sending the request
	url      string            // URL of the remote server to call
	header   map[string]string // HTTP Header values to send in the request
	query    url.Values        // Query String to append to the URL
	form     url.Values        // (if set) Form data to pass to the remote server as x-www-form-urlencoded
	body     any               // Other data to send in the body.  Encoding determined by header["Content-Type"]
	success  any               // Object to parse the response into -- IF the status code is successful
	failure  any               // Object to parse the response into -- IF the status code is NOT successful
	options  []Option          // options to execute on the request/response
	request  *http.Request     // HTTP request that is delivered to the remote server
	response *http.Response    // HTTP response that is returned from the remote server
}

/******************************************
 * Chaining API methods
 ******************************************/

// Header sets a designated header value in the HTTP request.
func (t *Transaction) Header(name string, value string) *Transaction {
	t.header[name] = value
	return t
}

// Accept sets the Content-Type header of the HTTP request.
func (t *Transaction) Accept(contentTypes ...string) *Transaction {

	switch len(contentTypes) {
	case 0:
		return t.Header(Accept, "*/*")

	case 1:
		return t.Header(Accept, contentTypes[0])

	}

	// Build the Accept header with priorities
	accept := ""
	q := 1.0
	for _, contentType := range contentTypes {
		accept += contentType + ";q=" + strconv.FormatFloat(q, 'f', 1, 64) + ", "
		q -= 0.1
	}

	return t.Header(Accept, strings.TrimRight(accept, ", "))
}

// ContentType sets the Content-Type header of the HTTP request.
func (t *Transaction) ContentType(value string) *Transaction {
	return t.Header(ContentType, value)
}

func (t *Transaction) UserAgent(value string) *Transaction {
	return t.Header(UserAgent, value)
}

// Query sets a name/value pair in the URL query string.
func (t *Transaction) Query(name string, value string) *Transaction {
	t.query.Set(name, value)
	return t
}

// Form adds a name/value pair to the form data to be sent to the remote server.
func (t *Transaction) Form(name string, value string) *Transaction {
	t.form.Set(name, value)
	return t.ContentType(ContentTypeForm)
}

// Body sets the request body, to be encoded as plain text
func (t *Transaction) Body(value string) *Transaction {
	t.body = value

	if t.isContentTypeEmpty() {
		t.ContentType(ContentTypePlain)
	}
	return t
}

// JSON sets the request body, to be encoded as JSON.
func (t *Transaction) JSON(value any) *Transaction {
	t.body = value

	if t.isContentTypeEmpty() {
		t.ContentType(ContentTypeJSON)
	}
	return t
}

// XML sets the request body, to be encoded as XML.
func (t *Transaction) XML(value any) *Transaction {
	t.body = value

	if t.isContentTypeEmpty() {
		t.ContentType(ContentTypeXML)
	}
	return t
}

func (t *Transaction) isContentTypeEmpty() bool {
	return t.header[ContentType] == ""
}

// Use lets you add remote.Options to the transaction. Options modify
// transaction data before and after it is sent to the remote server.
func (t *Transaction) Use(options ...Option) *Transaction {
	t.options = append(t.options, options...)
	return t
}

// WithOptions is an alias for the `Use` method. It applies one or more
// request.options to the Transaction.
func (t *Transaction) WithOptions(options ...Option) *Transaction {
	return t.Use(options...)
}

// Result sets the object for parsing HTTP success responses
func (t *Transaction) Result(object any) *Transaction {
	t.success = object
	return t
}

// Error sets the object for parsing HTTP error responses
func (t *Transaction) Error(object any) *Transaction {
	t.failure = object
	return t
}

// Send executes the transaction, sending the request to the remote server.
func (t *Transaction) Send() error {

	const location = "remote.Transaction.Send"

	var err error

	// onBeforeRequest modifies the transaction before an http.Request is created
	if err := t.onBeforeRequest(); err != nil {
		return err
	}

	// Assemble the HTTP request from the transaction data
	if request, err := t.assembleRequest(); err != nil {
		return derp.Wrap(err, location, "Error creating HTTP request", t.errorReport())
	} else {
		t.request = request
	}

	// Execute options.Request
	if response := t.onModifyRequest(t.request); response != nil {
		t.response = response

	} else {

		// Executing request using HTTP client
		t.response, err = t.client.Do(t.request)

		if err != nil {
			err = derp.Wrap(err, location, "Error executing HTTP request", t.errorReport())
			derp.SetErrorCode(err, derp.CodeInternalError)
			return err
		}
	}

	// onAfterRequest modifies the response received from the server.
	if err := t.onAfterRequest(t.response); err != nil {
		return derp.Wrap(err, location, "Error executing options.Response", t.errorReport())
	}

	// read the body of the response
	body, err := t.ResponseBody()

	if err != nil {
		err = derp.Wrap(err, location, "Error reading response body", t.errorReport(), t.response)
		derp.SetErrorCode(err, t.response.StatusCode)
		return err
	}

	// If Response Code is NOT "OK", then handle the error
	if (t.response.StatusCode < 200) || (t.response.StatusCode > 299) {

		// Try to decode the response body into the failure object
		if t.failure != nil {
			if err := t.decodeResponseBody(body, t.failure); err != nil {
				err = derp.Wrap(err, location, "Unable to parse error response", err, body)
				derp.SetErrorCode(err, t.response.StatusCode)
				return err
			}
		}

		// Return the error to the caller
		return derp.New(t.response.StatusCode, location, "Error returned by remote service", t.errorReport())
	}

	// Fall through to here means that this is a successful response.
	// Decode the response body into the success object.
	if err := t.decodeResponseBody(body, t.success); err != nil {
		return derp.NewInternalError(location, "Error processing response body", err, t.errorReport())
	}

	// Glorious success.
	return nil
}

func (t *Transaction) assembleRequest() (*http.Request, error) {

	const location = "remote.Transaction.assembleRequest"

	var bodyReader io.Reader

	// GET methods don't have an HTTP Body.  For all other methods,
	// it's time to defined the body content.
	if t.method != http.MethodGet {

		body, err := t.RequestBody()

		if err != nil {
			err = derp.Wrap(err, location, "Error Creating Request Body", t.body, t.errorReport())
			derp.SetErrorCode(err, derp.CodeInternalError)
			return nil, err
		}

		bodyReader = bytes.NewReader(body)
	}

	// Create the HTTP client request
	result, err := http.NewRequest(t.method, t.RequestURL(), bodyReader)

	if err != nil {
		err = derp.Wrap(err, location, "Error creating HTTP request", t.errorReport())
		derp.SetErrorCode(err, derp.CodeInternalError)
		return nil, err
	}

	// Add headers to httpRequest
	for key, value := range t.header {
		result.Header.Add(key, value)
	}

	return result, nil
}

// ErrorReport generates a data dump of the current state of the HTTP transaction.
// This is used when reporting errors via derp, to provide insights into what went wrong.
func (t *Transaction) errorReport() ErrorReport {

	result := ErrorReport{}

	result.URL = t.RequestURL()
	result.Request.Method = t.method
	result.Request.Header = t.header

	if body, err := t.RequestBody(); err == nil {
		result.Request.Body = string(body)
	}

	if t.response != nil {
		result.Response.StatusCode = t.response.StatusCode
		result.Response.Status = t.response.Status
		result.Response.Header = t.response.Header

		if body, err := t.ResponseBody(); err == nil {
			result.Response.Body = string(body)
		}
	}

	return result
}
