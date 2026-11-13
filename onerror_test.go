package remote

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// unreachableURL names a port that nothing listens on, so the request fails without a response
const unreachableURL = "http://127.0.0.1:0"

// TestSend_OnError_SeesTransportFailure verifies that OnError receives the request and the
// client's own error when no response arrives, and that AfterRequest does not run
func TestSend_OnError_SeesTransportFailure(t *testing.T) {

	var seenRequest *http.Request
	var seenError error
	afterRequestRan := false

	option := Option{
		AfterRequest: func(_ *Transaction, _ *http.Response) error {
			afterRequestRan = true
			return nil
		},
		OnError: func(_ *Transaction, request *http.Request, err error) error {
			seenRequest = request
			seenError = err
			return err
		},
	}

	err := Get(unreachableURL).AllowPrivateIPs(true).With(option).Send()

	require.Error(t, err)
	require.Error(t, seenError)
	require.NotNil(t, seenRequest)
	require.Equal(t, "127.0.0.1:0", seenRequest.URL.Host)
	require.False(t, afterRequestRan, "there is no response for AfterRequest to read")
	require.ErrorIs(t, err, seenError, "the error the hook saw is still in the chain")
}

// TestSend_OnError_CanReplaceTheError verifies that the error a hook returns is the one reported
func TestSend_OnError_CanReplaceTheError(t *testing.T) {

	replacement := errors.New("replaced by the hook")

	option := Option{
		OnError: func(_ *Transaction, _ *http.Request, _ error) error {
			return replacement
		},
	}

	err := Get(unreachableURL).AllowPrivateIPs(true).With(option).Send()
	require.ErrorIs(t, err, replacement)
}

// TestSend_OnError_CannotClearTheError verifies that a hook returning nil leaves the failure in place
func TestSend_OnError_CannotClearTheError(t *testing.T) {

	option := Option{
		OnError: func(_ *Transaction, _ *http.Request, _ error) error {
			return nil
		},
	}

	err := Get(unreachableURL).AllowPrivateIPs(true).With(option).Send()
	require.Error(t, err)
}

// TestSend_OnError_RunsInOrder verifies that each hook receives the error that the one before it returned
func TestSend_OnError_RunsInOrder(t *testing.T) {

	first := errors.New("from the first hook")
	var secondSaw error

	options := []Option{
		{OnError: func(_ *Transaction, _ *http.Request, _ error) error { return first }},
		{}, // An option with no OnError hook is skipped
		{OnError: func(_ *Transaction, _ *http.Request, err error) error {
			secondSaw = err
			return err
		}},
	}

	err := Get(unreachableURL).AllowPrivateIPs(true).With(options...).Send()

	require.ErrorIs(t, err, first)
	require.Equal(t, first, secondSaw)
}

// TestSend_OnError_NotCalledForAResponse verifies that a response, even an error response,
// goes to AfterRequest and not to OnError
func TestSend_OnError_NotCalledForAResponse(t *testing.T) {

	for _, statusCode := range []int{200, 404, 429, 500} {

		server := jsonServer(statusCode, `{}`)
		onErrorRan := false
		afterRequestRan := false

		option := Option{
			AfterRequest: func(_ *Transaction, _ *http.Response) error {
				afterRequestRan = true
				return nil
			},
			OnError: func(_ *Transaction, _ *http.Request, err error) error {
				onErrorRan = true
				return err
			},
		}

		_ = Get(server.URL).AllowPrivateIPs(true).With(option).Send()
		server.Close()

		require.True(t, afterRequestRan, "status %d", statusCode)
		require.False(t, onErrorRan, "status %d", statusCode)
	}
}

// TestSend_OnError_NotCalledWhenBeforeRequestRefuses verifies that a request refused by an
// option was never sent, so no hook hears about a failure
func TestSend_OnError_NotCalledWhenBeforeRequestRefuses(t *testing.T) {

	refusal := errors.New("refused")
	onErrorRan := false

	option := Option{
		BeforeRequest: func(_ *Transaction) error { return refusal },
		OnError: func(_ *Transaction, _ *http.Request, err error) error {
			onErrorRan = true
			return err
		},
	}

	err := Get(unreachableURL).AllowPrivateIPs(true).With(option).Send()

	require.ErrorIs(t, err, refusal)
	require.False(t, onErrorRan)
}
