package remote

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithContext_Success(t *testing.T) {
	body := []byte("ok")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	var result string
	err := Get(server.URL).AllowPrivateIPs(true).WithContext(context.Background()).Result(&result).Send()

	require.Nil(t, err)
	require.Equal(t, "ok", result)
}

func TestWithContext_Cancelled(t *testing.T) {
	// A cancelled context aborts the request before the server is contacted.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("server should not be contacted with a cancelled context")
	}))
	t.Cleanup(server.Close)

	err := Get(server.URL).WithContext(ctx).Send()
	require.Error(t, err)
}

func TestWithContext_DeadlineExceeded(t *testing.T) {
	// An already-expired deadline aborts the request before the server is contacted.
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Hour))
	t.Cleanup(cancel)

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("server should not be contacted with an expired deadline")
	}))
	t.Cleanup(server.Close)

	err := Get(server.URL).WithContext(ctx).Send()
	require.Error(t, err)
}

func TestRequestContext_DefaultTimeout(t *testing.T) {
	// With no context set, the request is bounded by the default one-minute timeout.
	ctx, cancel := New().requestContext()
	t.Cleanup(cancel)

	deadline, ok := ctx.Deadline()
	require.True(t, ok)
	require.WithinDuration(t, time.Now().Add(defaultRequestTimeout), deadline, 5*time.Second)
	require.Equal(t, time.Minute, defaultRequestTimeout)
}

func TestRequestContext_HonorsCallerContext(t *testing.T) {
	// A caller-supplied context is used as-is (no default timeout imposed), and
	// cancelling the parent cancels the resolved context.
	parent, cancelParent := context.WithCancel(context.Background())
	t.Cleanup(cancelParent)

	ctx, cancel := New().WithContext(parent).requestContext()
	t.Cleanup(cancel)

	_, ok := ctx.Deadline()
	require.False(t, ok)

	cancelParent()

	select {
	case <-ctx.Done():
		// success
	case <-time.After(time.Second):
		t.Fatal("expected the resolved context to be cancelled with its parent")
	}
}

func TestWithContext_DefaultWithoutContext(t *testing.T) {
	// Without WithContext, the request still works (context.Background is used).
	body := []byte("default")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)

	var result string
	require.Nil(t, Get(server.URL).AllowPrivateIPs(true).Result(&result).Send())
	require.Equal(t, "default", result)
}
