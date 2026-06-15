package remote

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// jsonServer returns an httptest server that responds with the given status code
// and JSON body.
func jsonServer(statusCode int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(ContentType, ContentTypeJSON)
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
}

func TestSend_Success(t *testing.T) {

	ts := jsonServer(200, `{"name":"Sarah","age":42}`)
	defer ts.Close()

	result := map[string]any{}
	err := Get(ts.URL).AllowPrivateIPs(true).Result(&result).Send()

	require.NoError(t, err)
	require.Equal(t, "Sarah", result["name"])
	require.Equal(t, float64(42), result["age"])
}

func TestSend_ErrorStatusWithFailureObject(t *testing.T) {

	ts := jsonServer(404, `{"error":"not found"}`)
	defer ts.Close()

	success := map[string]any{}
	failure := map[string]any{}

	err := Get(ts.URL).AllowPrivateIPs(true).Result(&success).Error(&failure).Send()

	require.Error(t, err)
	require.Equal(t, "not found", failure["error"])
}

func TestSend_ErrorStatusNoFailureObject(t *testing.T) {

	ts := jsonServer(500, `{"error":"boom"}`)
	defer ts.Close()

	err := Get(ts.URL).AllowPrivateIPs(true).Send()
	require.Error(t, err)
}

func TestSend_ErrorStatusBadFailureBody(t *testing.T) {

	// Error status with an unparseable body and a JSON failure target -> parse error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(ContentType, ContentTypeJSON)
		w.WriteHeader(400)
		_, _ = w.Write([]byte("not json"))
	}))
	defer ts.Close()

	failure := map[string]any{}
	err := Get(ts.URL).AllowPrivateIPs(true).Error(&failure).Send()
	require.Error(t, err)
}

func TestSend_InvalidURL(t *testing.T) {
	err := Get("not a valid url").Send()
	require.Error(t, err)
}

func TestSend_NetworkError(t *testing.T) {
	// A well-formed URL pointing at a closed port produces a transport error
	err := Get("http://127.0.0.1:0").AllowPrivateIPs(true).Send()
	require.Error(t, err)
}

func TestSend_BeforeRequestError(t *testing.T) {

	ts := jsonServer(200, `{}`)
	defer ts.Close()

	option := Option{
		BeforeRequest: func(*Transaction) error {
			return errSendTest
		},
	}

	err := Get(ts.URL).AllowPrivateIPs(true).With(option).Send()
	require.Error(t, err)
}

func TestSend_AfterRequestError(t *testing.T) {

	ts := jsonServer(200, `{}`)
	defer ts.Close()

	option := Option{
		AfterRequest: func(*Transaction, *http.Response) error {
			return errSendTest
		},
	}

	err := Get(ts.URL).AllowPrivateIPs(true).With(option).Send()
	require.Error(t, err)
}

func TestSend_ModifyRequestReplacesResponse(t *testing.T) {

	// A ModifyRequest option that returns a response short-circuits the network call.
	// We point at a closed port to prove the network is never touched.
	called := false
	option := Option{
		ModifyRequest: func(_ *Transaction, _ *http.Request) *http.Response {
			called = true
			return &http.Response{
				StatusCode: 200,
				Header:     http.Header{ContentType: []string{ContentTypeJSON}},
				Body:       io.NopCloser(strings.NewReader(`{"replaced":true}`)),
			}
		},
	}

	result := map[string]any{}
	err := Get("http://127.0.0.1:0").AllowPrivateIPs(true).With(option).Result(&result).Send()

	require.NoError(t, err)
	require.True(t, called)
	require.Equal(t, true, result["replaced"])
}

func TestSend_UnsupportedRequestContentType(t *testing.T) {

	ts := jsonServer(200, `{}`)
	defer ts.Close()

	// A POST with an XML content type fails when assembling the request body
	err := Post(ts.URL).AllowPrivateIPs(true).ContentType(ContentTypeXML).Send()
	require.Error(t, err)
}

func TestSend_PostFormRoundTrip(t *testing.T) {

	var receivedContentType string
	var receivedBody string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get(ContentType)
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		receivedBody = string(buf)
		w.WriteHeader(200)
	}))
	defer ts.Close()

	err := Post(ts.URL).AllowPrivateIPs(true).Form("a", "1").Form("b", "2").Send()
	require.NoError(t, err)
	require.Equal(t, ContentTypeForm, receivedContentType)
	require.Equal(t, "a=1&b=2", receivedBody)
}

var errSendTest = errTestRemote("synthetic option error")

type errTestRemote string

func (e errTestRemote) Error() string { return string(e) }
