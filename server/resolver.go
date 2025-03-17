// server/resolver.go
package server

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/Puneet-Pal-Singh/dns-server-go/server/records"
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

type ResolutionContext struct {
	QType  uint16
	Domain string
}

// NewDNSResolver initializes and returns a new DNSResolver
func NewDNSResolver(upstream string) *DNSResolver {
	return &DNSResolver{
		cache:        make(map[string]cacheEntry),
		upstreamAddr: upstream,
	}

}

// ResolveDomain resolves a domain name to an IP address
func (r *DNSResolver) ResolveDomain(domain string, qtype uint16) (string, error) {
	// Check cache
	entry, exists := r.checkCache(domain)
	if exists {
		return entry.ip, nil
	}

	// Forward query
	ip, err := r.forwardQuery(domain, qtype)
	if err != nil {
		return "", err
	}

	// Cache with 5min TTL
	r.updateCache(domain, ip)
	return ip, nil
}

// Update forwardQuery to handle different record types
func (r *DNSResolver) forwardQuery(domain string, qtype uint16) (string, error) {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, "udp", r.upstreamAddr)
		},
	}

	var ip string
	switch qtype {
	case records.TypeA:
		addrs, err := resolver.LookupIPAddr(context.Background(), domain)
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			if addr.IP.To4() != nil {
				ip = addr.IP.String()
				break
			}
		}
	case records.TypeAAAA:
		addrs, err := resolver.LookupIPAddr(context.Background(), domain)
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			if addr.IP.To16() != nil {
				ip = addr.IP.String()
				break
			}
		}
	default:
		return "", errors.New("unsupported query type")
	}

	if ip == "" {
		return "", errors.New("no records found")
	}
	return ip, nil
}

func (r *DNSResolver) Resolve(ctx context.Context, rc records.ResolutionContext) (interface{}, error) {
	handler, ok := records.GetHandler(rc.QType)
	if !ok {
		return nil, errors.New("unsupported query type")
	}

	// Common resolution logic
	if entry, exists := r.checkCache(rc.Domain); exists {
		return entry.ip, nil
	}

	ip, err := r.forwardQuery(rc.Domain, rc.QType)
	if err != nil {
		return nil, err
	}

	if err := handler.ValidateData(ip); err != nil {
		return nil, err
	}

	r.updateCache(rc.Domain, ip)
	return ip, nil
}

// Extract common cache operations
func (r *DNSResolver) checkCache(domain string) (cacheEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, exists := r.cache[domain]
	return entry, exists && time.Now().Before(entry.expiresAt)
}

func (r *DNSResolver) updateCache(domain, ip string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache[domain] = cacheEntry{
		ip:        ip,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
}
