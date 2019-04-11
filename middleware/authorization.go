package middleware

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/remote"
)

// Authorization is a sample Interceptor that adds a HTTP "Authorization" header to every request.
func Authorization(auth string) Middleware {

	return Middleware{

		Config: func(transaction *remote.Transaction) *derp.Error {
			transaction.Header("Authorization", auth)
			return nil
		},

		// This is executed on every Request before its sent to the server
		Request: func(_ *http.Request) *derp.Error {
			return nil
		},

		Response: func(_ *http.Response) *derp.Error {
			return nil
		},
	}
}
