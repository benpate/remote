package remote_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
)

// jsonServer returns a test server that answers every request with the given
// status and JSON body. Examples use it so that they run offline and print the
// same thing every time; a real call would name a public URL instead.
func jsonServer(status int, body string) *httptest.Server {

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(remote.ContentType, remote.ContentTypeJSON)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

// Get a JSON document from a remote server, decoded into a struct.
func ExampleGet() {

	server := jsonServer(http.StatusOK, `{"id":"ABC123","name":"Sarah Connor"}`)
	defer server.Close()

	user := struct {
		ID   string
		Name string
	}{}

	// AllowPrivateIPs is needed only because the test server listens on loopback.
	// A request to a public URL does not need it -- and should not use it.
	err := remote.Get(server.URL).
		AllowPrivateIPs(true).
		Result(&user).
		Send()

	fmt.Println(err)
	fmt.Println(user.ID, user.Name)
	// Output:
	// <nil>
	// ABC123 Sarah Connor
}

// Post a JSON document, and decode the server's reply.
func ExamplePost() {

	server := jsonServer(http.StatusOK, `{"status":"created"}`)
	defer server.Close()

	user := map[string]string{
		"name":  "Sarah Connor",
		"email": "sarah@sky.net",
	}

	response := struct {
		Status string
	}{}

	err := remote.Post(server.URL).
		AllowPrivateIPs(true).
		JSON(user).        // encode the user into the request body as JSON
		Result(&response). // decode a successful reply into this struct
		Send()

	fmt.Println(err)
	fmt.Println(response.Status)
	// Output:
	// <nil>
	// created
}

// Send returns an error for any status outside 200-299. When the server
// describes the problem in the response body, Error names the object to
// decode that body into.
func ExampleTransaction_Error() {

	server := jsonServer(http.StatusNotFound, `{"message":"No such user"}`)
	defer server.Close()

	success := struct{ Name string }{}
	failure := struct{ Message string }{}

	err := remote.Get(server.URL).
		AllowPrivateIPs(true).
		Result(&success).
		Error(&failure).
		Send()

	fmt.Println(err != nil)
	fmt.Println(failure.Message)
	// Output:
	// true
	// No such user
}

// Query adds parameters to the URL's query string.
func ExampleTransaction_Query() {

	transaction := remote.Get("https://example.com/search").
		Query("q", "hello world").
		Query("page", "2")

	fmt.Println(transaction.RequestURL())
	// Output: https://example.com/search?page=2&q=hello+world
}

// Accept lists acceptable content types in preference order, assigning each a
// descending q-value.
func ExampleTransaction_Accept() {

	// Echo the Accept header back, so the example can show what was sent.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.Header.Get(remote.Accept)))
	}))
	defer server.Close()

	var sent string
	err := remote.Get(server.URL).
		AllowPrivateIPs(true).
		Accept(remote.ContentTypeActivityPub, remote.ContentTypeJSONLD, remote.ContentTypeJSON).
		Result(&sent).
		Send()

	fmt.Println(err)
	fmt.Println(sent)
	// Output:
	// <nil>
	// application/activity+json;q=1.0, application/ld+json;q=0.9, application/json;q=0.8
}

// AllowHosts restricts a transaction to a known set of servers, which matters
// most when the URL comes from user input. A bare entry matches the host on any
// port; an entry that names a port matches only that port.
func ExampleTransaction_AllowHosts() {

	server := jsonServer(http.StatusOK, `{"ok":true}`)
	defer server.Close()

	allowed := remote.Get(server.URL).
		AllowPrivateIPs(true).
		AllowHosts("127.0.0.1").
		Send()

	// A host that is not on the list fails before any connection is made.
	blocked := remote.Get(server.URL).
		AllowPrivateIPs(true).
		AllowHosts("api.example.com", "cdn.example.com:8443").
		Send()

	fmt.Println(allowed)
	fmt.Println(blocked != nil)
	// Output:
	// <nil>
	// true
}

// Options carry hooks that observe or rewrite a transaction as it runs. The
// options subpackage ships the common ones.
func ExampleTransaction_With() {

	server := jsonServer(http.StatusOK, `{"ok":true}`)
	defer server.Close()

	var seen string

	err := remote.Get(server.URL).
		AllowPrivateIPs(true).
		With(options.BearerAuth("secret-token")).
		With(remote.Option{
			ModifyRequest: func(_ *remote.Transaction, request *http.Request) *http.Response {
				seen = request.Header.Get("Authorization")
				return nil // returning nil sends the request as normal
			},
		}).
		Send()

	fmt.Println(err)
	fmt.Println(seen)
	// Output:
	// <nil>
	// Bearer secret-token
}

// MaxResponseSize caps how many bytes are read from the response, so an
// untrusted server cannot exhaust memory. A larger body is an error.
func ExampleTransaction_MaxResponseSize() {

	server := jsonServer(http.StatusOK, `{"padding":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`)
	defer server.Close()

	var result []byte
	err := remote.Get(server.URL).
		AllowPrivateIPs(true).
		MaxResponseSize(10).
		Result(&result).
		Send()

	fmt.Println(err != nil)
	// Output: true
}

// NewHTTPClient hands remote's SSRF-hardened transport to a library that takes
// an *http.Client but not a remote.Transaction -- golang.org/x/oauth2, say.
func ExampleNewHTTPClient() {

	client := remote.NewHTTPClient(false)

	// The guard lives in the transport, so it applies to every request the
	// borrowing library makes, including each redirect hop.
	_, err := client.Get("http://169.254.169.254/latest/meta-data/")

	fmt.Println(err != nil)
	// Output: true
}
