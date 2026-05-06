package options

import (
	"github.com/benpate/remote"
)

// Accept is remote.Option that adds a HTTP "Accept" header to every request.
func Accept(accept string) remote.Option {

	return remote.Option{

		// This is executed on every transaction before it is compiled into an HTTP request
		BeforeRequest: func(transaction *remote.Transaction) error {
			transaction.Header("Accept", accept)
			return nil
		},
	}
}
