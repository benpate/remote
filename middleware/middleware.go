package middleware

import (
	"net/http"

	"github.com/benpate/derp"
	"github.com/benpate/remote"
)

// Middleware is the data structure implements the remote.Middleware interface, and is used by the
// default middlware provided by this library.
type Middleware struct {
	Config   func(*remote.Transaction) *derp.Error
	Request  func(*http.Request) *derp.Error
	Response func(*http.Response) *derp.Error
}
