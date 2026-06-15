package remote

import (
	"io"
	"net/http"
	"strings"
	"testing"

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
