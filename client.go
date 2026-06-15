package remote

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/benpate/uri"
)

// maxRedirects caps how many redirects the SafeClient will follow.
const maxRedirects = 5

// dialContextFunc matches the signature of net.Dialer.DialContext.
type dialContextFunc func(ctx context.Context, network string, address string) (net.Conn, error)

// DefaultClient returns an HTTP client with a reasonable timeout.
func DefaultClient() *http.Client {

	return &http.Client{
		Timeout: 10 * time.Second,
	}
}

// SafeClient returns an HTTP client that is hardened against SSRF: its dialer
// refuses to connect to any non-public address (loopback, private, link-local
// including the cloud-metadata endpoint, etc.). Use it with Transaction.Client
// when the request URL is untrusted or user-supplied.
func SafeClient() *http.Client {
	return newSafeClient(uri.IsPublicIP)
}

// newSafeClient builds an SSRF-hardened client whose dialer only connects to
// addresses that isPublic accepts.
func newSafeClient(isPublic func(net.IP) bool) *http.Client {

	baseDialer := &net.Dialer{Timeout: 10 * time.Second}

	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: guardedDialContext(baseDialer.DialContext, isPublic),
		},
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("remote: too many redirects")
			}
			return nil
		},
	}
}

// guardClient returns a copy of client whose dialer additionally rejects
// connections to non-public addresses. The caller's transport settings AND its
// existing dialer are preserved: the IP check is layered on top of the client's
// own DialContext, not swapped in for it.
//
// If the client uses a non-standard http.RoundTripper (not an *http.Transport),
// the dialer cannot be augmented and a guarded default transport is used instead.
func guardClient(client *http.Client, isPublic func(net.IP) bool) *http.Client {

	if client == nil {
		client = DefaultClient()
	}

	transport := cloneTransport(client.Transport)

	// Preserve the transport's own dialer (or the standard default) and wrap it.
	inner := transport.DialContext
	if inner == nil {
		inner = (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext
	}
	transport.DialContext = guardedDialContext(inner, isPublic)

	clone := *client
	clone.Transport = transport
	return &clone
}

// guardedDialContext wraps an inner DialContext so it refuses to connect to any
// non-public address, while delegating the actual connection to inner. The host
// is resolved and every candidate address is checked; the connection is then
// made to a validated IP literal, so it cannot be re-pointed at a private
// address via DNS rebinding.
func guardedDialContext(inner dialContextFunc, isPublic func(net.IP) bool) dialContextFunc {

	return func(ctx context.Context, network string, address string) (net.Conn, error) {

		host, port, err := net.SplitHostPort(address)

		if err != nil {
			return nil, err
		}

		// Resolve the host (or use the IP literal) and confirm every address is public.
		ips, err := publicIPs(ctx, host, isPublic)

		if err != nil {
			return nil, err
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

		return nil, lastErr
	}
}

// publicIPs resolves host to its IP addresses and returns them only if every one
// is public. A host that is already an IP literal is checked directly. Any
// non-public address causes an error, so the caller never connects to it.
func publicIPs(ctx context.Context, host string, isPublic func(net.IP) bool) ([]net.IP, error) {

	if ip := net.ParseIP(host); ip != nil {
		if !isPublic(ip) {
			return nil, blockedAddressError(ip)
		}
		return []net.IP{ip}, nil
	}

	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)

	if err != nil {
		return nil, err
	}

	if len(addrs) == 0 {
		return nil, errors.New("remote: no addresses found for " + host)
	}

	ips := make([]net.IP, 0, len(addrs))

	for _, addr := range addrs {
		if !isPublic(addr.IP) {
			return nil, blockedAddressError(addr.IP)
		}
		ips = append(ips, addr.IP)
	}

	return ips, nil
}

// blockedAddressError returns the error used when a connection to a non-public
// address is refused.
func blockedAddressError(ip net.IP) error {
	return errors.New("remote: blocked connection to non-public address " + ip.String())
}

// cloneTransport returns a mutable *http.Transport copy of rt: the transport
// itself when rt is a standard *http.Transport, otherwise a clone of the default.
func cloneTransport(rt http.RoundTripper) *http.Transport {

	if transport, ok := rt.(*http.Transport); ok {
		return transport.Clone()
	}

	if transport, ok := http.DefaultTransport.(*http.Transport); ok {
		return transport.Clone()
	}

	return &http.Transport{}
}
