// server/resolver.go
package server

import (
	"context"
	"errors"

	"github.com/Puneet-Pal-Singh/dns-server-go/server/records"
)

// ResolutionContext holds query context information
type ResolutionContext struct {
	Domain string
	QType  uint16
}

// DNSResolver coordinates resolution strategies
type DNSResolver struct {
	forwarder  *Forwarder
	strategies map[uint16]ResolutionStrategy
}

// NewDNSResolver initializes a new DNSResolver
func NewDNSResolver(upstream string) *DNSResolver {
	f := NewForwarder(upstream)
	return &DNSResolver{
		forwarder: f,
		strategies: map[uint16]ResolutionStrategy{
			records.TypeA:    NewIPResolution(f, isIPv4),
			records.TypeAAAA: NewIPResolution(f, isIPv6),
		},
	}
}

// ResolveDomain resolves a domain using the appropriate strategy
func (r *DNSResolver) ResolveDomain(domain string, qtype uint16) (string, error) {
	strategy, exists := r.strategies[qtype]
	if !exists {
		return "", errors.New("unsupported query type")
	}
	return strategy.Resolve(domain)
}

func (r *DNSResolver) Resolve(ctx context.Context, rc ResolutionContext) (interface{}, error) {
	handler, ok := records.GetHandler(rc.QType)
	if !ok {
		return nil, errors.New("unsupported query type")
	}

	ip, err := r.ResolveDomain(rc.Domain, rc.QType)
	if err != nil {
		return nil, err
	}

	if err := handler.ValidateData(ip); err != nil {
		return nil, err
	}

	return ip, nil
}
