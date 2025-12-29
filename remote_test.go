package remote

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

func TestRequestMethods(t *testing.T) {

	getTxn := Get("someurl.com")
	require.Equal(t, "GET", getTxn.method)
	require.Equal(t, "someurl.com", getTxn.url)

	postTxn := Post("someurl.com")
	require.Equal(t, "POST", postTxn.method)
	require.Equal(t, "someurl.com", postTxn.url)

	putTxn := Put("someurl.com")
	require.Equal(t, "PUT", putTxn.method)
	require.Equal(t, "someurl.com", putTxn.url)

	patchTxn := Patch("someurl.com")
	require.Equal(t, "PATCH", patchTxn.method)
	require.Equal(t, "someurl.com", patchTxn.url)

	deleteTxn := Delete("someurl.com")
	require.Equal(t, "DELETE", deleteTxn.method)
	require.Equal(t, "someurl.com", deleteTxn.url)
}

func TestGet(t *testing.T) {

	users := []struct {
		ID       int
		Name     string
		Username string
		Email    string
	}{}

	transaction := Get("https://jsonplaceholder.typicode.com/users").
		Result(&users)

	// Get data from a remote server
	// nolint:errcheck // just a test
	if err := transaction.Send(); err != nil {
		derp.Report(err)
		t.Fail()
	}

	require.Equal(t, users[0].Name, "Leanne Graham")
	require.Equal(t, users[0].Username, "Bret")
	require.Equal(t, users[0].Email, "Sincere@april.biz")
}

func TestPost(t *testing.T) {

	ts := echoBodyServer()
	defer ts.Close()

	body := map[string]string{
		"first":  "1",
		"second": "2",
		"third":  "3",
	}

	success := map[string]any{}
	failure := map[string]any{}

	txn := Post(ts.URL).
		JSON(body).
		Result(&success).
		Error(&failure)

	if err := txn.Send(); err != nil {
		derp.Report(err)
		t.Fail()
	}

	require.Equal(t, "1", success["first"])
	require.Equal(t, "2", success["second"])
	require.Equal(t, "3", success["third"])
}

func echoBodyServer() *httptest.Server {

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", r.Header.Get("Content-Type"))

		body := new(bytes.Buffer)
		// nolint:errcheck // just a test
		body.ReadFrom(r.Body)
		// nolint:errcheck // just a test
		w.Write(body.Bytes())
	}))
}
