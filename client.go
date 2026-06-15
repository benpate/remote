package remote

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/benpate/derp"
	"github.com/benpate/uri"
)

// defaultTimeout is the default time limit applied to a request and its dialer.
const defaultTimeout = 1 * time.Minute

// maxRedirects caps how many redirects a request will follow.
const maxRedirects = 5

// dialContextFunc matches the signature of net.Dialer.DialContext.
type dialContextFunc func(ctx context.Context, network string, address string) (net.Conn, error)

// safeTransport is the shared, SSRF-hardened base transport used for every
// request by default. Its dialer refuses to connect to non-public addresses,
// and it is shared (not rebuilt per request) so connections are pooled.
var safeTransport = newGuardedTransport()

// newGuardedTransport returns a transport whose dialer rejects connections to
// non-public addresses. It is cloned from http.DefaultTransport so it keeps the
// standard pooling, proxy, and TLS defaults.
func newGuardedTransport() *http.Transport {

	var transport *http.Transport

	if base, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = base.Clone()
	} else {
		transport = &http.Transport{}
	}

	dialer := &net.Dialer{Timeout: defaultTimeout, KeepAlive: 30 * time.Second}
	transport.DialContext = guardedDialContext(dialer.DialContext)

	return transport
}

// limitRedirects is the CheckRedirect policy that caps a redirect chain.
func limitRedirects(_ *http.Request, via []*http.Request) error {

	const location = "remote.limitRedirects"

	if len(via) >= maxRedirects {
		return derp.BadRequest(location, "Too many redirects")
	}

	return nil
}

// guardedDialContext wraps an inner DialContext so it refuses to connect to any
// non-public address, while delegating the actual connection to inner. The host
// is resolved and every candidate address is checked; the connection is then
// made to a validated IP literal, so it cannot be re-pointed at a private
// address via DNS rebinding.
func guardedDialContext(inner dialContextFunc) dialContextFunc {

	const location = "remote.guardedDialContext"

	return func(ctx context.Context, network string, address string) (net.Conn, error) {

		host, port, err := net.SplitHostPort(address)

		if err != nil {
			return nil, derp.Wrap(err, location, "Invalid dial address", address)
		}

		// Resolve the host (or use the IP literal) and confirm every address is public.
		ips, err := publicIPs(ctx, host)

		if err != nil {
			return nil, derp.Wrap(err, location, "Unable to validate host address", host)
		}

		// Dial a validated IP literal directly (rebinding-safe), trying each in turn.
		var lastErr error

		for _, ip := range ips {
			conn, err := inner(ctx, network, net.JoinHostPort(ip.String(), port))

			if err != nil {
				lastErr = err
				continue
			}

			return conn, nil
		}

		return nil, derp.Wrap(lastErr, location, "Unable to connect to host", host)
	}
}

// publicIPs resolves host to its IP addresses and returns them only if every one
// is public. A host that is already an IP literal is checked directly. Any
// non-public address causes an error, so the caller never connects to it.
func publicIPs(ctx context.Context, host string) ([]net.IP, error) {

	const location = "remote.publicIPs"

	if ip := net.ParseIP(host); ip != nil {
		if !uri.IsPublicIP(ip) {
			return nil, derp.BadRequest(location, "Blocked connection to non-public address (1)", ip)
		}
		return []net.IP{ip}, nil
	}

	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to resolve host", host)
	}

	if len(addrs) == 0 {
		return nil, derp.BadRequest(location, "No addresses found for host", host)
	}

	ips := make([]net.IP, 0, len(addrs))

	for _, addr := range addrs {
		if !uri.IsPublicIP(addr.IP) {
			return nil, derp.BadRequest(location, "Blocked connection to non-public address (2)", addr.IP)
		}
		ips = append(ips, addr.IP)
	}

	return ips, nil
}
