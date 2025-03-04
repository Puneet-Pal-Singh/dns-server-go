package server

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
	"log"
)

type DNSHandler interface {
	HandleQuery(ctx context.Context, domain string) (string, error)
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

func (h *dnsHandler) HandleQuery(ctx context.Context, domain string) (string, error) {
	return h.resolver.ResolveDomain(domain)
}

func NewRateLimitedHandler(handler DNSHandler, limiter RateLimiter) DNSHandler {
	return &RateLimitedHandler{
		handler:   handler,
		limiter:   limiter,
		blockedIP: make(map[string]time.Time),
	}
}

func (h *RateLimitedHandler) HandleQuery(ctx context.Context, domain string) (string, error) {
	ip, ok := GetClientIPFromContext(ctx)

	if !ok {
		log.Printf("[WARNING] Missing client IP in context for domain: %s", domain)
		return "", errors.New("client IP missing")
	}

	// Add debug logging
	log.Printf("[RATE DEBUG] Checking rate limit for %s", ip)
	
	if !h.limiter.AllowQuery(ip) {
		h.mu.Lock()
		log.Printf("[RATE LIMIT] Blocked request from %s for %s", ip, domain)
		h.mu.Unlock()
		return "", errors.New("rate limit exceeded")
	}

	return h.handler.HandleQuery(ctx, domain)
}

// Add missing context helper function
func GetClientIPFromContext(ctx context.Context) (string, bool) {
	if ip, ok := ctx.Value("client_ip").(string); ok {
		return ip, true
	}
	if host, _, err := net.SplitHostPort(ctx.Value("peer").(string)); err == nil {
		return host, true
	}
	return "unknown", false
}
