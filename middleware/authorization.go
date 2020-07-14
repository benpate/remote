package middleware

import (
	"net/http"

	"github.com/benpate/remote"
)

// Authorization is a sample middleware that adds a HTTP "Authorization" header to every request.
func Authorization(auth string) remote.Middleware {

	return remote.Middleware{

		// This is executed on every transaction before it is compiled into an HTTP request
		Config: func(transaction *remote.Transaction) error {
			transaction.Header("Authorization", auth)
			return nil
		},

		// This is executed on every HTTP request before its sent to the server
		Request: func(_ *http.Request) error {
			return nil
		},

		// This is executed on every HTTP response before it is parsed
		// These functions are empty, and could just be removed from the code.
		Response: func(_ *http.Response, _ *[]byte) error {
			return nil
		},
	}
}
