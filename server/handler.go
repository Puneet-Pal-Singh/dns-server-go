package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/Puneet-Pal-Singh/dns-server-go/server/records"
)

type DNSHandler interface {
	HandleQuery(ctx context.Context, domain string, qtype uint16) (interface{}, error)
}

type dnsHandler struct {
	resolver *DNSResolver
}

// RateLimitedHandler wrapper
type RateLimitedHandler struct {
	handler   DNSHandler
	limiter   RateLimiter
	mu        sync.Mutex
	blockedIP map[string]time.Time
}

func NewDNSHandler(resolver *DNSResolver) DNSHandler {
	return &dnsHandler{resolver: resolver}
}

func (h *dnsHandler) HandleQuery(ctx context.Context, domain string, qtype uint16) (interface{}, error) {
	// return h.resolver.ResolveDomain(domain, qtype)
	switch qtype {
	case records.TypeA:
		return "192.0.2.1", nil

	case records.TypeMX:
		// Ensure proper MX record format
		if !strings.Contains(domain, ".") {
			return nil, fmt.Errorf("invalid domain for MX record: %s", domain)
		}
		return records.MXData{
			Preference: 10,
			Exchange:   fmt.Sprintf("mail.%s", strings.TrimSuffix(domain, ".")),
		}, nil

	case records.TypeCNAME:
		// validation for CNAME
		if !strings.HasPrefix(domain, "www.") {
			return nil, fmt.Errorf("CNAME record only available for www subdomain")
		}
		return strings.TrimPrefix(domain, "www."), nil

	case records.TypeTXT:
		// Fix: Return string array for TXT record
		return []string{"v=spf1 include:_spf.google.com ~all"}, nil

	default:
		return nil, fmt.Errorf("unsupported query type: %d", qtype)
	}
}

func NewRateLimitedHandler(handler DNSHandler, limiter RateLimiter) DNSHandler {
	return &RateLimitedHandler{
		handler:   handler,
		limiter:   limiter,
		blockedIP: make(map[string]time.Time),
	}
}

func (h *RateLimitedHandler) HandleQuery(ctx context.Context, domain string, qtype uint16) (interface{}, error) {
	ip, ok := GetClientIPFromContext(ctx)

	if !ok {
		log.Printf("[WARNING] Missing client IP in context for domain: %s", domain)
		return nil, errors.New("client IP missing")
	}

	// Add debug logging
	log.Printf("[RATE DEBUG] Checking rate limit for %s", ip)

	if !h.limiter.AllowQuery(ip) {
		h.mu.Lock()
		log.Printf("[RATE LIMIT] Blocked request from %s for %s", ip, domain)
		h.mu.Unlock()
		return nil, errors.New("rate limit exceeded")
	}

	return h.handler.HandleQuery(ctx, domain, qtype)
}

// Add missing context helper function
func GetClientIPFromContext(ctx context.Context) (string, bool) {
	if ip, ok := ctx.Value(clientIPKey).(string); ok {
		return ip, true
	}
	if host, _, err := net.SplitHostPort(ctx.Value("peer").(string)); err == nil {
		return host, true
	}
	return "unknown", false
}
