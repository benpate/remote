package remote

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/assert"
)

func TestRequestMethods(t *testing.T) {

	get := Get("someurl.com")
	assert.Equal(t, "GET", get.Method)
	assert.Equal(t, "someurl.com", get.URLValue)

	post := Post("someurl.com")
	assert.Equal(t, "POST", post.Method)
	assert.Equal(t, "someurl.com", post.URLValue)

	put := Put("someurl.com")
	assert.Equal(t, "PUT", put.Method)
	assert.Equal(t, "someurl.com", put.URLValue)

	patch := Patch("someurl.com")
	assert.Equal(t, "PATCH", patch.Method)
	assert.Equal(t, "someurl.com", patch.URLValue)

	delete := Delete("someurl.com")
	assert.Equal(t, "DELETE", delete.Method)
	assert.Equal(t, "someurl.com", delete.URLValue)
}

func TestGet(t *testing.T) {

	users := []struct {
		ID       int
		Name     string
		Username string
		Email    string
	}{}

	transaction := Get("https://jsonplaceholder.typicode.com/users").
		Response(&users, nil)

	// Get data from a remote server
	// nolint:errcheck // just a test
	if err := transaction.Send(); err != nil {
		derp.Report(err)
		t.Fail()
	}

	assert.Equal(t, users[0].Name, "Leanne Graham")
	assert.Equal(t, users[0].Username, "Bret")
	assert.Equal(t, users[0].Email, "Sincere@april.biz")
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

	if err := Post(ts.URL).JSON(body).Response(&success, &failure).Send(); err != nil {
		t.Log(err)
		t.Log(failure)
		return
	}

	assert.Equal(t, "1", success["first"])
	assert.Equal(t, "2", success["second"])
	assert.Equal(t, "3", success["third"])
}

func echoBodyServer() *httptest.Server {

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

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
