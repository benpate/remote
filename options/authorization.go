package options

import (
	"github.com/benpate/remote"
)

// Authorization is remote.Option that adds a HTTP "Authorization" header to every request.
func Authorization(auth string) remote.Option {

	return remote.Option{

		// This is executed on every transaction before it is compiled into an HTTP request
		BeforeRequest: func(transaction *remote.Transaction) error {
			transaction.Header("Authorization", auth)
			return nil
		},
	}
}
