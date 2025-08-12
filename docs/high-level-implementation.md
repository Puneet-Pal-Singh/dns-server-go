## High-Level Implementation

### Overview
This project implements a minimalist DNS server over UDP with a modular architecture:
- Transport loop: reads UDP packets and dispatches to handler
- Parsing: decodes QNAME and QTYPE with compression support
- Resolution: forwards queries upstream using strategy pattern
- Records: builds RFC-compliant wire-format answers per record type
- Middleware: per-IP token-bucket rate limiter

### Flow
1. `cmd/app/main.go` initializes the UDP listener, resolver, and wraps the handler with rate limiting.
2. `server.HandleDNSRequest` parses the request: extracts `txnID`, `domain`, and `qtype` via `NewDomainParser`.
3. `server.request.resolveDomain` fetches the `RecordHandler` for `qtype` and calls `DNSHandler.HandleQuery`.
4. `dnsHandler.HandleQuery` validates the type and invokes `resolver.Resolve` with a `ResolutionContext`.
5. `DNSResolver.Resolve` chooses a strategy and currently supports A/AAAA via upstream; validates data via the record handler.
6. `server.BuildResponse` and `RecordHandler.BuildAnswer` compose the DNS response bytes.
7. Response is sent back over UDP.

### Key Components
- `server/message_parser.go`: Robust domain parser with compression handling.
- `server/request.go`: Orchestrates request parsing and response writing.
- `server/handler.go`: Core handler and rate-limited wrapper.
- `server/resolver.go`: Coordinates strategies; validates via handlers.
- `server/strategy.go`: Forwarder and IP filtering strategies.
- `server/response_builder.go`: Header, question, and answer construction.
- `server/records/*`: Record-specific validation and wire formatting (A, AAAA, CNAME, MX, TXT, NS).

### Current Behavior Notes
- Only A and AAAA are resolved via upstream. Other types have handlers but are not looked up; they will error in resolver.
- Response header `ANCOUNT` is static (1) in builder; `buildAndSendResponse` appends answers again, which may duplicate bytes. Needs alignment and dynamic counts.
- Error responses use a single `responseServerFailure` flag; richer RCODE mapping is pending.

### Configuration
- `UPSTREAM_DNS`: address of upstream DNS (default `8.8.8.8:53`).
- `RATE_LIMIT_CAPACITY`: bucket size per IP (default 100).
- `RATE_LIMIT_REFILL`: seconds per token (default 1s).

### Deployment
- Multi-stage Docker builds to a distroless image.
- docker-compose exposes UDP 5354 with envs.

### Next Steps
- Implement upstream resolution for MX/TXT/CNAME/NS, propagate TTLs.
- Correct response building: dynamic counts and avoid duplicate appends.
- Add caching and local zone support.
- Add TCP fallback and EDNS(0) basics.
- Add structured logging and metrics.

### Dry Run for 'A' Record Request:
Step-by-step dry run of what happens when you issue the command `dig @localhost -p 5354 google.com A`.