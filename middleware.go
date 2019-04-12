package remote

import (
	"net/http"

	"github.com/benpate/derp"
)

// Middleware is a decorator that can modify the request before it is sent to the remote HTTP server,
// or modify the response after it is returned by the remote HTTP server.
type Middleware interface {
	Config(*Transaction) *derp.Error
	Request(*http.Request) *derp.Error
	Response(*http.Response, *[]byte) *derp.Error
}

func (t *Transaction) doMiddlewareConfig() *derp.Error {
	for _, middleware := range t.Middleware {
		if middleware != nil {
			if err := middleware.Config(t); err != nil {
				return derp.New("remote.Send", "Error executing `config` middleware", err, 0, t.getErrorReport())
			}
		}
	}

	return nil
}

func (t *Transaction) doMiddlewareRequest(request *http.Request) *derp.Error {

	for _, middleware := range t.Middleware {
		if middleware != nil {
			if err := middleware.Request(request); err != nil {
				return derp.New("remote.Send", "Error executing `request` middleware", err, 0, t.getErrorReport())
			}
		}
	}

	return nil

}

func (t *Transaction) doMiddlewareResponse(response *http.Response, body *[]byte) *derp.Error {

	for _, middleware := range t.Middleware {
		if middleware != nil {
			if err := middleware.Response(response, body); err != nil {
				return derp.New("remote.Send", "Error executing `response` middleware", err, 0, t.getErrorReport())
			}
		}
	}

	return nil
}
