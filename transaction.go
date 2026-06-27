// Package remote provides a simple and clean API for making HTTP requests to remote servers.
package remote

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/uri"
)

// Transaction represents a single HTTP request/response to a remote HTTP server.
type Transaction struct {
	method          string            // HTTP method to use when sending the request
	url             string            // URL of the remote server to call
	header          map[string]string // HTTP Header values to send in the request
	query           url.Values        // Query String to append to the URL
	form            url.Values        // (if set) Form data to pass to the remote server as x-www-form-urlencoded
	body            any               // Other data to send in the body.  Encoding determined by header["Content-Type"]
	success         any               // Object to parse the response into -- IF the status code is successful
	failure         any               // Object to parse the response into -- IF the status code is NOT successful
	options         []Option          // options to execute on the request/response
	allowedHosts    []string          // (if set) request URL host must match one of these values
	allowPrivateIPs bool              // if FALSE (the default), refuse to connect to non-public (private/internal) IP addresses
	maxResponseSize int64             // maximum number of bytes to read from the response body
	ctx             context.Context   // NOSONAR(S8242): request-scoped builder

	request  *http.Request  // HTTP request that is delivered to the remote server
	response *http.Response // HTTP response that is returned from the remote server

	roundTripper func(http.RoundTripper) http.RoundTripper // (if set) wraps the base transport with caller-supplied middleware
}

/******************************************
 * Chaining API methods
 ******************************************/

// Method assigns the HTTP method for this transaction.
func (t *Transaction) Method(method string) *Transaction {
	t.method = method
	return t
}

// URL assigns the URL for this transaction.
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

// WithRoundTripper wraps the transaction's SSRF-hardened base transport with the
// given middleware, letting callers add behavior such as caching or custom
// headers while keeping the private-IP guard underneath. The middleware receives
// the base transport as "next" and must delegate to it to perform the request.
func (t *Transaction) WithRoundTripper(wrap func(next http.RoundTripper) http.RoundTripper) *Transaction {
	t.roundTripper = wrap
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

	// Build the Accept header, assigning each type a descending q-value. The
	// value is computed from the index (not by repeated subtraction, which drifts)
	// and floored at 0.1, since RFC 9110 requires q to stay within [0, 1] and a
	// q of 0 means "not acceptable".
	var accept strings.Builder

	for index, contentType := range contentTypes {

		q := 1.0 - (float64(index) * 0.1)
		if q < 0.1 {
			q = 0.1
		}

		if index > 0 {
			accept.WriteString(", ")
		}

		accept.WriteString(contentType)
		accept.WriteString(";q=")
		accept.WriteString(strconv.FormatFloat(q, 'f', 1, 64))
	}

	return t.Header(Accept, accept.String())
}

// ContentType sets the Content-Type header of the HTTP request.
func (t *Transaction) ContentType(value string) *Transaction {
	return t.Header(ContentType, value)
}

// UserAgent sets the User-Agent header of the HTTP request.
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

// isContentTypeEmpty returns true if the Content-Type header has not been set.
func (t *Transaction) isContentTypeEmpty() bool {
	return t.header[ContentType] == ""
}

// With lets you add remote.Options to the transaction. Options modify
// transaction data before and after it is sent to the remote server.
func (t *Transaction) With(options ...Option) *Transaction {
	t.options = append(t.options, options...)
	return t
}

// AllowHosts restricts this transaction to the named hosts. When set, Send
// returns an error before contacting the server if the request URL's host is
// not in the list. This guards against requests to unexpected servers, for
// instance when the URL is user-supplied. Matching is case-insensitive.
func (t *Transaction) AllowHosts(hosts ...string) *Transaction {
	for _, host := range hosts {
		t.allowedHosts = append(t.allowedHosts, strings.ToLower(host))
	}
	return t
}

// AllowPrivateIPs controls whether the transaction may connect to non-public IP
// addresses (loopback, private, link-local, etc.). The default is FALSE, so such
// addresses are blocked to guard against SSRF: Send returns an error if the
// request (or any redirect) resolves to one. Set it to TRUE to permit them — for
// instance, when intentionally calling an internal or localhost service.
func (t *Transaction) AllowPrivateIPs(value bool) *Transaction {
	t.allowPrivateIPs = value
	return t
}

// MaxResponseSize sets the maximum number of bytes that will be read from the
// response body. A response larger than this causes Send to return an error,
// preventing an untrusted server from exhausting memory. The default is 1GB.
// A value of zero or less restores the default.
func (t *Transaction) MaxResponseSize(bytes int64) *Transaction {
	t.maxResponseSize = bytes
	return t
}

// WithContext attaches a context to the transaction, used to cancel the request
// or apply a deadline. If no context is set, a background context with a default
// one-minute timeout is used.
func (t *Transaction) WithContext(ctx context.Context) *Transaction {
	t.ctx = ctx
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

	// onBeforeRequest modifies the transaction before an http.Request is created
	if err := t.onBeforeRequest(); err != nil {
		return derp.Wrap(err, location, "Error in BeforeRequest option")
	}

	// Resolve the request context (applying the default timeout if none was set).
	ctx, cancel := t.requestContext()
	defer cancel()

	// Assemble the HTTP request from the transaction data
	request, err := t.assembleRequest(ctx)

	if err != nil {
		return derp.Wrap(err, location, "Creating HTTP request")
	}

	t.request = request

	// Send the request (or use a response substituted by a ModifyRequest option).
	if err := t.executeRequest(); err != nil {
		return derp.Wrap(err, location, "Sending request")
	}

	// A response must exist past this point; guard so we never dereference a nil.
	if t.response == nil {
		return derp.Internal(location, "No response received from server")
	}

	// Close the response body when we're done, to release the underlying
	// connection. ResponseBody (below) buffers the body in memory and swaps in a
	// re-readable NopCloser, so closing the original here does not prevent
	// callers from reading the response afterward.
	if body := t.response.Body; body != nil {
		defer func() {
			_ = body.Close()
		}()
	}

	// onAfterRequest modifies the response received from the server.
	if err := t.onAfterRequest(t.response); err != nil {
		err = derp.WrapHTTPError(err, t.request, t.response)
		err = derp.Wrap(err, location, "Building options.Response", derp.WithInternalError())
		return err
	}

	// read the body of the response
	body, err := t.ResponseBody()

	if err != nil {
		err = derp.WrapHTTPError(err, t.request, t.response)
		err = derp.Wrap(err, location, "Reading response body", derp.WithInternalError())
		return err
	}

	// Decode the response body into the success or failure object.
	if err := t.processResponse(body); err != nil {
		return derp.Wrap(err, location, "Processing response")
	}

	// Glorious success.
	return nil
}

// processResponse decodes the (already-read) response body into the transaction's
// success or failure object, based on the response status code. A non-2xx status
// always yields an error.
func (t *Transaction) processResponse(body []byte) error {

	const location = "remote.Transaction.processResponse"

	// A non-2xx status is an error; decode the body into the failure object if one is set.
	if statusCode := t.statusCode(); (statusCode < 200) || (statusCode > 299) {

		if t.failure != nil {
			if err := t.decodeResponseBody(body, t.failure); err != nil {
				err = derp.WrapHTTPError(err, t.request, t.response)
				return derp.Wrap(err, location, "Parsing error response", body, derp.WithInternalError())
			}
		}

		return derp.NewHTTPError(t.request, t.response)
	}

	// Otherwise this is a success; decode the body into the success object.
	if err := t.decodeResponseBody(body, t.success); err != nil {
		err = derp.WrapHTTPError(err, t.request, t.response)
		return derp.Wrap(err, location, "Processing response body", body, derp.WithInternalError())
	}

	return nil
}

// executeRequest sends the assembled request to the remote server, storing the
// result in t.response. A ModifyRequest option may substitute its own response,
// in which case the network is not contacted.
func (t *Transaction) executeRequest() error {

	const location = "remote.Transaction.executeRequest"

	// A ModifyRequest option may replace the response; if so, use it and skip the network.
	if replacedResponse := t.onModifyRequest(t.request); replacedResponse != nil {
		t.response = replacedResponse
		return nil
	}

	// Otherwise, send the request to the remote server using the assembled client.
	var err error
	t.response, err = t.buildClient().Do(t.request)

	if err != nil {
		err = derp.WrapHTTPError(err, t.request, t.response)
		return derp.Wrap(err, location, "Sending HTTP request", derp.WithInternalError())
	}

	return nil
}

// defaultRequestTimeout bounds a request when the caller has not supplied a
// context (via WithContext) carrying its own deadline.
const defaultRequestTimeout = 1 * time.Minute

// requestContext returns the context for this request and a cancel function that
// must always be called. A caller-supplied context (via WithContext) is used as
// is; otherwise a background context bounded by defaultRequestTimeout is used.
func (t *Transaction) requestContext() (context.Context, context.CancelFunc) {

	if t.ctx != nil {
		return context.WithCancel(t.ctx)
	}

	return context.WithTimeout(context.Background(), defaultRequestTimeout)
}

// buildClient assembles the http.Client used to execute the request. The base
// transport blocks non-public IPs unless AllowPrivateIPs is set; any
// WithRoundTripper middleware wraps that base, so the guard stays underneath.
func (t *Transaction) buildClient() *http.Client {

	var base http.RoundTripper = safeTransport

	if t.allowPrivateIPs {
		base = http.DefaultTransport
	}

	transport := base

	if t.roundTripper != nil {
		transport = t.roundTripper(base)
	}

	return &http.Client{
		Timeout:       defaultTimeout,
		Transport:     transport,
		CheckRedirect: t.checkRedirect,
	}
}

// checkRedirect is the http.Client CheckRedirect policy. It caps the redirect
// chain and re-applies the host allow-list to each redirect target, so an
// allow-listed server cannot redirect the request to a host that is not on the
// list. (The private-IP guard re-runs automatically, since it lives in the dialer.)
func (t *Transaction) checkRedirect(request *http.Request, via []*http.Request) error {

	const location = "remote.Transaction.checkRedirect"

	if len(via) >= maxRedirects {
		return derp.BadRequest(location, "Too many redirects")
	}

	if !t.hostIsAllowed(request.URL.Hostname()) {
		return derp.Forbidden(location, "Redirect to host not in the allow-list", request.URL.Hostname())
	}

	return nil
}

func (t *Transaction) assembleRequest(ctx context.Context) (*http.Request, error) {

	const location = "remote.Transaction.assembleRequest"

	var bodyReader io.Reader

	// Assemble BearCap URLs, if needed.
	if err := t.assembleBearCap(); err != nil {
		return nil, derp.Wrap(err, location, "Assembling BearCap", derp.WithInternalError())
	}

	// Validate the URL before we try to send a request.
	if err := uri.ValidateURL(t.url); err != nil {
		return nil, derp.Wrap(err, location, "Invalid URL", t.url, derp.WithInternalError())
	}

	// If an allow-list is set, confirm the (post-BearCap) host is permitted.
	if err := t.validateAllowedHosts(); err != nil {
		return nil, derp.Wrap(err, location, "Host is not allowed", t.url)
	}

	// GET methods don't have an HTTP Body.  For all other methods,
	// it's time to defined the body content.
	if t.method != http.MethodGet {

		body, err := t.RequestBody()

		if err != nil {
			return nil, derp.Wrap(err, location, "Creating Request Body", t.body, derp.WithInternalError())
		}

		bodyReader = bytes.NewReader(body)
	}

	// Create the HTTP client request, bound to the resolved context.
	result, err := http.NewRequestWithContext(ctx, t.method, t.RequestURL(), bodyReader)

	if err != nil {
		return nil, derp.Wrap(err, location, "Creating HTTP request", derp.WithInternalError())
	}

	// Add headers to httpRequest
	for key, value := range t.header {
		result.Header.Add(key, value)
	}

	return result, nil
}

// validateAllowedHosts confirms that the request URL's host is in the
// transaction's allow-list. An empty allow-list permits any host.
func (t *Transaction) validateAllowedHosts() error {

	const location = "remote.Transaction.validateAllowedHosts"

	if len(t.allowedHosts) == 0 {
		return nil
	}

	parsed, err := url.Parse(t.url)

	if err != nil {
		return derp.Wrap(err, location, "Parsing URL", t.url, derp.WithInternalError())
	}

	if t.hostIsAllowed(parsed.Hostname()) {
		return nil
	}

	return derp.Forbidden(location, "Host is not in the allow-list", parsed.Hostname())
}

// hostIsAllowed reports whether the given host satisfies the transaction's host
// allow-list. An empty allow-list permits any host. Matching is case-insensitive.
func (t *Transaction) hostIsAllowed(host string) bool {

	if len(t.allowedHosts) == 0 {
		return true
	}

	return slices.Contains(t.allowedHosts, strings.ToLower(host))
}

// assembleBearCap pre-processes special bearer capability URLs.
// And of course, there's no "actual" documentation for this, so we're just going to use
// https://docs.joinmastodon.org/spec/bearcaps/ as the canonical source.
func (t *Transaction) assembleBearCap() error {

	const location = "remote.Transaction.assembleBearCap"

	// BearCap URLs are special.  They are used to pass a token to a remote server
	if strings.HasPrefix(t.url, "bear:?") {

		// Split the URL into the bearcap "protocol" and the query string
		_, queryString, _ := strings.Cut(t.url, "?")

		values, err := url.ParseQuery(queryString)

		if err != nil {
			return derp.Wrap(err, location, "Invalid BearCap URL", t.url)
		}

		// Validate the "u" parameter is present
		target := values.Get("u")

		if target == "" {
			return derp.Internal(location, "BearCap URL is required", t.url)
		}

		// Validate the "t" parameter is present
		token := values.Get("t")

		if token == "" {
			return derp.Internal(location, "BearCap Token is required", t.url)
		}

		// Set the correct values in the transaction.
		t.url = target
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
