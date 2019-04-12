package remote

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
)

func TestHookbinGet(t *testing.T) {

	Get("https://envocp9hr03ig.x.pipedream.net").
		Query("name1", "value1").
		Query("name2", "value2").
		Query("name3", "value3").
		Send()
}

func TestHookbinPost(t *testing.T) {

	body := map[string]string{
		"hello":  "darkness",
		"my old": "friend",
	}

	err := Post("https://envocp9hr03ig.x.pipedream.net").
		JSON(body).
		Header("User-Agent", "remote").
		Send()

	t.Log(spew.Sdump(err))
	t.Fail()
}

func TestGet(t *testing.T) {

	users := []struct {
		ID       string
		Name     string
		Username string
		Email    string
	}{}

	// Get data from a remote server
	Get("https://jsonplaceholder.typicode.com/users").
		Response(&users, nil).
		Send()

	t.Log(spew.Sdump(users))
}

func TestPost(t *testing.T) {

	ts := echoBodyServer()
	defer ts.Close()

	body := map[string]string{
		"first":  "1",
		"second": "2",
		"third":  "3",
	}

	success := map[string]interface{}{}
	failure := map[string]interface{}{}

	if err := Post(ts.URL).JSON(body).Response(&success, &failure).Send(); err != nil {
		t.Log(spew.Sdump(err))
		t.Log(spew.Sdump(failure))
		return
	}

	// t.Log(spew.Sdump(success))
	assert.Equal(t, success["first"], "1")
	assert.Equal(t, success["second"], "2")
	assert.Equal(t, success["third"], "3")
}

func TestPut(t *testing.T) {

	ts := echoHeaderServer()
	defer ts.Close()

	success := ""
	failure := map[string]interface{}{}

	if err := Put(ts.URL).Response(&success, &failure).Send(); err != nil {
		return
	}

	t.Log(spew.Sdump(success))
}

func echoBodyServer() *httptest.Server {

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		body := new(bytes.Buffer)
		body.ReadFrom(r.Body)
		w.Write(body.Bytes())
	}))
}

func echoHeaderServer() *httptest.Server {

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := new(bytes.Buffer)
		r.Header.Write(header)

		spew.Dump(header.String())
		r.Header.Write(w)
	}))
}
