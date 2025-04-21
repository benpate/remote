package options

import (
	"github.com/benpate/remote"
)

// BearerAuth is a remote.Option that sets the Authorization HTTP header witha for Bearer token
func BearerAuth(accessToken string) remote.Option {
	return remote.Option{

		BeforeRequest: func(transaction *remote.Transaction) error {
			transaction.Header("Authorization", "Bearer "+accessToken)
			return nil
		},
	}
}
