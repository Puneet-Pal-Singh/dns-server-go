## Project Tasks

Status legend: [x] done, [ ] todo

### Core Server
- [x] UDP DNS server listening on `:5354` with concurrent request handling (`cmd/app/main.go`)
- [x] Configurable upstream DNS via `UPSTREAM_DNS` env
- [x] Basic logging of requests and results
- [x] Rate limiting (token bucket) with per-IP buckets and cleanup (`server/ratelimit.go`)
- [x] Configurable rate limit via `RATE_LIMIT_CAPACITY`, `RATE_LIMIT_REFILL`

### Request Parsing & Response Building
- [x] Robust domain name parser with compression pointer support (`server/message_parser.go`)
- [x] Request parsing: transaction ID, QNAME, QTYPE/QCLASS extraction (`server/request.go`)
- [x] Response builder composing header, question and answers (`server/response_builder.go`)
- [ ] Correct answer count (ANCOUNT) and multi-answer handling in header (currently static 1)
- [ ] Proper error responses (NXDOMAIN, NOTIMP, REFUSED) beyond generic server failure

### Record Handling (authoritative formatting logic)
- [x] Unified record handler interface (`server/records/base.go`)
- [x] A (IPv4) record handler
- [x] AAAA (IPv6) record handler
- [x] CNAME record handler
- [x] MX record handler (with `Preference` and `Exchange`)
- [x] TXT record handler (single and multi-string)
- [x] NS record handler
- [x] Record data validation and wire-format construction

### Resolution Strategy (data lookup)
- [x] Forwarder using `net.Resolver` with custom dialer to upstream (`server/strategy.go`)
- [x] Strategy pattern for resolution (`server/resolver.go`, `server/strategy.go`)
- [x] A/AAAA lookups via upstream
- [ ] Return full answers for non-A/AAAA types (MX/TXT/CNAME/NS) from upstream
- [ ] Local zone or static records support (file or in-memory map)
- [ ] Caching layer with TTL respect and negative caching

### Middleware / Policies
- [x] Rate-limited handler wrapper (`server/handler.go`)
- [ ] Structured logging (fields, request IDs)
- [ ] Metrics and observability (Prometheus counters, histograms)
- [ ] Access controls (allow/deny lists)

### Deployment & Ops
- [x] Multi-stage Dockerfile producing a distroless image
- [x] docker-compose exposing UDP 5354 with envs
- [ ] Helm chart / Kubernetes manifests
- [ ] Health and readiness probes
- [ ] Graceful shutdown and lifecycle hooks

### Protocol Completeness
- [ ] TCP fallback for truncated responses
- [ ] EDNS(0) basic support
- [ ] Support multiple questions per query (if needed)
- [ ] Recursion desired/ad flags handling
- [ ] Proper name compression in responses across sections

### Testing
- [x] Unit tests for record handlers (A/AAAA/CNAME/MX/TXT/NS)
- [x] Unit tests for domain parser (including compression)
- [x] Unit tests for rate limiter (concurrency, refill, cleanup)
- [ ] End-to-end tests: query → response bytes for each type
- [ ] Resolver tests aligned with current behavior for MX/TXT/CNAME/NS

### Known Gaps / Tech Debt
- [ ] Resolver only returns IP strings for A/AAAA; other types currently unsupported in `ResolveDomain`, while handlers exist
- [ ] Response building path may duplicate the answer: `BuildResponse` returns an answer and `buildAndSendResponse` appends again; fix and align ANCOUNT
- [ ] Implement and return appropriate DNS RCODEs (NXDOMAIN, REFUSED, NOTIMP)
- [ ] Validate and clamp TTLs; propagate upstream TTLs when forwarding

### Nice-to-haves (later)
- [ ] Admin API or config file for static zones
- [ ] Query logging with sampling
- [ ] Rate limiter persistence/redis option
- [ ] IPv6/dual-stack deployment examples


