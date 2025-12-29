package remote

import (
	"net/http"
	"time"
)

// DefaultClient returns an HTTP client with a reasonable timeout.
func DefaultClient() *http.Client {

	return &http.Client{
		Timeout: 10 * time.Second,
	}
}
