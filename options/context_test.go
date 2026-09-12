package options

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/benpate/remote"
	"github.com/stretchr/testify/require"
)

func TestWithContext_ReachesTransaction(t *testing.T) {

	option := WithContext(context.Background())
	require.NotNil(t, option.BeforeRequest)

	// BeforeRequest runs before the transaction resolves its request context,
	// so the context it sets is the one the request is built with.
	transaction := remote.Get("https://example.com")
	require.NoError(t, option.BeforeRequest(transaction))
}

func TestWithContext_CancelAbortsSend(t *testing.T) {

	// The server blocks until the test cancels, so only cancellation can end the request.
	released := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-released
		w.WriteHeader(http.StatusOK)
	}))

	defer server.Close()
	defer close(released)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// AllowPrivateIPs is required because the test server listens on loopback.
	err := remote.Get(server.URL).
		AllowPrivateIPs(true).
		With(WithContext(ctx)).
		Send()

	require.Error(t, err, "expected the cancelled context to abort the request")
}

func TestWithContext_DeadlineBoundsSend(t *testing.T) {

	released := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-released
		w.WriteHeader(http.StatusOK)
	}))

	defer server.Close()
	defer close(released)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// A deadline shorter than remote's own default is what ends this request.
	start := time.Now()
	err := remote.Get(server.URL).
		AllowPrivateIPs(true).
		With(WithContext(ctx)).
		Send()

	require.Error(t, err)
	require.Less(t, time.Since(start), 10*time.Second, "the caller's deadline must bound the request")
}
