package remote

import (
	"net/http"
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

func TestMiddlewareErrors(t *testing.T) {

	server := echoBodyServer()

	// Create a simple middleware to write the transaction to stdout
	configError := Middleware{
		Config: func(transaction *Transaction) *derp.Error {
			return derp.New("Middleware.Config", "Error Running Middleware", 0, nil)
		},
	}

	// Create a simple middleware to write the transaction to stdout
	requestError := Middleware{
		Request: func(request *http.Request) *derp.Error {
			return derp.New("Middleware.Request", "Error Running Middleware", 0, nil)
		},
	}

	// Create a simple middleware to write the transaction to stdout
	responseError := Middleware{
		Response: func(response *http.Response, body *[]byte) *derp.Error {
			return derp.New("Middleware.Response", "Error Running Middleware", 0, nil)
		},
	}

	// We're EXPECTING an error, so if there's no error, then that's BAD.
	if err := Get(server.URL).Use(configError).Send(); err == nil {
		t.Fail()
	}

	// We're EXPECTING an error, so if there's no error, then that's BAD.
	if err := Get(server.URL).Use(requestError).Send(); err == nil {
		t.Fail()
	}

	// We're EXPECTING an error, so if there's no error, then that's BAD.
	if err := Get(server.URL).Use(responseError).Send(); err == nil {
		t.Fail()
	}
}
