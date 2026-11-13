package remote

import (
	"net/http"

	"github.com/benpate/derp"
)

// Option is a decorator that can modify the request before it is sent to the remote HTTP server,
// or modify the response after it is returned by the remote HTTP server.
type Option struct {

	// BeforeRequest is called before an http.Request is generated. It can be used to
	// modify Transaction values before they are assembled into an http.Request object.
	BeforeRequest func(*Transaction) error

	// ModifyRequest is called after an http.Request has been generated, but before it is sent to the
	// remote server. It can be used to modify the request, or to replace it entirely.
	// If it returns a non-nil http.Response, then that is used INSTEAD OF calling the remote server.
	// If it returns a nil http.Response, then the request is sent to the remote server as normal.
	// If it returns an error, then the request is never sent, and no AfterRequest or OnError hook runs.
	ModifyRequest func(*Transaction, *http.Request) (*http.Response, error)

	// AfterRequest is executed after an http.Response has been received from the remote server, but before it is
	// parsed and returned to the calling application.
	//
	// RULE: an AfterRequest hook that replaces response.Body OWNS the reader it
	// replaced. remote closes only the outermost body, so the wrapper's Close
	// must close the reader it wrapped, or the underlying connection leaks.
	AfterRequest func(*Transaction, *http.Response) error

	// OnError is called when the HTTP client returns an error in place of a response to read, such
	// as a refused connection, a DNS or TLS failure, a timeout, or a redirect that remote refused.
	// It receives the client's own error, and returns the error to report, usually the same one.
	OnError func(*Transaction, *http.Request, error) error
}

// onBeforeRequest runs each option's BeforeRequest hook, returning the first error.
func (t *Transaction) onBeforeRequest() error {

	const location = "remote.Transaction.onBeforeRequest"

	for _, option := range t.options {
		if option.BeforeRequest != nil {
			if err := option.BeforeRequest(t); err != nil {
				return derp.Wrap(err, location, "Error executing BeforeRequest option")
			}
		}
	}

	// All hooks agree: proceed
	return nil
}

// onModifyRequest runs each option's ModifyRequest hook, returning the first error or the
// first non-nil response (which replaces the call to the remote server).
func (t *Transaction) onModifyRequest(request *http.Request) (*http.Response, error) {

	const location = "remote.Transaction.onModifyRequest"

	for _, option := range t.options {

		if option.ModifyRequest == nil {
			continue
		}

		response, err := option.ModifyRequest(t, request)

		// RULE: An error wins over a response, so a refused request is never answered
		if err != nil {
			closeResponse(response)
			return nil, derp.Wrap(err, location, "Error executing ModifyRequest option")
		}

		if response != nil {
			return response, nil
		}
	}

	// Send it as it stands
	return nil, nil
}

// closeResponse closes the body of a response that will never be read
func closeResponse(response *http.Response) {

	if response == nil {
		return
	}

	if response.Body == nil {
		return
	}

	_ = response.Body.Close() // Nothing reads this response, so a failed Close has no one to tell
}

// onAfterRequest runs each option's AfterRequest hook, returning the first error.
func (t *Transaction) onAfterRequest(response *http.Response) error {

	const location = "remote.Transaction.onAfterRequest"

	for _, option := range t.options {
		if option.AfterRequest != nil {
			if err := option.AfterRequest(t, response); err != nil {
				return derp.Wrap(err, location, "Error executing AfterRequest option")
			}
		}
	}

	// Nothing to see here, move along
	return nil
}

// onError runs each option's OnError hook in order, handing each one the error that the last returned
func (t *Transaction) onError(request *http.Request, err error) error {

	for _, option := range t.options {

		if option.OnError == nil {
			continue
		}

		// RULE: A hook cannot turn a failed request into a success, because there is no response
		if replaced := option.OnError(t, request, err); replaced != nil {
			err = replaced
		}
	}

	return err
}

// Options extracts valid remote.Option values from a list of `any` values.
func Options(values ...any) []Option {

	result := make([]Option, 0, len(values))

	for _, value := range values {

		if option, isRemoteOption := value.(Option); isRemoteOption {
			result = append(result, option)
		}
	}

	return result
}
