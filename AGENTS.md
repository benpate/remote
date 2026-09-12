# remote — Agent Notes

See [README.md](README.md) for the public API. These are the non-obvious rules.

## The SSRF guard is the DEFAULT, and it lives in the transport

`remote.Get`/`remote.Post` refuse to connect to non-public addresses (loopback, private, link-local incl. `169.254.169.254`, etc.) **unless** the caller sets `AllowPrivateIPs(true)`. The guard is `safeTransport`'s dialer (`guardedDialContext` → `publicIPs`), which resolves the host and dials a validated IP literal, so it is DNS-rebinding-safe and re-runs on every redirect hop. A raw `remote.Get(attackerURL)` is therefore already protected — do NOT "fix" a caller by routing it through some other guarded wrapper; there is nothing to add here.

## `baseTransport(allowPrivateIPs)` is the single source of truth — don't duplicate the branch

The `safeTransport`-vs-`http.DefaultTransport` decision exists in exactly one place. Both `NewHTTPClient` and `Transaction.buildClient` call `baseTransport`. If you add another client-construction path, call `baseTransport` too — never re-write `if allowPrivateIPs { DefaultTransport }`, or the two will drift on which transport guards a request.

## `NewHTTPClient` is for libraries that take an `*http.Client`, not a `Transaction`

Use it to hand the guarded transport to code you don't control — notably `golang.org/x/oauth2` via the `oauth2.HTTPClient` context key. The guard rides in the transport, so those libraries get full private-IP protection for free. Its `guardedRedirect` policy only caps the hop count (no host allow-list — standalone clients have none); the private-IP guard still fires per hop because it's in the dialer.

## `safeTransport` is a shared, pooled singleton — never mutate it

It is exported to callers only as a fresh `*http.Client` from `NewHTTPClient` (cheap wrapper, shared transport underneath) so connection pooling is preserved and no caller can disarm the guard by writing to a `*http.Transport` field. Keep it that way: hand out clients, not the transport.

## Body ownership: `Send` closes the OUTERMOST body, once

`Send` registers its deferred `Body.Close()` *after* `onAfterRequest` runs, capturing whatever body the hooks left in place. Two rules fall out of that, and both are easy to break:

- An `AfterRequest` hook that replaces `response.Body` **owns the reader it replaced** — its `Close` must close the wrapper's inner reader, or the connection leaks. `remote` closes only the outermost body.
- Do **not** "simplify" the deferred close to read `t.response.Body` at defer time. `ResponseBody` swaps in an `io.NopCloser` over an in-memory buffer once the body is read, so a late-binding close would close that no-op and leak the real connection on every request. The capture must stay between `onAfterRequest` and `ResponseBody`.

The error path out of `onAfterRequest` returns before the defer is registered, so it closes the body itself via `closeResponseBody`. `TestSend_ClosesAfterRequestWrapper` and `TestSend_ClosesResponseBodyWhenAfterRequestFails` pin both behaviors.

## `ResponseBody` copies; `responseBody` does not

The exported `ResponseBody` returns `bytes.Clone` of the buffer, so a caller owns what it hands back. The unexported `responseBody` returns the buffer itself, and is what `Send` and `ResponseBodyReader` call — neither writes to it, so neither should pay to duplicate a body that may be up to `MaxResponseSize` (1GB by default). Keep in-package readers on `responseBody`; anything crossing the package boundary goes through `ResponseBody`. `decodeResponseBody` clones separately for its `*[]byte` case, because that slice becomes the caller's result object.

## `AllowHosts` entries are port-optional, and the port is inferred from the scheme

A bare entry (`"example.com"`) matches any port; an entry with a port (`"example.com:8443"`) matches only that one. `hostIsAllowed` therefore takes the whole `*url.URL`, not a hostname string — it needs the scheme to fill in an omitted default port, so that `AllowHosts("example.com:443")` still matches `https://example.com/`. Don't "simplify" it back to a string parameter; the scheme is important.

## `RequestBody` replaces an `io.Reader` body with its bytes

Reading a stream consumes it, so `RequestBody` stores what it read back into `t.body`. Without that, the SECOND caller gets an empty body and no error — and there is more than one caller: `MarshalMap` (used to log or queue a transaction) reads the body, and so does `assembleRequest`. Marshalling a transaction for a log line before sending it would otherwise send an empty request. `TestRequestBody_ReaderIsRepeatable` pins this.

## `MaxResponseSize` is clamped below `MaxInt64`

`ResponseBody` reads `maxSize+1` bytes so it can tell "at the limit" from "over the limit". A caller passing `math.MaxInt64` to mean "no practical limit" would overflow that into a negative count, and `io.LimitReader` reads NOTHING when N is negative — every response would come back empty, with no error. The `min(maxSize, math.MaxInt64-1)` clamp is important; `TestMaxResponseSize_HugeLimitDoesNotOverflow` pins it.

## The seam

SSRF hardening is centralized in [client.go](client.go) (`safeTransport`, `guardedDialContext`, `publicIPs`, `filterPublicIPs`, `baseTransport`). Public/non-public IP classification is delegated to `benpate/uri` (`uri.IsPublicIP`). To change what counts as "non-public," change it there, not here.
