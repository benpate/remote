package remote

import "net/http"

// ErrorReport includes all the data returned by a transaction if it throws an error for any reason.
type ErrorReport struct {
	URL     string `json:"url"`
	Request struct {
		Method string            `json:"method"`
		Header map[string]string `json:"header"`
		Body   string            `json:"body"`
	} `json:"request"`
	Response struct {
		StatusCode int         `json:"statusCode"`
		Status     string      `json:"status"`
		Header     http.Header `json:"header"`
		Body       string      `json:"body"`
	} `json:"response"`
}
