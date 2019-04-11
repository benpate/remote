// Package middleware contains several stock / example middleware functions that may be useful
package middleware

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/remote"
)

// Middleware is the data structure implements the remote.Middleware interface, and is used by the
// default middlware provided by this library.
type Middleware struct {
	Config   func(*remote.Transaction) *derp.Error // Config is executed on the Transaction, before it is compiled into an http.Request object
	Request  func(*http.Request) *derp.Error       // Request is executed on the http.Request, before it is sent to the remote server
	Response func(*http.Response) *derp.Error      // Response is executed on the http.Response, before it is parsed into a success or failure object.
}
