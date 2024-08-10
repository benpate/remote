package options

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/benpate/derp"
	"github.com/benpate/remote"
)

// Debug is a remote.Option that adds debugging output to every request.
func Debug() remote.Option {

	return remote.Option{

		ModifyRequest: func(transaction *remote.Transaction, request *http.Request) *http.Response {

			content, err := httputil.DumpRequestOut(request, true)

			if err != nil {
				derp.Report(derp.Wrap(err, "remote.option.Debug", "Error reading body"))
			}

			fmt.Println("")
			fmt.Println("Begin HTTP Request -----------------------")
			fmt.Println(string(content))
			fmt.Println("END --------------------------------------")
			fmt.Println("")

			return nil
		},

		AfterRequest: func(transaction *remote.Transaction, response *http.Response) error {

			fmt.Println("")
			fmt.Println("Begin HTTP Response ----------------------")

			content, err := httputil.DumpResponse(response, true)

			if err != nil {
				derp.Report(derp.Wrap(err, "remote.option.Debug", "Error reading body"))
			}

			fmt.Println(string(content))
			fmt.Println("END --------------------------------------")
			fmt.Println("")
			return nil
		},
	}
}
