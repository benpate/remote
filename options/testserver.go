package options

import (
	"bufio"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/benpate/derp"
	"github.com/benpate/remote"
)

// TestServer is a remote.Option that mocks requests for a specific hostname.
func TestServer(hostname string, filesystem fs.FS) remote.Option {

	// errorResponse just simplifies error handling in the actual option.
	errorResponse := func(request *http.Request, err error) *http.Response {

		derp.Report(err)
		body := io.NopCloser(strings.NewReader(err.Error()))

		return &http.Response{
			Request:    request,
			StatusCode: http.StatusNotFound,
			Body:       body,
		}
	}

	// Generate the actual option
	return remote.Option{

		ModifyRequest: func(_ *remote.Transaction, request *http.Request) *http.Response {

			// Only match requests for the specified hostname
			if request.URL.Hostname() != hostname {
				return nil
			}

			// Locate the file using the URL path
			filename := strings.TrimPrefix(request.URL.Path, "/")
			file, err := filesystem.Open(filename)

			if err != nil {
				return errorResponse(request, err)
			}

			// Read the response from the fs.File
			response, err := http.ReadResponse(bufio.NewReader(file), request)

			if err != nil {
				return errorResponse(request, err)
			}

			// I see this as a complete success!
			return response
		},
	}
}
