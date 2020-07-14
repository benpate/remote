package middleware

import (
	"net/http"

	"github.com/benpate/remote"
)

// Opaque modifies the raw URL string before it is sent to the server.  This can be useful
// when the server requires odd characters in the URL string to NOT be urlencoded.
// (e.g. Such as the REST API for LinkedIn)
//
// Additional documentation can be found at http://golang.org/pkg/net/url/#URL
func Opaque(opaqueValue string) remote.Middleware {

	return remote.Middleware{

		// This is executed on every Request before its sent to the server
		Request: func(r *http.Request) error {
			r.URL.Opaque = opaqueValue
			return nil
		},
	}
}
