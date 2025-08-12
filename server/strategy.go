// server/strategy.go
package server

import (
	"context"
	"errors"
	"net"
)

// ResolutionStrategy defines a DNS resolution method
type ResolutionStrategy interface {
	Resolve(domain string) (string, error)
}

// Forwarder handles upstream DNS queries
type Forwarder struct {
	resolver *net.Resolver
}

// NewForwarder initializes a new Forwarder
func NewForwarder(upstream string) *Forwarder {
	return &Forwarder{
		resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
				d := net.Dialer{}
				return d.DialContext(ctx, network, upstream)
			},
		},
	}
}

// IPResolution is a generic resolver that filters IP addresses
type IPResolution struct {
	forwarder *Forwarder
	isValidIP func(net.IP) bool
}

// NewIPResolution creates a new instance of IPResolution
func NewIPResolution(f *Forwarder, filterFunc func(net.IP) bool) *IPResolution {
	return &IPResolution{forwarder: f, isValidIP: filterFunc}
}

// Resolve filters the resolved IP addresses based on the provided filter function
func (r *IPResolution) Resolve(domain string) (string, error) {
	addrs, err := r.forwarder.resolver.LookupIPAddr(context.Background(), domain)
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		if r.isValidIP(addr.IP) {
			return addr.IP.String(), nil
		}
	}
	return "", errors.New("no valid record found")
}

// Helper functions for filtering IP addresses
func isIPv4(ip net.IP) bool {
	return ip.To4() != nil
}

func isIPv6(ip net.IP) bool {
	return ip.To16() != nil && ip.To4() == nil
}
