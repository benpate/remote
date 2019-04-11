# remote

[![Go Report Card](https://goreportcard.com/badge/github.com/benpate/remote)](https://goreportcard.com/report/github.com/benpate/remote)
[![Documentation](https://godoc.org/github.com/benpate/remote?status.svg)](http://godoc.org/github.com/benpate/remote)

Crazy simple, chainable API for making HTTP requests to remote servers using Go.

Inspired by [Brandon Romano's Wrecker](https://github.com/BrandonRomano/wrecker).  Thanks Brandon!


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
	Result(users, errorResponse) // parse response (or error) into a data structure

if err := transaction.Send(); err != nil {
	// Handle errors...
}
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

// Post data to the remote server (use your own URL)
transaction := remote.Post("https://hookbin.com/abc123").
	JSON(user). // encode the user object into the request body as JSON
	Result(response, errorResonse) // parse response (or error) into a data structure

if err := transaction.Send(); err != nil {
	// Handle errors...
}
```


# Middleware
Middleware allows you to modify a request before it is sent to the remote server, or modify the response after it is returned by the remote server.  Each middleware object includes three hooks

## Included Middleware

**Authorization** adds a simple `Authorization` header to your HTTP request
```go
remote.Get("https://jsonplaceholder.typicode.com/users").
	Use(middleware.Authorization(myAuthorizationKey)).
	Result(users, nil).
	Send()
```

**/middleware/basicauth.go** adds a `Authorization` header to your HTTP request, which is Base 64 encoded to follow the basic authorization standard.
```go
remote.Get("https://jsonplaceholder.typicode.com/users").
	Use(middleware.BasicAuth(username, password)).
	Result(users, nil).
	Send()
```

**/middleware/debug.go** prints debugging statements to the console to help you see what's happening with your HTTP requests
```go
remote.Get("https://jsonplaceholder.typicode.com/users").
	Use(middleware.Debug()).
	Result(users, nil).
	Send()
```

**/middleware/opaque.go** adds an "Opaque" value to the HTTP request.  This should not be required under most circumstances.
```go
remote.Get("https://jsonplaceholder.typicode.com/users").
	Use(middleware.Opaque(opaqueURLStringHere)).
	Result(users, nil).
	Send()
```

## Writing Custom Middleware
It's easy to write additional, custom middleware for your project.  Just follow the samples in the `/middleware` folder, and pass in any object that follows the `Middleware` interface.

**`Config(*Transaction)`** allows you to change the transaction configuration before it is compiled into an HTTP request.  This is typically the simplest, and easiest way to modify a request

**`Request(*http.Request)`** allows you to modify the raw HTTP request before it is sent to the remote server.  This is useful in the rare cases when you need to make changes to a request that this library doesn't support.

**`Response(*http.Response)`** allows you to modify the raw HTTP response before its results are parsed and returned to the caller.