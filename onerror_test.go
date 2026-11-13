package remote

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

// onErrorCounter returns an Option that counts its OnError and AfterRequest calls
func onErrorCounter(onErrorCalls *int, afterRequestCalls *int) Option {
	return Option{
		AfterRequest: func(_ *Transaction, _ *http.Response) error {
			*afterRequestCalls++
			return nil
		},
		OnError: func(_ *Transaction, _ *http.Request, err error) error {
			*onErrorCalls++
			return err
		},
	}
}

// redirectServer returns a test server that answers every request with a redirect to the target
func redirectServer(t *testing.T, target string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, target, http.StatusFound)
	}))

	t.Cleanup(server.Close)
	return server
}

// TestSend_OnError_CalledWhenRedirectsRunOut verifies that a redirect chain cut off at the cap
// goes to OnError, and that the transaction still holds the last redirect response
func TestSend_OnError_CalledWhenRedirectsRunOut(t *testing.T) {

	// This server redirects to itself forever
	server := redirectServer(t, "/again")

	onErrorCalls, afterRequestCalls := 0, 0
	var seenResponse *http.Response

	inspector := Option{
		OnError: func(transaction *Transaction, _ *http.Request, err error) error {
			seenResponse = transaction.Response()
			return err
		},
	}

	err := Get(server.URL).
		AllowPrivateIPs(true).
		With(onErrorCounter(&onErrorCalls, &afterRequestCalls), inspector).
		Send()

	require.Error(t, err)
	require.Equal(t, 1, onErrorCalls, "the whole chain is one failure, not one per hop")
	require.Zero(t, afterRequestCalls, "a refused redirect never reaches AfterRequest")

	// This is the one failure that leaves a response behind: the redirect that was not followed
	require.NotNil(t, seenResponse)
	require.Equal(t, http.StatusFound, seenResponse.StatusCode)
}

// TestSend_OnError_CalledWhenRedirectLeavesAllowList verifies that a redirect to a host
// outside the allow-list goes to OnError
func TestSend_OnError_CalledWhenRedirectLeavesAllowList(t *testing.T) {

	server := redirectServer(t, "http://elsewhere.example/")
	onErrorCalls, afterRequestCalls := 0, 0

	err := Get(server.URL).
		AllowPrivateIPs(true).
		AllowHosts("127.0.0.1").
		With(onErrorCounter(&onErrorCalls, &afterRequestCalls)).
		Send()

	require.Error(t, err)
	require.Equal(t, 1, onErrorCalls)
	require.Zero(t, afterRequestCalls)
}

// TestSend_OnError_CalledWhenPrivateIPGuardRefuses verifies that a dial refused by remote's
// own private-IP guard goes to OnError, like any other failed connection
func TestSend_OnError_CalledWhenPrivateIPGuardRefuses(t *testing.T) {

	// The handler must never run, because the guard refuses the dial
	handlerRan := false
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		handlerRan = true
	}))
	t.Cleanup(server.Close)

	onErrorCalls, afterRequestCalls := 0, 0

	// AllowPrivateIPs is left off, so the guard is active
	err := Get(server.URL).
		With(onErrorCounter(&onErrorCalls, &afterRequestCalls)).
		Send()

	require.Error(t, err)
	require.False(t, handlerRan)
	require.Equal(t, 1, onErrorCalls)
	require.Zero(t, afterRequestCalls)
}

// TestSend_OnError_CalledWhenContextIsCanceled verifies that a canceled request goes to OnError,
// and that the hook can recognize it with errors.Is
func TestSend_OnError_CalledWhenContextIsCanceled(t *testing.T) {

	server := jsonServer(http.StatusOK, `{}`)
	t.Cleanup(server.Close)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var seenError error

	option := Option{
		OnError: func(_ *Transaction, _ *http.Request, err error) error {
			seenError = err
			return err
		},
	}

	err := Get(server.URL).AllowPrivateIPs(true).WithContext(ctx).With(option).Send()

	require.Error(t, err)
	require.ErrorIs(t, seenError, context.Canceled, "a hook that counts failures can skip this one")
}

// TestSend_OnError_NotCalledWhenRequestCannotBeBuilt verifies that a URL that does not parse
// fails before anything is sent, so no hook hears about a failure
func TestSend_OnError_NotCalledWhenRequestCannotBeBuilt(t *testing.T) {

	onErrorCalls, afterRequestCalls := 0, 0

	err := Get("http://[::1").
		AllowPrivateIPs(true).
		With(onErrorCounter(&onErrorCalls, &afterRequestCalls)).
		Send()

	require.Error(t, err)
	require.Zero(t, onErrorCalls)
	require.Zero(t, afterRequestCalls)
}
