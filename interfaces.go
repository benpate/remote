// Package remote provides a simple and clean API for making HTTP requests to remote servers.
package remote

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/benpate/derp"
)

const (

	// ContentType is the string used in the HTTP header to designate a MIME type
	ContentType = "Content-Type"

	// ContentTypePlain is the default plaintext MIME type
	ContentTypePlain = "text/plain"

	// ContentTypeJSON is the standard MIME Type for JSON content
	ContentTypeJSON = "application/json"

	// ContentTypeForm is the standard MIME Type for Form encoded content
	ContentTypeForm = "application/x-www-form-urlencoded"

	// ContentTypeXML is the standard MIME Type for XML content
	ContentTypeXML = "application/xml"
)

// Transaction represents a single HTTP request/response to a remote HTTP server.
type Transaction struct {
	Client       *http.Client
	Method       string
	URLValue     string
	HeaderValues map[string]string
	QueryString  url.Values
	FormData     url.Values
	BodyObject   interface{}
	ResultObject interface{}
	ErrorObject  interface{}
	Middleware   []Middleware
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
	return t.Header("Content-Type", value)
}

// BasicAuth sets the value of a HTTP header for basic Authentication
func (t *Transaction) BasicAuth(username, password string) *Transaction {
	return t.Header("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
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

func (t *Transaction) Result(success interface{}, failure interface{}) *derp.Error {

	var err *derp.Error
	var bodyReader io.Reader

	// Execute middleware.Config
	for _, middleware := range t.Middleware {
		if err := middleware.Config(t); err != nil {
			return derp.New("remote.Result", "Middleware Error: Config", err, 0, t.getErrorReport())
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

	// Execute middleware.Request
	for _, middleware := range t.Middleware {
		if err := middleware.Request(httpRequest); err != nil {
			return derp.New("remote.Result", "Middleware Error: Request", err, 0, t.getErrorReport())
		}
	}

	// Execute middleware.Response
	for _, middleware := range t.Middleware {
		middleware.Config(t)
	}

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

func (t *Transaction) getErrorReport() ErrorReport {
	return ErrorReport{}
}

type Middleware interface {
	Config(*Transaction) *derp.Error
	Request(*http.Request) *derp.Error
	Response(*http.Response) *derp.Error
}
