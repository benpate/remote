package remote

import (
	"net/http"
	"testing"

	"github.com/benpate/derp"
)

func TestOption(t *testing.T) {

	// Create a simple middleware to write the transaction to stdout
	middleware := Option{
		BeforeRequest: func(transaction *Transaction) error {
			t.Log(transaction.url)
			return nil
		},
	}

	server := echoBodyServer()

	// nolint:errcheck // just a test
	Get(server.URL).With(middleware).Send()
}

func TestOptionErrors(t *testing.T) {

	server := echoBodyServer()

	// Create a simple middleware to write the transaction to stdout
	configError := Option{
		BeforeRequest: func(transaction *Transaction) error {
			return derp.InternalError("Option.Config", "Error Running Option")
		},
	}

	// Create a simple middleware to write the transaction to stdout
	responseError := Option{
		AfterRequest: func(transaction *Transaction, response *http.Response) error {
			return derp.InternalError("Option.Response", "Error Running Option")
		},
	}

	// We're EXPECTING an error, so if there's no error, then that's BAD.
	if err := Get(server.URL).With(configError).Send(); err == nil {
		t.Fail()
	}

	// We're EXPECTING an error, so if there's no error, then that's BAD.
	if err := Get(server.URL).With(responseError).Send(); err == nil {
		t.Fail()
	}
}
