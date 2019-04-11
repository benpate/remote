package middleware

import (
	"encoding/base64"

	"github.com/benpate/derp"
	"github.com/benpate/remote"
)

// BasicAuth sets the value of a HTTP header for basic Authentication
func BasicAuth(username, password string) Middleware {
	return Middleware{

		Config: func(transaction *remote.Transaction) *derp.Error {
			transaction.Header("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
			return nil
		},
	}
}
