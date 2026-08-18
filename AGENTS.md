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

## The seam

SSRF hardening is centralized in [client.go](client.go) (`safeTransport`, `guardedDialContext`, `publicIPs`, `filterPublicIPs`, `baseTransport`). Public/non-public IP classification is delegated to `benpate/uri` (`uri.IsPublicIP`). To change what counts as "non-public," change it there, not here.
