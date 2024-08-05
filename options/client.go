package options

import (
	"net/http"

	"github.com/benpate/remote"
)

func WithClient(client *http.Client) remote.Option {
	return remote.Option{
		BeforeRequest: func(txn *remote.Transaction) error {
			txn.Client(client)
			return nil
		},
	}
}
