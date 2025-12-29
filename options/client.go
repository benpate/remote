package options

import (
	"net/http"

	"github.com/benpate/remote"
)

// WithClient is a remote.Option that sets a custom HTTP client for the request.
func WithClient(client *http.Client) remote.Option {
	return remote.Option{
		BeforeRequest: func(txn *remote.Transaction) error {
			txn.Client(client)
			return nil
		},
	}
}
