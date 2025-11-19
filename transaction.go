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
	"github.com/benpate/turbine/queue"
)

// Transaction represents a single HTTP request/response to a remote HTTP server.
type Transaction struct {
	client       *http.Client       // HTTP client to use to execute the request.  This may be overridden or updated by the calling program.
	method       string             // HTTP method to use when sending the request
	url          string             // URL of the remote server to call
	header       map[string]string  // HTTP Header values to send in the request
	query        url.Values         // Query String to append to the URL
	form         url.Values         // (if set) Form data to pass to the remote server as x-www-form-urlencoded
	body         any                // Other data to send in the body.  Encoding determined by header["Content-Type"]
	success      any                // Object to parse the response into -- IF the status code is successful
	failure      any                // Object to parse the response into -- IF the status code is NOT successful
	options      []Option           // options to execute on the request/response
	request      *http.Request      // HTTP request that is delivered to the remote server
	response     *http.Response     // HTTP response that is returned from the remote server
	queue        *queue.Queue       // (optional) Queue to use for sending transactions
	queueOptions []queue.TaskOption // (optional) Options to use when sending transactions to the queue
}

/******************************************
 * Chaining API methods
 ******************************************/

// Get assigns the HTTP method for this transaction.
func (t *Transaction) Method(method string) *Transaction {
	t.method = method
	return t
}

// Get assigns the URL for this transaction.
func (t *Transaction) URL(url string) *Transaction {
	t.url = url
	return t
}

// Get assigns the HTTP method and URL for this transaction.
func (t *Transaction) Get(url string) *Transaction {
	t.method = http.MethodGet
	t.url = url
	return t
}

// Post assigns the HTTP method and URL for this transaction.
func (t *Transaction) Post(url string) *Transaction {
	t.method = http.MethodPost
	t.url = url
	return t
}

// Put assigns the HTTP method and URL for this transaction.
func (t *Transaction) Put(url string) *Transaction {
	t.method = http.MethodPut
	t.url = url
	return t
}

// Patch assigns the HTTP method and URL for this transaction.
func (t *Transaction) Patch(url string) *Transaction {
	t.method = http.MethodPatch
	t.url = url
	return t
}

// Delete assigns the HTTP method and URL for this transaction.
func (t *Transaction) Delete(url string) *Transaction {
	t.method = http.MethodDelete
	t.url = url
	return t
}

// Client sets the HTTP client to use for the transaction.
func (t *Transaction) Client(client *http.Client) *Transaction {
	t.client = client
	return t
}

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
	t.query.Add(name, value)
	return t
}

// Form adds a name/value pair to the form data to be sent to the remote server.
func (t *Transaction) Form(name string, value string) *Transaction {
	t.form.Add(name, value)
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

func (t *Transaction) Queue(queue *queue.Queue, options ...queue.TaskOption) *Transaction {
	t.queue = queue
	t.queueOptions = options
	return t
}

// With lets you add remote.Options to the transaction. Options modify
// transaction data before and after it is sent to the remote server.
func (t *Transaction) With(options ...Option) *Transaction {
	t.options = append(t.options, options...)
	return t
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
		return derp.Wrap(err, location, "Unable to create HTTP request", t.errorReport())
	} else {
		t.request = request
	}

	// Execute options.ModifyRequest
	replacedResponse := t.onModifyRequest(t.request)

	switch {

	// If the response has been replaced,
	// then use it and DON'T send the request.
	case replacedResponse != nil:
		t.response = replacedResponse

	// If a queue has been provided, DON'T use HTTP
	// and send the transaction to the queue instead.
	case t.queue != nil:

		// Serialize the Transaction
		task := queue.NewTask(queueTransactionName, t.MarshalMap(), t.queueOptions...)

		// Send it to the queue
		if err := t.queue.Publish(task); err != nil {
			return derp.Wrap(err, location, "Error sending transaction to queue", task)
		}

		return nil

	// Otherwise, send the request to the remote server.
	default:

		// Executing request using HTTP client
		t.response, err = t.client.Do(t.request)

		if err != nil {
			return derp.Wrap(err, location, "Error executing HTTP request", t.errorReport(), derp.WithCode(http.StatusInternalServerError))
		}
	}

	// onAfterRequest modifies the response received from the server.
	if err := t.onAfterRequest(t.response); err != nil {
		return derp.Wrap(err, location, "Error executing options.Response", t.errorReport())
	}

	// read the body of the response
	body, err := t.ResponseBody()
	statusCode := t.statusCode()

	if err != nil {
		return derp.Wrap(err, location, "Unable to read response body", t.errorReport(), t.response, derp.WithCode(statusCode))
	}

	// If Response Code is NOT "OK", then handle the error
	if (statusCode < 200) || (statusCode > 299) {

		// Try to decode the response body into the failure object
		if t.failure != nil {
			if err := t.decodeResponseBody(body, t.failure); err != nil {
				return derp.Wrap(err, location, "Unable to parse error response", err, body, derp.WithCode(statusCode))
			}
		}

		// Return the error to the caller
		return derp.InternalError(location, "Error returned by remote service", t.errorReport(), derp.WithCode(statusCode))
	}

	// Fall through to here means that this is a successful response.
	// Decode the response body into the success object.
	if err := t.decodeResponseBody(body, t.success); err != nil {
		return derp.InternalError(location, "Error processing response body", err, t.errorReport())
	}

	// Glorious success.
	return nil
}

func (t *Transaction) assembleRequest() (*http.Request, error) {

	const location = "remote.Transaction.assembleRequest"

	var bodyReader io.Reader

	if err := t.assembleBearCap(); err != nil {
		return nil, derp.Wrap(err, location, "Error assembling BearCap")
	}

	// GET methods don't have an HTTP Body.  For all other methods,
	// it's time to defined the body content.
	if t.method != http.MethodGet {

		body, err := t.RequestBody()

		if err != nil {
			return nil, derp.Wrap(err, location, "Unable to create Request Body", t.body, t.errorReport(), derp.WithCode(http.StatusInternalServerError))
		}

		bodyReader = bytes.NewReader(body)
	}

	// Create the HTTP client request
	result, err := http.NewRequest(t.method, t.RequestURL(), bodyReader)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to create HTTP request", t.errorReport(), derp.WithCode(http.StatusInternalServerError))
	}

	// Add headers to httpRequest
	for key, value := range t.header {
		result.Header.Add(key, value)
	}

	return result, nil
}

// assembleBearCap pre-processes special bearer capability URLs.
// And of course, there's no "actual" documentation for this, so we're just going to use
// https://docs.joinmastodon.org/spec/bearcaps/ as the canonical source.
func (t *Transaction) assembleBearCap() error {

	const location = "remote.Transaction.assembleBearCap"

	// BearCap URLs are special.  They are used to pass a token to a remote server
	if strings.HasPrefix(t.url, "bear:?") {

		// Splie the URL into the bearcap "protocol" and the query string
		_, queryString, _ := strings.Cut(t.url, "?")

		values, err := url.ParseQuery(queryString)

		if err != nil {
			return derp.Wrap(err, location, "Invalid BearCap URL", t.url)
		}

		// Validate the "u" parameter is present
		uri := values.Get("u")

		if uri == "" {
			return derp.InternalError(location, "BearCap URL is required", t.url)
		}

		// Validate the "t" parameter is present
		token := values.Get("t")

		if token == "" {
			return derp.InternalError(location, "BearCap Token is required", t.url)
		}

		// Set the correct values in the transaction.
		t.url = uri
		t.header["Authorization"] = "Bearer " + token
	}

	// Success!!
	return nil
}

func (t *Transaction) statusCode() int {

	if t.response != nil {
		return t.response.StatusCode
	}

	return 0
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
