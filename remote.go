package remote

import (
	"net/url"
)

// New returns a fully initialized Transaction object with default settings.
func New() *Transaction {

	t := &Transaction{
		client:  DefaultClient(),
		method:  "",
		url:     "",
		header:  map[string]string{},
		query:   url.Values{},
		form:    url.Values{},
		options: []Option{},
	}

	return t
}

// Get creates a new HTTP request to the designated URL, using the GET method
func Get(url string) *Transaction {
	return New().Get(url)
}

// Post creates a new HTTP request to the designated URL, using the POST method
func Post(url string) *Transaction {
	return New().Post(url)
}

// Put creates a new HTTP request to the designated URL, using the PUT method
func Put(url string) *Transaction {
	return New().Put(url)
}

// Patch creates a new HTTP request to the designated URL, using the PATCH method
func Patch(url string) *Transaction {
	return New().Patch(url)
}

// Delete creates a new HTTP request to the designated URL, using the DELETE method.
func Delete(url string) *Transaction {
	return New().Delete(url)
}
