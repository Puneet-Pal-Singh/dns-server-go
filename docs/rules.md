## Project Rules

### Coding Principles (Go)
- Keep it idiomatic and simple; avoid unnecessary abstractions.
- Single Responsibility per function; favor small, testable units.
- Prefer early returns; avoid deep nesting.
- Dependency injection for external services and policies.
- Handle errors explicitly; no silent failures or panics in core paths.
- Prefer standard library; minimize external deps.
- Use interfaces at boundaries; concrete types internally.

### Architecture Principles
- Separation of concerns: parsing, resolving, records, transport.
- Open-Closed: extend with new record types or strategies without modifying core logic.
- High cohesion, loose coupling among `server` subpackages.
- Avoid global state; pass context and dependencies.

### Logging & Observability
- Use structured logging with request ID, client IP, domain, qtype.
- Add metrics: request counts, error counts, latency, rate-limit denials.

### Testing
- Unit tests for all record handlers and parsers.
- E2E tests for request→response bytes for each record type.
- Fuzz tests for parser on malformed inputs (future).

### Security & Ops
- Run as non-root in containers; use distroless base.
- Expose only UDP 5354; consider TCP fallback on demand.
- Rate limiting enabled by default with safe defaults.

### References
- Implementation inspiration: `miekg/dns`.


