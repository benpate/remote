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
	ModifyRequest func(*Transaction, *http.Request) *http.Response

	// AfterRequest is executed after an http.Response has been received from the remote server, but before it is
	// parsed and returned to the calling application.
	AfterRequest func(*Transaction, *http.Response) error
}

// onBeforeRequest executes all "BeforeRequest" option for a transaction.
// It returns an error if any option returns a non-nil response.
func (t *Transaction) onBeforeRequest() error {
	for _, option := range t.options {
		if option.BeforeRequest != nil {
			if err := option.BeforeRequest(t); err != nil {
				return derp.Wrap(err, "remote.Send", "Error executing `config` option")
			}
		}
	}

	return nil
}

// onModifyRequest executes all "ModifyRequest" option for a transaction.
// It returns a new http.Response if any option returns a non-nil response.
func (t *Transaction) onModifyRequest(request *http.Request) *http.Response {

	for _, option := range t.options {
		if option.ModifyRequest != nil {
			if response := option.ModifyRequest(t, request); response != nil {
				return response
			}
		}
	}

	return nil
}

// onAfterRequest executes all "AfterRequest" option for a transaction.
// It returns an error if any option returns a non-nil error.
func (t *Transaction) onAfterRequest(response *http.Response) error {

	for _, option := range t.options {
		if option.AfterRequest != nil {
			if err := option.AfterRequest(t, response); err != nil {
				return derp.Wrap(err, "remote.Send", "Error executing `response` option")
			}
		}
	}

	return nil
}
