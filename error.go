package remote

import "net/http"

// ErrorReport includes all the data returned by a transaction if it throws an error for any reason.
type ErrorReport struct {
	Request    string      `json:"request"`
	StatusCode int         `json:"statusCode"`
	Status     string      `json:"status"`
	Header     http.Header `json:"header"`
	Body       string      `json:"body"`
}
