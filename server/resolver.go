// server/resolver.go
package server

import (
	"errors"
)

// DNSResolver defines a simple resolver with in-memory domain-to-IP mappings
type DNSResolver struct {
	domainIPMap map[string]string
}

// NewDNSResolver initializes and returns a new DNSResolver
func NewDNSResolver() *DNSResolver {
	return &DNSResolver{
		domainIPMap: map[string]string{
			"example.com": "93.184.216.34",
			"localhost": "127.0.0.1",
		},
	}
}

// ResolveDomain resolves a domain name to an IP address
func (r *DNSResolver) ResolveDomain(domain string) (string, error) {
	if ip, exists := r.domainIPMap[domain]; exists {
		return ip, nil
	}
	return "", errors.New("domain not found")
}
