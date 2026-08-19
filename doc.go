// Package remote is a chainable HTTP client for Go, wrapping net/http with
// sensible defaults and a fluent API.
//
// A request is built by chaining methods onto a Transaction and finished with
// Send, which performs the request and decodes the response in one step:
//
//	err := remote.Get("https://example.com/users").
//		Result(&users).
//		Send()
//
// Send returns a non-nil error whenever the response status falls outside the
// 200-299 range. Use Result for the success body and Error for the failure
// body; both are decoded according to the response Content-Type, so JSON, XML,
// and plain text are handled without any extra work at the call site.
//
// The package covers four areas:
//
//   - Building a request — Get, Post, Put, Patch, and Delete start a
//     Transaction, which Header, Query, Form, Body, JSON, and XML fill in.
//   - Reading a response — Result and Error name the objects to decode into,
//     and the Response* methods expose the raw http.Response underneath.
//   - Hooks — an Option carries BeforeRequest, ModifyRequest, and AfterRequest
//     functions that observe or rewrite a transaction as it runs. The options
//     subpackage ships the common ones (auth headers, debugging, a test server).
//   - Guardrails — every request is bounded by a timeout, a redirect cap, and a
//     maximum response size, and may be restricted to an allow-list of hosts.
//
// # SSRF hardening is the default
//
// Requests refuse to connect to non-public addresses (loopback, private,
// link-local including the cloud metadata endpoint) unless the caller sets
// AllowPrivateIPs(true). The guard lives in the transport's dialer, which
// resolves the host and connects to a validated IP literal, so it is
// DNS-rebinding-safe and re-runs on every redirect hop.
//
// NewHTTPClient hands the same guarded transport to libraries that accept an
// *http.Client but not a Transaction, such as golang.org/x/oauth2.
package remote
