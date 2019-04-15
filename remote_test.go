package remote

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
)

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

func TestGet(t *testing.T) {

	users := []struct {
		ID       string
		Name     string
		Username string
		Email    string
	}{}

	transaction := Get("https://jsonplaceholder.typicode.com/users").
		Response(&users, nil)

	// Get data from a remote server
	if err := transaction.Send(); err != nil {
		err.Report()
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

func echoBodyServer() *httptest.Server {

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		body := new(bytes.Buffer)
		body.ReadFrom(r.Body)
		w.Write(body.Bytes())
	}))
}
