# REMOTE 🏝

[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/benpate/remote)
[![Go Report Card](https://goreportcard.com/badge/github.com/benpate/remote?style=flat-square)](https://goreportcard.com/report/github.com/benpate/remote)
[![Build Status](http://img.shields.io/travis/benpate/remote.svg?style=flat-square)](https://travis-ci.org/benpate/remote)
[![Codecov](https://img.shields.io/codecov/c/github/benpate/remote.svg?style=flat-square)](https://codecov.io/gh/benpate/remote)

## Crazy simple, chainable API for making HTTP requests to remote servers using Go.

Remote is a paper-thin wrapper on top of Go's HTTP library, that gives you sensible defaults, a pretty API with some modern conveniences, and full control of your HTTP requests.  It's the fastest and easiest way to make an HTTP call using Go.

Inspired by [Brandon Romano's Wrecker](https://github.com/BrandonRomano/wrecker).  Thanks Brandon!


### Get data from an HTTP server
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


### Post data to an HTTP server
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


## Middleware
Middleware allows you to modify a request before it is sent to the remote server, or modify the response after it is returned by the remote server.  Each middleware object includes three hooks

### Included Middleware

```go
// AUTHORIZATION adds a simple "Authorization" header to your request
remote.Get("https://jsonplaceholder.typicode.com/users").
	Use(middleware.Authorization(myAuthorizationKey)).
	Result(users, nil).
	Send()
```

```go
// BASIC AUTH adds a Base64 encoded "Authorization" header to your request,
// which follows the basic authorization standard
remote.Get("https://jsonplaceholder.typicode.com/users").
	Use(middleware.BasicAuth(username, password)).
	Result(users, nil).
	Send()
```

```go
// DEBUG prints debugging statements to the console
remote.Get("https://jsonplaceholder.typicode.com/users").
	Use(middleware.Debug()).
	Result(users, nil).
	Send()
```

```go
// OPAQUE makes direct changes to the URL string.
remote.Get("https://jsonplaceholder.typicode.com/users").
	Use(middleware.Opaque(opaqueURLStringHere)).
	Result(users, nil).
	Send()
```

### Writing Custom Middleware
It's easy to write additional, custom middleware for your project.  Just follow the samples in the `/middleware` folder, and pass in any object that follows the `Middleware` interface.

**`Config(*Transaction)`** allows you to change the transaction configuration before it is compiled into an HTTP request.  This is typically the simplest, and easiest way to modify a request

**`Request(*http.Request)`** allows you to modify the raw HTTP request before it is sent to the remote server.  This is useful in the rare cases when you need to make changes to a request that this library doesn't support.

**`Response(*http.Response)`** allows you to modify the raw HTTP response before its results are parsed and returned to the caller.


## Pull Requests Welcome
Original versions of the derp library have been used in production on commercial applications for years, and the extra data collection has been a tremendous help for everyone involved.  

I'm now open sourcing this library, and others, with hopes that you'll also benefit from a more robust error package.

Please use GitHub to make suggestions, pull requests, and enhancements.  We're all in this together! 🏝