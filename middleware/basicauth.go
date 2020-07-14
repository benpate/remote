package middleware

import (
	"encoding/base64"

	"github.com/benpate/remote"
)

// BasicAuth sets the value of a HTTP header for basic Authentication
func BasicAuth(username, password string) remote.Middleware {
	return remote.Middleware{

		Config: func(transaction *remote.Transaction) error {
			transaction.Header("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
			return nil
		},
	}
}
