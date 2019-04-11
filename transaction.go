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
	"time"

	"github.com/benpate/derp"
)

// Transaction represents a single HTTP request/response to a remote HTTP server.
type Transaction struct {
	Client        *http.Client      // HTTP client to use to execute the request.  This may be overridden or updated by the calling program.
	Method        string            // HTTP method to use when sending the request
	URLValue      string            // URL of the remote server to call
	HeaderValues  map[string]string // HTTP Header values to send in the request
	QueryString   url.Values        // Query String to append to the URL
	FormData      url.Values        // (if set) Form data to pass to the remote server as x-www-form-urlencoded
	BodyObject    interface{}       // Other data to send in the body.  Encoding determined by header["Content-Type"]
	SuccessObject interface{}       // Object to parse the response into -- IF the status code is successful
	FailureObject interface{}       // Object to parse the response into -- IF the status code is NOT successful
	Middleware    []Middleware      // Middleware to execute on the request/response
}

// Middleware is a decorator that can modify the request before it is sent to the remote HTTP server,
// or modify the response after it is returned by the remote HTTP server.
type Middleware interface {
	Config(*Transaction) *derp.Error
	Request(*http.Request) *derp.Error
	Response(*http.Response) *derp.Error
}

// ErrorReport includes all the data returned by a transaction if it throws an error for any reason.
type ErrorReport struct {
	Request    string
	StatusCode int
	Status     string
	Header     http.Header
	Body       string
}

func newTransaction(method string, urlValue string) *Transaction {

	t := &Transaction{
		Client:       &http.Client{Timeout: 10 * time.Second},
		Method:       method,
		URLValue:     urlValue,
		HeaderValues: map[string]string{},
		QueryString:  url.Values{},
		FormData:     url.Values{},
		Middleware:   []Middleware{},
	}

	t.ContentType(ContentTypePlain)

	return t
}

// Get creates a new HTTP request to the designated URL, using the GET method
func Get(url string) *Transaction {
	return newTransaction(http.MethodGet, url)
}

// Post creates a new HTTP request to the designated URL, using the POST method
func Post(url string) *Transaction {
	return newTransaction(http.MethodPost, url)
}

// Put creates a new HTTP request to the designated URL, using the PUT method
func Put(url string) *Transaction {
	return newTransaction(http.MethodPut, url)
}

// Patch creates a new HTTP request to the designated URL, using the PATCH method
func Patch(url string) *Transaction {
	return newTransaction(http.MethodPatch, url)
}

// Delete creates a new HTTP request to the designated URL, using the DELETE method.
func Delete(url string) *Transaction {
	return newTransaction(http.MethodDelete, url)
}

// Query sets a name/value pair in the URL query string.
func (t *Transaction) Query(name string, value string) *Transaction {
	t.QueryString.Set(name, value)
	return t
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

// Form adds a name/value pair to the form data to be sent to the remote server.
func (t *Transaction) Form(name string, value string) *Transaction {
	t.FormData.Set(name, value)
	return t.ContentType(ContentTypeForm)
}

// Use lets you add middleware to the transaction. Middleware is able to modify
// transaction data before and after it is sent to the remote server.
func (t *Transaction) Use(middleware ...Middleware) *Transaction {
	t.Middleware = append(t.Middleware, middleware...)
	return t
}

// Result sets the objects for parsing HTTP success and failure responses
func (t *Transaction) Result(success interface{}, failure interface{}) *Transaction {
	t.SuccessObject = success
	t.FailureObject = failure
	return t
}

// Send executes the transaction, sending the request to the remote server.
func (t *Transaction) Send() *derp.Error {

	var err *derp.Error
	var bodyReader io.Reader

	// Execute middleware.Config
	for _, middleware := range t.Middleware {
		if middleware != nil {
			if err := middleware.Config(t); err != nil {
				return derp.New("remote.Result", "Middleware Error: Config", err, 0, t.getErrorReport())
			}
		}
	}

	// GET methods don't have an HTTP Body.  For all other methods,
	// it's time to defined the body content.
	if t.Method != "GET" {

		bodyReader, err = t.getRequestBody()

		if err != nil {
			return derp.New("remote.Result", "Error Creating Request Body", err, 0, nil)
		}
	}

	// Create the HTTP client request
	httpRequest, errr := http.NewRequest(t.Method, t.getURL(), bodyReader)

	if errr != nil {
		return derp.New("remote.Result", "Error creating HTTP request", err, 0, t.getErrorReport())
	}

	// Add headers to httpRequest
	for key, value := range t.HeaderValues {
		httpRequest.Header.Add(key, value)
	}

	// Execute middleware.Request
	for _, middleware := range t.Middleware {
		if middleware != nil {
			if err := middleware.Request(httpRequest); err != nil {
				return derp.New("remote.Result", "Middleware Error: Request", err, 0, t.getErrorReport())
			}
		}
	}

	// Executing request using HTTP client
	response, errr := t.Client.Do(httpRequest)

	if errr != nil {
		return derp.New("remote.Result", "Error executing HTTP request", errr, response.StatusCode, t.getErrorReport())
	}

	// Execute middleware.Response
	for _, middleware := range t.Middleware {
		if middleware != nil {
			if err := middleware.Response(response); err != nil {
				return derp.New("remote.Result", "Middleware Error: Response", err, 0, t.getErrorReport())
			}
		}
	}

	// Packing into response
	body, errr := ioutil.ReadAll(response.Body)

	if errr != nil {
		return derp.New("netclient.Do", "Error Reading Response Body", errr, 0, t.getErrorReport(), response)
	}

	// If Response Code is NOT "OK", then handle the error
	if (response.StatusCode < 200) || (response.StatusCode > 299) {

		/*
			errorReport := HTTPErrorReport{
				Request:    t.getFullURL(),
				StatusCode: response.StatusCode,
				Status:     http.StatusText(response.StatusCode),
				Header:     response.Header,
				Body:       string(body),
			}
		*/

		err := derp.New("netclient.Do", "Error Result from Remote Service", nil, response.StatusCode, t.getErrorReport())

		// If we ALSO have an error object, then try to process the response body into that
		if t.FailureObject != nil {
			if e := t.readResponseBody(body, t.FailureObject); e != nil {
				err = derp.New("netclient.Do", "Error Parsing Error Body", e, 0, body)
			}
		}

		return err
	}

	// Fall through to here means that this is a successful response.
	// Try to read the response body
	if err := t.readResponseBody(body, t.SuccessObject); err != nil {

		/*
			errorReport := HTTPErrorReport{
				Request:    t.getFullURL(),
				StatusCode: response.StatusCode,
				Status:     http.StatusText(response.StatusCode),
				Header:     response.Header,
				Body:       string(body),
			}
		*/
		return derp.New("netclient.Do", "Error reading response body", err, 0, t.getErrorReport())
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
			return nil, derp.New("remote.getJSONReader", "Error Marshalling JSON", err, 0, t.BodyObject)
		}

		return bytes.NewReader(j), nil
	}

	// Fall through to here means that we have an unrecognized content type.  Return an error.
	return strings.NewReader(""), derp.New("remote.getRequestBodyReader", "Unsupported Content-Type", nil, 0, contentType)
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
			return derp.New("netclient.Do", "Error Unmarshalling JSON Response", err, 0, string(body), result)
		}
	}

	return nil
}

//TODO: finish error reporting.
func (t *Transaction) getErrorReport() ErrorReport {
	return ErrorReport{}
}
