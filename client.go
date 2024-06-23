package remote

import (
	"net/http"
	"time"
)

func DefaultClient() *http.Client {

	return &http.Client{
		Timeout: 10 * time.Second,
	}
}
