package middleware

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/benpate/derp"
	"github.com/benpate/remote"
)

// Dump is a sample middleware that dumps the raw request value
func Dump() remote.Middleware {

	return remote.Middleware{

		Request: func(r *http.Request) error {
			dump, err := httputil.DumpRequest(r, true)

			if err != nil {
				return derp.Wrap(err, "middleware.Dump", "Error dumping request")
			}

			fmt.Println(string(dump))
			return nil
		},
	}
}
