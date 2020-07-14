package remote

import (
	"net/http"

	"github.com/benpate/derp"
)

// Middleware is a decorator that can modify the request before it is sent to the remote HTTP server,
// or modify the response after it is returned by the remote HTTP server.
type Middleware struct {
	Config   func(*Transaction) error            // Config is executed on the Transaction, before it is compiled into an http.Request object
	Request  func(*http.Request) error           // Request is executed on the http.Request, before it is sent to the remote server
	Response func(*http.Response, *[]byte) error // Response is executed on the http.Response, before it is parsed into a success or failure object.
}

func (t *Transaction) doMiddlewareConfig() error {
	for _, middleware := range t.Middleware {
		if middleware.Config != nil {
			if err := middleware.Config(t); err != nil {
				return derp.Wrap(err, "remote.Send", "Error executing `config` middleware", t.ErrorReport())
			}
		}
	}

	return nil
}

func (t *Transaction) doMiddlewareRequest(request *http.Request) error {

	for _, middleware := range t.Middleware {
		if middleware.Request != nil {
			if err := middleware.Request(request); err != nil {
				return derp.Wrap(err, "remote.Send", "Error executing `request` middleware", t.ErrorReport())
			}
		}
	}

	return nil
}

func (t *Transaction) doMiddlewareResponse(response *http.Response, body *[]byte) error {

	for _, middleware := range t.Middleware {
		if middleware.Response != nil {
			if err := middleware.Response(response, body); err != nil {
				return derp.Wrap(err, "remote.Send", "Error executing `response` middleware", t.ErrorReport())
			}
		}
	}

	return nil
}
