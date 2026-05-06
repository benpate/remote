package options

import (
	"github.com/benpate/remote"
)

// UserAgent is remote.Option that adds a HTTP "User-Agent" header to every request.
func UserAgent(userAgent string) remote.Option {

	return remote.Option{

		// This is executed on every transaction before it is compiled into an HTTP request
		BeforeRequest: func(transaction *remote.Transaction) error {
			transaction.Header("User-Agent", userAgent)
			return nil
		},
	}
}
