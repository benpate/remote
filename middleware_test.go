package remote

import (
	"testing"

	"github.com/benpate/derp"
)

func TestMiddleware(t *testing.T) {

	// Create a simple middleware to write the transaction to stdout
	middleware := Middleware{
		Config: func(transaction *Transaction) *derp.Error {
			t.Log(transaction.URLValue)
			return nil
		},
	}

	server := echoBodyServer()

	Get(server.URL).Use(middleware).Send()
}

func TestMiddlewareError(t *testing.T) {

	// Create a simple middleware to write the transaction to stdout
	middleware := Middleware{
		Config: func(transaction *Transaction) *derp.Error {
			return derp.New("Middleware.Config", "Error Running Middleware", 0, nil).Report()
		},
	}

	server := echoBodyServer()

	// We're EXPECTING an error, so if there's no error, then that's BAD.
	if err := Get(server.URL).Use(middleware); err == nil {
		t.Fail()
	}
}
