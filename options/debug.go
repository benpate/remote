package options

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/benpate/derp"
	"github.com/benpate/remote"
)

// Debug is a remote.Option that adds debugging output to every request.
func Debug() remote.Option {

	return remote.Option{

		ModifyRequest: func(transaction *remote.Transaction, request *http.Request) *http.Response {

			body, err := transaction.RequestBody()

			if err != nil {
				derp.Report(derp.Wrap(err, "remote.option.Debug", "Error reading body"))
			}

			fmt.Println("")
			fmt.Println("HTTP Request")
			fmt.Println("-------------")
			fmt.Println("Method: ", request.Method)
			fmt.Println("URL: ", request.URL.String())
			fmt.Println("Content-Length: " + strconv.FormatInt(request.ContentLength, 10))
			fmt.Println("Headers:")

			for i := range request.Header {
				fmt.Println("- ", i, ": ", request.Header.Get(i))
			}

			fmt.Println("")
			fmt.Println(string(body))
			fmt.Println("")

			return nil
		},

		AfterRequest: func(transaction *remote.Transaction, response *http.Response) error {

			fmt.Println("")
			fmt.Println("HTTP Response")
			fmt.Println("-------------")

			fmt.Println("Status Code: ", strconv.Itoa(response.StatusCode))
			fmt.Println("Status: ", response.Status)
			fmt.Println("Headers:")

			for i := range response.Header {
				fmt.Println("- ", i, ": ", response.Header.Get(i))
			}

			if body, err := transaction.ResponseBody(); err == nil {
				fmt.Println("")
				fmt.Println(string(body))
				fmt.Println("")
			}

			return nil
		},
	}
}
