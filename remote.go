package remote

import (
	"net/http"
	"net/url"
	"time"
)

func newTransaction(method string, urlValue string) *Transaction {

	t := &Transaction{
		client:  &http.Client{Timeout: 10 * time.Second},
		method:  method,
		url:     urlValue,
		header:  map[string]string{},
		query:   url.Values{},
		form:    url.Values{},
		options: []Option{},
	}

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
