## Low-Level Implementation

### The Journey of an 'A' Record Request

#### 1. Startup and Listening (`cmd/app/main.go`)

*   **`main()`**: The application starts. It initializes the UDP connection on port `5354`, creates the `DNSResolver` pointing to the upstream DNS, and wraps the core `DNSHandler` with a `RateLimitedHandler`.
*   **`serveDNS()`**: The server is now in its infinite `for` loop, blocked on the `conn.ReadFromUDP(buf)` call, waiting for a request to arrive.

#### 2. The Request Arrives (`cmd/app/main.go`)

*   Your `dig` command sends a DNS query packet over UDP to `localhost:5354`.
*   **`serveDNS()`**: The `ReadFromUDP` call unblocks. It reads the packet's bytes into the `buf` buffer and gets your client address (`127.0.0.1` plus a random port).
*   **`go server.HandleDNSRequest(...)`**: A new goroutine is launched to handle this specific request, allowing the `serveDNS` loop to immediately go back to waiting for the next request. This is key for concurrency.

#### 3. Request Parsing and Handling (`server/request.go`)

*   **`HandleDNSRequest()`**: This is the entry point for our new goroutine.
    *   It creates a `context` and stores your client IP (`127.0.0.1`) in it for rate-limiting purposes.
    *   It calls **`parseRequest()`** on the raw bytes from `buf`.
*   **`parseRequest()`**:
    *   It reads the first two bytes to get the **Transaction ID** (e.g., `5134`).
    *   It calls `NewDomainParser().Parse()` which reads the "Question" section of the packet, correctly decoding `google.com`.
    *   It then reads the **Query Type**, which is `1` for an `A` record.
    *   It returns the `txnID`, `domain`, and `qtype` back to `HandleDNSRequest`.
*   **`HandleDNSRequest()`**: Now with the parsed data, it calls **`resolveDomain()`**.

#### 4. Domain Resolution (`server/request.go` -> `server/handler.go` -> `server/resolver.go`)

*   **`resolveDomain()`** (`request.go`):
    *   It calls `records.GetHandler(1)` to get the specific handler for `A` records.
    *   It then calls `handler.HandleQuery(ctx, "google.com", 1)`.
*   **`HandleQuery()`** (`RateLimitedHandler` in `handler.go`):
    *   It gets your IP from the context.
    *   It calls `limiter.AllowQuery("127.0.0.1")`. Since this is the first query, it's allowed.
    *   It passes the call through to the wrapped `dnsHandler`.
*   **`HandleQuery()`** (`dnsHandler` in `handler.go`):
    *   It calls `h.resolver.Resolve(ctx, ...)` to do the actual lookup.
*   **`Resolve()`** (`resolver.go`):
    *   It calls its own `ResolveDomain("google.com", 1)` method.
*   **`ResolveDomain()`** (`resolver.go`):
    *   It looks in its `strategies` map for key `1` (the `A` type). It finds the `IPResolution` strategy that was configured at startup.
    *   It calls `strategy.Resolve("google.com")`.

#### 5. Forwarding to Upstream (`server/strategy.go`)

*   **`Resolve()`** (`IPResolution` in `strategy.go`):
    *   This is where the external query happens. It calls `r.forwarder.resolver.LookupIPAddr(..., "google.com")`.
    *   The `forwarder`'s custom `Dial` function ensures this lookup is sent directly to your configured upstream server (e.g., `8.8.8.8:53`).
    *   The upstream server responds with a list of IP addresses for `google.com`.
    *   The function loops through the returned addresses. For each one, it calls `r.isValidIP()`, which in this case is the `isIPv4` function.
    *   The first IPv4 address it finds (e.g., `142.250.193.110`) is returned as a string.

#### 6. The Response Bubbles Up

The IP address string "142.250.193.110" is now returned all the way back up the call stack:
*   From `strategy.go` to `resolver.go`.
*   From `resolver.go` to `handler.go`.
*   From `handler.go` back to `resolveDomain()` in `request.go`.

#### 7. Building the Final Output (`server/request.go` -> `server/records/a.go`)

*   **`resolveDomain()`** (`request.go`): It receives the IP string. It calls the `A` record handler's `ValidateData()` method to ensure it's a valid IP. It is. The function returns the record handler and the IP string data.
*   **`HandleDNSRequest()`** (`request.go`): It receives the resolved data and calls **`buildAndSendResponse()`**.
*   **`buildAndSendResponse()`** (`request.go`):
    *   It first calls **`BuildResponse()`** (from `response_builder.go`) which constructs the DNS header (copying the original Transaction ID, setting flags for a successful response) and re-creates the "Question" section. This gives the base of our response packet.
    *   It then calls `handler.BuildAnswer("google.com", "142.250.193.110", ...)`
*   **`BuildAnswer()`** (`ARecordHandler` in `server/records/a.go`):
    *   This function knows exactly how to format an `A` record.
    *   It takes the IP string "142.250.193.110" and parses it into a 4-byte slice (`[]byte{142, 250, 193, 110}`).
    *   It constructs the full answer record in wire format:
        *   **NAME**: A pointer to the `google.com` name in the question section.
        *   **TYPE**: `1` (A record)
        *   **CLASS**: `1` (IN)
        *   **TTL**: The default TTL (e.g., 300 seconds).
        *   **RDLENGTH**: `4` (for the 4 bytes of the IP).
        *   **RDATA**: The 4-byte IP address itself.
    *   It returns these bytes.
*   **`buildAndSendResponse()`** (`request.go`): It appends the answer bytes to the header and question bytes, creating the complete response packet.
*   **`conn.WriteToUDP(response, clientAddr)`**: The final, complete packet is sent back to your `dig` client.

Your `dig` client receives these raw bytes, parses them, and displays the familiar, human-readable output showing the `ANSWER SECTION` with the IP address.


