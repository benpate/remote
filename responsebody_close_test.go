package remote

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// trackingBody is a ReadCloser that records whether Close was called.
type trackingBody struct {
	io.Reader
	closed bool
}

func (b *trackingBody) Close() error {
	b.closed = true
	return nil
}

// rtFunc adapts a function to the http.RoundTripper interface.
type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestSend_ClosesResponseBody(t *testing.T) {
	// A middleware returns a fixed response with a body we can observe; it does
	// not delegate to "next", so no real connection is made.
	body := &trackingBody{Reader: strings.NewReader("ok")}

	middleware := func(http.RoundTripper) http.RoundTripper {
		return rtFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: body}, nil
		})
	}

	var result string
	err := Get("http://example.com").WithRoundTripper(middleware).Result(&result).Send()

	require.Nil(t, err)
	require.Equal(t, "ok", result) // body was read
	require.True(t, body.closed, "Send must close the response body")
}

// wrappedBody is a ReadCloser that records its own Close, and closes the
// reader it wrapped -- the contract AfterRequest hooks must honor.
type wrappedBody struct {
	io.Reader
	inner  io.ReadCloser
	closed bool
}

func (b *wrappedBody) Close() error {
	b.closed = true
	return b.inner.Close()
}

func TestSend_ClosesAfterRequestWrapper(t *testing.T) {

	// An AfterRequest hook replaces the body AFTER the response arrives. Send
	// must close that wrapper, not the body it replaced.
	inner := &trackingBody{Reader: strings.NewReader("ok")}
	var wrapper *wrappedBody

	middleware := func(http.RoundTripper) http.RoundTripper {
		return rtFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: inner}, nil
		})
	}

	capture := Option{
		AfterRequest: func(_ *Transaction, response *http.Response) error {
			wrapper = &wrappedBody{Reader: response.Body, inner: response.Body}
			response.Body = wrapper
			return nil
		},
	}

	var result string
	err := Get("http://example.com").WithRoundTripper(middleware).With(capture).Result(&result).Send()

	require.Nil(t, err)
	require.Equal(t, "ok", result)
	require.NotNil(t, wrapper)
	require.True(t, wrapper.closed, "Send must close the body installed by AfterRequest")
	require.True(t, inner.closed, "the wrapper must close the body it replaced")
}

func TestSend_ClosesResponseBodyWhenAfterRequestFails(t *testing.T) {

	// An AfterRequest hook that errors returns before the deferred Close is
	// registered, so Send must release the body on that path too.
	body := &trackingBody{Reader: strings.NewReader("ok")}

	middleware := func(http.RoundTripper) http.RoundTripper {
		return rtFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: body}, nil
		})
	}

	failing := Option{
		AfterRequest: func(*Transaction, *http.Response) error {
			return derp.Internal("test", "AfterRequest failed")
		},
	}

	err := Get("http://example.com").WithRoundTripper(middleware).With(failing).Send()

	require.NotNil(t, err)
	require.True(t, body.closed, "Send must close the response body when AfterRequest fails")
}
