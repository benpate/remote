package middleware

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/benpate/derp"
)

// Debug is a sample middleware that adds debugging output to every request.
func Debug() Middleware {

	return Middleware{

		Request: func(r *http.Request) *derp.Error {

			fmt.Println("")
			fmt.Println("HTTP Request")
			fmt.Println("-------------")
			fmt.Println("Method: ", r.Method)
			fmt.Println("URL: ", r.URL.String())
			fmt.Println("")

			return nil
		},

		Response: func(r *http.Response, body *[]byte) *derp.Error {

			fmt.Println("")
			fmt.Println("HTTP Response")
			fmt.Println("-------------")

			fmt.Println("Status Code: ", strconv.Itoa(r.StatusCode))
			fmt.Println("Status: ", r.Status)
			fmt.Println("Headers:")

			for i := range r.Header {
				fmt.Println("- ", i, ": ", r.Header.Get(i))
			}

			fmt.Println("Body: ", string(body))
			fmt.Println("")

			return nil
		},
	}
}
