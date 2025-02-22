// server/resolver.go
package server

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

// DNSResolver defines a simple resolver with in-memory domain-to-IP mappings
type DNSResolver struct {
	cache        map[string]cacheEntry
	upstreamAddr string // e.g. "8.8.8.8:53"
	mu           sync.RWMutex
}

type cacheEntry struct {
	ip        string
	expiresAt time.Time
}

// NewDNSResolver initializes and returns a new DNSResolver
func NewDNSResolver(upstream string) *DNSResolver {
	return &DNSResolver{
		cache:        make(map[string]cacheEntry),
		upstreamAddr: upstream,
	}
}

// ResolveDomain resolves a domain name to an IP address
func (r *DNSResolver) ResolveDomain(domain string) (string, error) {
	// Check cache
	r.mu.RLock()
	entry, exists := r.cache[domain]
	r.mu.RUnlock()

	if exists && time.Now().Before(entry.expiresAt) {
		return entry.ip, nil
	}

	// Forward query
	ip, err := r.forwardQuery(domain)
	if err != nil {
		return "", err
	}

	// Cache with 5min TTL
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache[domain] = cacheEntry{
		ip:        ip,
		expiresAt: time.Now().Add(5 * time.Minute),
	}

	return ip, nil
}

func (r *DNSResolver) forwardQuery(domain string) (string, error) {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, "udp", r.upstreamAddr)
		},
	}

	ips, err := resolver.LookupHost(context.Background(), domain)
	if err != nil {
		return "", err
	}
	if len(ips) == 0 {
		return "", errors.New("no records found")
	}

	return ips[0], nil
}
