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

// safeTransport is the SSRF-hardened base transport used for every request by
// default. Its dialer refuses to connect to non-public addresses.
//
// It is built once, not per request, so that connections are pooled.
var safeTransport = newGuardedTransport()

// NewHTTPClient returns an *http.Client backed by remote's SSRF-hardened
// transport, for libraries (e.g. golang.org/x/oauth2) that accept an
// *http.Client but not a remote.Transaction. Like every remote request, it
// refuses to connect to non-public addresses unless allowPrivateIPs is TRUE.
// The returned client is fresh, but shares the pooled safeTransport singleton.
func NewHTTPClient(allowPrivateIPs bool) *http.Client {

	return &http.Client{
		Timeout:       defaultTimeout,
		Transport:     baseTransport(allowPrivateIPs),
		CheckRedirect: guardedRedirect,
	}
}

// guardedRedirect is the http.Client CheckRedirect policy used by the standalone
// clients from NewHTTPClient. It caps the redirect chain at maxRedirects.
func guardedRedirect(_ *http.Request, via []*http.Request) error {

	const location = "remote.guardedRedirect"

	// There is no host allow-list to re-check here, because a standalone client has
	// none. The private-IP guard still fires on every hop, since it lives in the dialer.

	// RULE: A redirect chain has to end somewhere.
	if len(via) >= maxRedirects {
		return derp.BadRequest(location, "Too many redirects")
	}

	return nil
}

// baseTransport picks the round-tripper for a client: the SSRF-hardened
// safeTransport, or http.DefaultTransport when private addresses are allowed.
func baseTransport(allowPrivateIPs bool) http.RoundTripper {

	// This is the ONE place the guarded-vs-unguarded choice is made. NewHTTPClient and
	// buildClient both call it, so the two can never drift on which transport is used.

	// RULE: Drop the guard only when the caller has explicitly asked for it
	// (local development, self-federation, a test server on loopback).
	if allowPrivateIPs {
		return http.DefaultTransport
	}

	return safeTransport
}

// newGuardedTransport returns a transport whose dialer rejects connections to
// non-public addresses.
func newGuardedTransport() *http.Transport {

	// Start from a clone of the standard transport, to inherit its pooling,
	// proxy, and TLS defaults rather than re-deriving them here.
	var transport *http.Transport

	if base, ok := http.DefaultTransport.(*http.Transport); ok {
		transport = base.Clone()
	} else {
		transport = &http.Transport{}
	}

	dialer := &net.Dialer{
		Timeout:   defaultTimeout,
		KeepAlive: 30 * time.Second,
	}

	transport.DialContext = guardedDialContext(dialer.DialContext)

	return transport
}

// guardedDialContext wraps an inner DialContext so that it refuses to connect to
// a non-public address, delegating the actual connection to inner.
func guardedDialContext(inner dialContextFunc) dialContextFunc {

	const location = "remote.guardedDialContext"

	// The guard sits in the dialer, not in the request path, so it re-runs on every
	// redirect hop and on every pooled reconnect, with nothing left to bypass it.
	return func(ctx context.Context, network string, address string) (net.Conn, error) {

		// Split the address into its host and port.
		host, port, err := net.SplitHostPort(address)

		if err != nil {
			return nil, derp.Wrap(err, location, "Invalid dial address", address)
		}

		// Resolve the host (or use the IP literal) and confirm every address is public.
		ips, err := publicIPs(ctx, host)

		if err != nil {
			return nil, derp.Wrap(err, location, "Unable to validate host address", host)
		}

		// Dial a validated IP literal directly, trying each address in turn. Dialing the
		// literal (not the name) is what makes this safe against DNS rebinding: the address
		// that was checked is the address that gets connected to.
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

// publicIPs resolves host and returns its public IP addresses, erroring when
// none of them is public.
func publicIPs(ctx context.Context, host string) ([]net.IP, error) {

	const location = "remote.publicIPs"

	// If host is an IP literal, parse and check it directly without DNS resolution.
	if ip := net.ParseIP(host); ip != nil {
		if !uri.IsPublicIP(ip) {
			return nil, derp.BadRequest(location, "Blocked connection to non-public address (1)", ip)
		}
		return []net.IP{ip}, nil
	}

	// Otherwise resolve the host and check every candidate address.
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to resolve host", host)
	}

	if len(addrs) == 0 {
		return nil, derp.BadRequest(location, "No addresses found for host", host)
	}

	// Keep only the public addresses, dropping any non-public ones.
	ips := filterPublicIPs(addrs)

	// Reject only when NO public address remains — a host that resolves entirely
	// to private space is the real SSRF / DNS-rebinding case.
	if len(ips) == 0 {
		return nil, derp.BadRequest(location, "Blocked connection to non-public address (2)", host)
	}

	// These addresses are cleared for takeoff
	return ips, nil
}

// filterPublicIPs returns only the public addresses from addrs, and an empty
// slice when none of them is public.
func filterPublicIPs(addrs []net.IPAddr) []net.IP {

	// A misconfigured host may advertise a non-public record (an IPv6 link-local
	// address leaked into DNS, say) alongside its real ones. Dropping just the bad
	// addresses lets the caller connect via the good ones instead of failing outright.
	ips := make([]net.IP, 0, len(addrs))

	for _, addr := range addrs {
		if uri.IsPublicIP(addr.IP) {
			ips = append(ips, addr.IP)
		}
	}

	return ips
}
