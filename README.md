# remote
Simple, chainable API for making HTTP requests to remote servers using Go.

Inspired by [Wrecker, from Brandon Romano](https://github.com/BrandonRomano/wrecker)

## Get data from an HTTP server
```go
users := []struct {
	ID string
	Name string
	Username string
	Email string
}{}

errorResponse := map[string]string{}

// Get data from a remote server
transaction := remote.Get("https://jsonplaceholder.typicode.com/users").
	Result(users, errorResponse)

if err := transaction.Do(); err != nil {
	// Handle errors...
	// http error codes are in the err object.
	// http body is parsed into errorResponse
}

// Now the `users` data structure is available here.
```


## Post data to an HTTP server
```go
user := map[string]string{
	"id": "ABC123",
	"name": "Sarah Connor",
	"email": "sarah@sky.net",
}

response := map[string]string{}
errorResponse := map[string]string{}

transaction := remote.Post("https://hookbin.com/abc123").
	JSON(user).
	Result(response, errorResonse)

if err := transaction.Do(); err != nil {
	// Handle errors here...
}

// Use the `response` value here, if needed
```