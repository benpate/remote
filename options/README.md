# Remote Options 🧩

A library of common [`remote.Option`](../README.md) values for modifying a `remote.Transaction`. Add any of these to a transaction with `.With(...)`.

See the [parent README](../README.md) for the client itself.

```go
import "github.com/benpate/remote/options"

remote.Get("https://example.com").
    With(options.BearerAuth(token)).
    With(options.UserAgent("my-app/1.0")).
    Send()
```

## What's Included

Most options set a single request header before the request is assembled:

* **`Accept(value)`** — sets the `Accept` header.
* **`UserAgent(value)`** — sets the `User-Agent` header.
* **`Authorization(value)`** — sets a raw `Authorization` header.
* **`BasicAuth(username, password)`** — sets `Authorization` to a Base64 HTTP Basic credential.
* **`BearerAuth(token)`** — sets `Authorization` to a `Bearer` token.

Two options act on the raw `http.Request` instead:

* **`Opaque(value)`** — overrides `request.URL.Opaque`, for servers that require characters in the path to *not* be URL-encoded (e.g. LinkedIn's REST API).
* **`Debug()`** — dumps the full request and response to stdout. For development only.

And one mocks the network entirely:

* **`TestServer(hostname, fs.FS)`** — intercepts requests for a given hostname and serves canned responses from a filesystem, so tests never touch the real network. See below.

## What matters here

* **Header options live in `BeforeRequest`; request-mutating options live in `ModifyRequest`.** Header options (`Accept`, `BasicAuth`, etc.) run before the `http.Request` exists, so they call `transaction.Header(...)`. `Opaque` and `Debug` need the assembled request, so they run later. This ordering is why a `BeforeRequest` option can't touch `request.URL` and a `ModifyRequest` option can't change a value the request was already built from — match the hook to what you need to mutate.

* **`TestServer` returns a response from `ModifyRequest`, which short-circuits the network — and that means the SSRF guards never run for mocked hosts.** That is intentional and exactly what you want in a test, but don't lean on `TestServer` to exercise the dialer-level private-IP or redirect guards; those only fire on a real dial.

* **`TestServer` files are raw HTTP responses, not bare bodies.** Each fixture is parsed with `http.ReadResponse`, so a file must include a status line and headers (e.g. `HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{...}`), not just the body. The opened file's lifetime is tied to the response body, so it is closed when the caller closes the body — don't add your own close.

* **`Debug()` writes to stdout and is not safe for production.** It dumps full request and response bodies, including any `Authorization` header. Keep it out of committed code paths.

* **These are templates.** Each option is a few lines returning a `remote.Option`. To write your own, copy the nearest match — a header tweak from `userAgent.go`, a request mutation from `opaque.go`.
