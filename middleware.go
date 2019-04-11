package remote

import (
	"net/http"

	"github.com/benpate/derp"
)

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

func (t *Transaction) doMiddlewareResponse(response *http.Response) *derp.Error {

	for _, middleware := range t.Middleware {
		if middleware != nil {
			if err := middleware.Response(response); err != nil {
				return derp.New("remote.Send", "Error executing `response` middleware", err, 0, t.getErrorReport())
			}
		}
	}

	return nil
}
