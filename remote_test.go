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

	get := Get("someurl.com")
	require.Equal(t, "GET", get.method)
	require.Equal(t, "someurl.com", get.url)

	post := Post("someurl.com")
	require.Equal(t, "POST", post.method)
	require.Equal(t, "someurl.com", post.url)

	put := Put("someurl.com")
	require.Equal(t, "PUT", put.method)
	require.Equal(t, "someurl.com", put.url)

	patch := Patch("someurl.com")
	require.Equal(t, "PATCH", patch.method)
	require.Equal(t, "someurl.com", patch.url)

	delete := Delete("someurl.com")
	require.Equal(t, "DELETE", delete.method)
	require.Equal(t, "someurl.com", delete.url)
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

/*
func TestRequestBinGet(t *testing.T) {

	// Check results at: https://pipedream.com/r/envocp9hr03ig

	Get("https://envocp9hr03ig.x.pipedream.net").
		Query("name1", "value1").
		Query("name2", "value2").
		Query("name3", "value3").
		Send()
}

func TestRequestBinPost(t *testing.T) {

	body := map[string]string{
		"hello":  "darkness",
		"my-old": "friend",
	}

	// Check results at: https://pipedream.com/r/envocp9hr03ig

	transaction := Post("https://envocp9hr03ig.x.pipedream.net/path/goes/here").
		JSON(body).
		Header("User-Agent", "remote")

	if err := transaction.Send(); err != nil {
		err.Report()
		t.Fail()
	}
}
*/
