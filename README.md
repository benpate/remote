# REMOTE 🏝

[![GoDoc](https://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://pkg.go.dev/github.com/benpate/remote)
[![Version](https://img.shields.io/github/v/release/benpate/remote?include_prereleases&style=flat-square&color=brightgreen)](https://github.com/benpate/remote/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/benpate/remote/go.yml?branch=main&style=flat-square)](https://github.com/benpate/remote/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/benpate/remote?style=flat-square)](https://goreportcard.com/report/github.com/benpate/remote)
[![Codecov](https://img.shields.io/codecov/c/github/benpate/remote.svg?style=flat-square)](https://codecov.io/gh/benpate/remote)

## Crazy Simple, Chainable HTTP Client for Go

Remote is a paper-thin wrapper for Go's HTTP library that gives you sensible defaults, a chainable API with modern conveniences, and full control of your HTTP requests. It's a fast, easy way to make an HTTP call in Go — and it is hardened against SSRF by default.

Inspired by [Brandon Romano's Wrecker](https://github.com/BrandonRomano/wrecker). Thanks Brandon!

### Get data from an HTTP server

```go
// Structure to read the remote data into
users := []struct {
    ID       string
    Name     string
    Username string
    Email    string
}{}

// Get data from a remote server
err := remote.Get("https://jsonplaceholder.typicode.com/users").
    Result(&users).
    Send()
```

### Post/Put/Patch/Delete data to an HTTP server

```go
// Data to send to the remote server
user := map[string]string{
    "id":    "ABC123",
    "name":  "Sarah Connor",
    "email": "sarah@sky.net",
}

// Structure to read the response into
response := map[string]string{}

// Post data to the remote server
err := remote.Post("https://example.com/post-service").
    JSON(user).         // encode the user object into the request body as JSON
    Result(&response).  // parse a successful response into this structure
    Send()
```

### Handling HTTP Errors

Web services represent errors in many ways. Some return only an HTTP status code; others return a structured document in the response body. Remote handles both. Use `.Result()` for the success body and `.Error()` for the failure body — `Send()` returns a non-nil error whenever the response status is outside the 200–299 range, and populates whichever object matches the outcome.

```go
success := SuccessResponse{} // shape defined by the remote service
failure := ErrorResponse{}   // shape defined by the remote service

err := remote.Get("https://example.com/service-that-might-error").
    Result(&success).
    Error(&failure).
    Send()

// Send returns an error IF the HTTP status is not 2xx.
if err != nil {
    // `failure` is populated with the error body from the remote service.
    return err
}

// Fall through means success: `success` is populated.
```

## Security

Remote is built for calling untrusted, user-supplied URLs safely. These guards are on by default.

* **Private IPs are blocked.** By default the client refuses to connect to loopback, private, and link-local addresses — defending against SSRF. The check lives in the dialer and re-runs on every redirect hop, so it is safe against DNS rebinding. Call `.AllowPrivateIPs(true)` to opt out (e.g. for localhost or internal services).
* **Host allow-listing.** `.AllowHosts("example.com", ...)` restricts a transaction to specific hosts. The list is re-checked on every redirect, so an allow-listed server cannot redirect you somewhere unexpected.
* **Response size is capped** at 1GB by default, preventing a hostile server from exhausting memory. Tune it with `.MaxResponseSize(n)`.
* **Redirects are capped** at 5 hops.
* **Requests are time-bounded.** Without a context, a one-minute timeout applies. Supply your own deadline or cancellation with `.WithContext(ctx)`.

```go
err := remote.Get(userSuppliedURL).
    AllowHosts("api.trusted.com").
    MaxResponseSize(10 * 1024 * 1024). // 10MB
    WithContext(ctx).
    Result(&data).
    Send()
```

## Options

Options modify a request before it is sent, or the response after it returns. Add them with `.With(...)`. The [`options`](https://github.com/benpate/remote/tree/main/options) subpackage ships a library of common ones.

```go
import "github.com/benpate/remote/options"

err := remote.Get("https://jsonplaceholder.typicode.com/users").
    With(options.BearerAuth(myAccessToken)).
    With(options.UserAgent("my-app/1.0")).
    Result(&users).
    Send()
```

See the [options README](options/README.md) for the full list. An `Option` exposes three hooks — `BeforeRequest`, `ModifyRequest`, and `AfterRequest` — so you can write your own; the included options are short and make good templates.

## Custom Transport

`.WithRoundTripper(...)` wraps the SSRF-hardened base transport with your own middleware (for caching, instrumentation, custom headers, etc.). Your middleware receives the base transport as `next` and **must delegate to it** to perform the request — keeping the private-IP guard underneath.

```go
err := remote.Get("https://example.com").
    WithRoundTripper(func(next http.RoundTripper) http.RoundTripper {
        return myCachingTransport{next: next}
    }).
    Send()
```

Note that if your middleware short-circuits and returns a response *without* delegating to `next` (e.g. a cache hit), no dial happens — and therefore neither SSRF guard runs. That is by design for caching, but worth knowing when the source URL is untrusted.

## Pull Requests Welcome

Original versions of this library have been used in production on commercial applications for years, and have helped speed up development for everyone involved.

I'm open sourcing this library, and others, with hopes that you'll also benefit from an easy HTTP library.

Please use GitHub to make suggestions, pull requests, and enhancements. We're all in this together! 🏝
