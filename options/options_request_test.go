package options

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/benpate/remote"
	"github.com/stretchr/testify/require"
)

func TestOpaque(t *testing.T) {

	option := Opaque("//custom/opaque/path")
	require.NotNil(t, option.ModifyRequest)

	request, err := http.NewRequest(http.MethodGet, "http://example.com/normal", nil)
	require.NoError(t, err)

	// ModifyRequest returns nil (request is sent normally) but mutates the URL
	response := option.ModifyRequest(nil, request)
	require.Nil(t, response)
	require.Equal(t, "//custom/opaque/path", request.URL.Opaque)
}

func TestDebug(t *testing.T) {

	option := Debug()
	require.NotNil(t, option.ModifyRequest)
	require.NotNil(t, option.AfterRequest)

	// ModifyRequest dumps the request and returns nil (does not replace it)
	request, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)
	require.Nil(t, option.ModifyRequest(nil, request))

	// AfterRequest dumps the response and returns no error
	response := &http.Response{
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		Header:     http.Header{},
		Body:       http.NoBody,
	}
	require.NoError(t, option.AfterRequest(nil, response))
}

func TestWithClient(t *testing.T) {

	// A custom client with a recording transport proves WithClient swaps the client.
	recorder := &recordingTransport{}
	customClient := &http.Client{Transport: recorder}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()

	err := remote.Get(ts.URL).With(WithClient(customClient)).Send()
	require.NoError(t, err)
	require.True(t, recorder.called, "expected the custom client's transport to be used")
}

// recordingTransport records whether it was invoked, then delegates to the default transport.
type recordingTransport struct {
	called bool
}

func (rt *recordingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	rt.called = true
	return http.DefaultTransport.RoundTrip(request)
}

func TestTestServer_Match(t *testing.T) {

	// A raw HTTP response stored in the filesystem is returned for matching hosts.
	rawResponse := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 15\r\n" +
		"\r\n" +
		`{"hello":"yes"}`

	filesystem := fstest.MapFS{
		"users/1.json": &fstest.MapFile{Data: []byte(rawResponse)},
	}

	option := TestServer("example.com", filesystem)

	request, err := http.NewRequest(http.MethodGet, "http://example.com/users/1.json", nil)
	require.NoError(t, err)

	response := option.ModifyRequest(nil, request)
	require.NotNil(t, response)
	require.Equal(t, 200, response.StatusCode)
}

func TestTestServer_HostnameMismatch(t *testing.T) {

	option := TestServer("example.com", fstest.MapFS{})

	request, err := http.NewRequest(http.MethodGet, "http://other.com/file.json", nil)
	require.NoError(t, err)

	// A non-matching hostname passes through (returns nil)
	require.Nil(t, option.ModifyRequest(nil, request))
}

func TestTestServer_FileNotFound(t *testing.T) {

	option := TestServer("example.com", fstest.MapFS{})

	request, err := http.NewRequest(http.MethodGet, "http://example.com/missing.json", nil)
	require.NoError(t, err)

	// A missing file produces a 404 response
	response := option.ModifyRequest(nil, request)
	require.NotNil(t, response)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestTestServer_InvalidResponse(t *testing.T) {

	// A file that does not contain a valid raw HTTP response produces a 404
	filesystem := fstest.MapFS{
		"bad.json": &fstest.MapFile{Data: []byte("this is not an http response")},
	}

	option := TestServer("example.com", filesystem)

	request, err := http.NewRequest(http.MethodGet, "http://example.com/bad.json", nil)
	require.NoError(t, err)

	response := option.ModifyRequest(nil, request)
	require.NotNil(t, response)
	require.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestTestServer_Integration(t *testing.T) {

	// End-to-end: a remote.Get against the mocked host returns the canned body.
	rawResponse := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 16\r\n" +
		"\r\n" +
		`{"name":"mocky"}`

	filesystem := fstest.MapFS{
		"data.json": &fstest.MapFile{Data: []byte(rawResponse)},
	}

	result := map[string]any{}
	err := remote.Get("http://example.com/data.json").
		With(TestServer("example.com", filesystem)).
		Result(&result).
		Send()

	require.NoError(t, err)
	require.Equal(t, "mocky", result["name"])
}
