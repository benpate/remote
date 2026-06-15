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

// stubTransport returns a fixed response built around the given body.
type stubTransport struct {
	body io.ReadCloser
}

func (s *stubTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       s.body,
	}, nil
}

func TestSend_ClosesResponseBody(t *testing.T) {
	// Use a stub transport so we can observe the response body being closed.
	body := &trackingBody{Reader: strings.NewReader("ok")}
	client := &http.Client{Transport: &stubTransport{body: body}}

	var result string
	err := Get("http://example.com").Client(client).Result(&result).Send()

	require.Nil(t, err)
	require.Equal(t, "ok", result) // body was read
	require.True(t, body.closed, "Send must close the response body")
}
