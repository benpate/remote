package middleware

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/benpate/derp"
	"github.com/benpate/re"
	"github.com/benpate/remote"
)

// Debug is a sample middleware that adds debugging output to every request.
func Debug() remote.Middleware {

	return remote.Middleware{

		Request: func(r *http.Request) error {

			body, err := re.ReadRequestBody(r)

			if err != nil {
				return derp.Wrap(err, "remote.middleware.Debug", "Error reading body")
			}

			fmt.Println("")
			fmt.Println("HTTP Request")
			fmt.Println("-------------")
			fmt.Println("Method: ", r.Method)
			fmt.Println("URL: ", r.URL.String())
			fmt.Println("Content-Length: " + strconv.FormatInt(r.ContentLength, 10))
			fmt.Println("Headers:")

			for i := range r.Header {
				fmt.Println("- ", i, ": ", r.Header.Get(i))
			}

			fmt.Println("")
			fmt.Println(string(body))
			fmt.Println("")

			return nil
		},

		Response: func(r *http.Response, body *[]byte) error {

			fmt.Println("")
			fmt.Println("HTTP Response")
			fmt.Println("-------------")

			fmt.Println("Status Code: ", strconv.Itoa(r.StatusCode))
			fmt.Println("Status: ", r.Status)
			fmt.Println("Headers:")

			for i := range r.Header {
				fmt.Println("- ", i, ": ", r.Header.Get(i))
			}

			fmt.Println(string(*body))
			fmt.Println("")

			return nil
		},
	}
}
