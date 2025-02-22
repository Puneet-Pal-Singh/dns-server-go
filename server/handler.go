package server

type DNSHandler interface {
	HandleQuery(domain string) (string, error)
}

type dnsHandler struct {
	resolver *DNSResolver
}

func NewDNSHandler(resolver *DNSResolver) DNSHandler {
	return &dnsHandler{resolver: resolver}
}

func (h *dnsHandler) HandleQuery(domain string) (string, error) {
	return h.resolver.ResolveDomain(domain)
}
