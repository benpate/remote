package options

import (
	"context"

	"github.com/benpate/remote"
)

// WithContext is a remote.Option that applies a context to the transaction, bounding
// the request by the caller's own deadline and cancellation.
func WithContext(ctx context.Context) remote.Option {

	// A context is request-scoped, so build this Option per request. Storing one on a
	// long-lived client bakes in a deadline that expires once, then fails every request.
	return remote.Option{

		// This is executed on every transaction before it is compiled into an HTTP request
		BeforeRequest: func(transaction *remote.Transaction) error {
			transaction.WithContext(ctx)
			return nil
		},
	}
}
