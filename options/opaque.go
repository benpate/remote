package options

import (
	"net/http"

	"github.com/benpate/remote"
)

// Opaque is a remote.Option that modifies the raw URL string before it is sent to
// the server.  This can be useful when the server requires odd characters in the
// URL string to NOT be urlencoded. (e.g. Such as the REST API for LinkedIn)
//
// Additional documentation can be found at http://golang.org/pkg/net/url/#URL
func Opaque(opaqueValue string) remote.Option {

	return remote.Option{

		// This is executed on every Request before its sent to the server
		ModifyRequest: func(_ *remote.Transaction, request *http.Request) *http.Response {
			request.URL.Opaque = opaqueValue
			return nil
		},
	}
}
