package remote

import (
	"net/http"
	"net/url"
	"time"
)

// ErrorReport includes all the data returned by a transaction if it throws an error for any reason.
type ErrorReport struct {
	Request    string
	StatusCode int
	Status     string
	Header     http.Header
	Body       string
}

func newTransaction(method string, urlValue string) *Transaction {

	t := &Transaction{
		Client:       &http.Client{Timeout: 10 * time.Second},
		Method:       method,
		URLValue:     urlValue,
		HeaderValues: map[string]string{},
		QueryString:  url.Values{},
		FormData:     url.Values{},
		Middleware:   []Middleware{},
	}

	t.ContentType(ContentTypePlain)

	return t
}

// Get creates a new HTTP request to the designated URL, using the GET method
func Get(url string) *Transaction {
	return newTransaction(http.MethodGet, url)
}

// Post creates a new HTTP request to the designated URL, using the POST method
func Post(url string) *Transaction {
	return newTransaction(http.MethodPost, url)
}

// Put creates a new HTTP request to the designated URL, using the PUT method
func Put(url string) *Transaction {
	return newTransaction(http.MethodPut, url)
}

// Patch creates a new HTTP request to the designated URL, using the PATCH method
func Patch(url string) *Transaction {
	return newTransaction(http.MethodPatch, url)
}

// Delete creates a new HTTP request to the designated URL, using the DELETE method.
func Delete(url string) *Transaction {
	return newTransaction(http.MethodDelete, url)
}
