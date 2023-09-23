package options

import (
	"encoding/base64"

	"github.com/benpate/remote"
)

// BasicAuth is a remote.Option that sets the value of a HTTP header for basic Authentication
func BasicAuth(username, password string) remote.Option {
	return remote.Option{

		BeforeRequest: func(transaction *remote.Transaction) error {
			transaction.Header("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
			return nil
		},
	}
}
