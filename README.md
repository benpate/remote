# REMOTE 🏝

[![GoDoc](https://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://pkg.go.dev/github.com/benpate/remote)
[![Build Status](https://img.shields.io/github/workflow/status/benpate/remote/Go/master)](https://github.com/benpate/remote/actions/workflows/go.yml)
[![Codecov](https://img.shields.io/codecov/c/github/benpate/remote.svg?style=flat-square)](https://codecov.io/gh/benpate/remote)
[![Go Report Card](https://goreportcard.com/badge/github.com/benpate/remote?style=flat-square)](https://goreportcard.com/report/github.com/benpate/remote)
[![Version](https://img.shields.io/github/v/release/benpate/remote?include_prereleases&style=flat-square&color=brightgreen)](https://github.com/benpate/remote/releases)

## Crazy Simple, Chainable HTTP Client for Go

Remote is a paper-thin wrapper on top of Go's HTTP library, that gives you sensible defaults, a pretty API with some modern conveniences, and full control of your HTTP requests.  It's the fastest and easiest way to make an HTTP call using Go.

Inspired by [Brandon Romano's Wrecker](https://github.com/BrandonRomano/wrecker).  Thanks Brandon!

### How to Get data from an HTTP server

```go
// Structure to read remote data into
users := []struct {
    ID string
    Name string
    Username string
    Email string
}{}

// Get data from a remote server
remote.Get("https://jsonplaceholder.typicode.com/users").
    Result(users, nil).
    Send()

```

### How to Post/Put/Patch/Delete data to an HTTP server

```go
// Data to send to the remote server
user := map[string]string{
    "id": "ABC123",
    "name": "Sarah Connor",
    "email": "sarah@sky.net",
}

// Structure to read response into
response := map[string]string{}

// Post data to the remote server (use your own URL)
remote.Post("https://example.com/post-service").
    JSON(user). // encode the user object into the request body as JSON
    Result(response, nil). // parse response (or error) into a data structure
    Send()
```

### Handling HTTP Errors

Web services represent errors in a number of ways.  Some simply return an HTTP error code,
while others return complex data structures in the response body.  REMOTE works with each
style of error handling, so that your app always has the best information to work from.

```go
// Structure to read successful response into.  This format is specific to the HTTP service.
success := struct{Name: string, Value: string, Comment: string}

// Structure to read failed response data into.  This format is specific to the HTTP service.
failure := struct(Code: int, Reason: string, StackTrace: string)

transaction := remote.Get("https://example.com/service-that-might-error").
    .Result(&success, &failure)

// Transaction returns an error **IF** the HTTP response code is not successful (200-299)
if err := transaction.Send(); err != nil {
    // Handle errors here.
    // `failure` variable will be populated with data from the remote service
    return
}

// Fall through to here means that the transaction was successful.  
// `success` variable will be populated with data from the remote service.
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

Original versions of this library have been used in production on commercial applications for years, and have helped speed up development for everyone involved.  

I'm now open sourcing this library, and others, with hopes that you'll also benefit from an easy HTTP library.

Please use GitHub to make suggestions, pull requests, and enhancements.  We're all in this together! 🏝
