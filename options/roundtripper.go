package options

import (
	"net/http"

	"github.com/benpate/remote"
)

// WithRoundTripper is a remote.Option that wraps the request's SSRF-hardened base
// transport with the given middleware (for caching, custom headers, etc.). The
// middleware receives the base transport as "next" and must delegate to it.
func WithRoundTripper(wrap func(next http.RoundTripper) http.RoundTripper) remote.Option {
	return remote.Option{
		BeforeRequest: func(txn *remote.Transaction) error {
			txn.WithRoundTripper(wrap)
			return nil
		},
	}
}
