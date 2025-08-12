# Rate Limiting Subsystem

The DNS server includes a robust, per-IP rate-limiting feature to prevent abuse and ensure fair usage. This document provides an overview of its implementation and a detailed step-by-step walkthrough of its logic.

## Overview

The rate limiter uses a **Token Bucket** algorithm, a common and effective method for controlling request rates.

*   **Implementation:** The core logic resides in `server/ratelimit.go`.
*   **Per-IP Buckets:** Each unique client IP address is assigned its own token bucket. This ensures that one aggressive client cannot exhaust the server's capacity for other legitimate users.
*   **Concurrency Safe:** The system uses a mutex to protect access to the shared map of client buckets, making it safe for concurrent use by many goroutines.
*   **Automatic Cleanup:** A background goroutine periodically cleans up old, inactive IP entries from memory to prevent leaks over time.

### Configuration

The rate limiter is configured via two environment variables:

| Variable              | Description                               | Default |
| --------------------- | ----------------------------------------- | ------- |
| `RATE_LIMIT_CAPACITY` | The burst capacity of the token bucket.   | `100`   |
| `RATE_LIMIT_REFILL`   | The number of seconds it takes to add one token back to the bucket. | `1`     |

---

## Step-by-Step Dry Run

This dry run explains how the rate limiter processes requests from a new client.

**Assumptions:**
*   Server is running with default settings (100 capacity, 1s refill).
*   A new client with IP `192.168.1.10` makes requests.

### 1. Initialization

1.  **`cmd/app/main.go`**: On startup, `createRateLimiter()` is called.
2.  **`server/ratelimit.go`**: `NewTokenBucketRateLimiter(100, 1 * time.Second)` is invoked. It initializes a `TokenBucketRateLimiter` struct containing an empty `clients` map and starts the background cleanup routine.
3.  **`cmd/app/main.go`**: The `RateLimitedHandler` is created, wrapping the core `dnsHandler` with the rate limiter instance.

### 2. First Request from a New Client

1.  A request from `192.168.1.10` arrives. The `HandleDNSRequest` function in `server/request.go` extracts the IP and stores it in a `context`.
2.  The `RateLimitedHandler`'s `HandleQuery` method in `server/handler.go` is called.
3.  It calls `limiter.AllowQuery("192.168.1.10")`.
4.  **Inside `AllowQuery` (`ratelimit.go`):**
    *   The code locks the mutex for safe access.
    *   It checks the `clients` map for the IP `192.168.1.10`. **It is not found.**
    *   A **new token bucket** is created specifically for this IP. It is initialized with `capacity - 1` tokens (so, `99` tokens). The first token is consumed upon creation.
    *   The new bucket is stored in the map: `clients["192.168.1.10"] = bucket`.
    *   The mutex is unlocked.
    *   The function returns `true`.
5.  The request is allowed and proceeds to the resolver.

### 3. Subsequent 99 Requests (Instant)

*   For the next 99 requests, `AllowQuery` finds the existing bucket for `192.168.1.10`.
*   It checks if the bucket has tokens (`> 0`). It does.
*   It decrements the token count by one for each request and returns `true`.
*   After a total of 100 requests, the bucket's token count is now **0**.

### 4. The 101st Request (Instant) - The Block

1.  The 101st request arrives.
2.  **Inside `AllowQuery` (`ratelimit.go`):**
    *   The bucket for `192.168.1.10` is found.
    *   The `refill()` logic runs, but since no time has passed, no new tokens are added.
    *   The code checks if `bucket.tokens > 0`. **This is now false.**
    *   The function immediately returns `false`.
3.  **Back in `RateLimitedHandler` (`handler.go`):**
    *   The `if` condition `!h.limiter.AllowQuery(ip)` is now met.
    *   A log message is printed: `[RATE LIMIT] Blocked request...`.
    *   An error is returned, and the request is terminated. A `SERVFAIL` response is sent to the client.

### 5. A Request After a Pause

1.  The client waits for **2 seconds** and sends another request.
2.  **Inside `AllowQuery` (`ratelimit.go`):**
    *   The bucket is found.
    *   The **`refill()`** logic is executed.
    *   It calculates that 2 seconds have passed since the last refill.
    *   It adds `2` new tokens to the bucket (`2 seconds / 1s refill rate`). The bucket now has **2 tokens**.
    *   The `lastRefill` timestamp is updated.
    *   The check `bucket.tokens > 0` passes.
    *   One token is consumed (leaving 1 in the bucket), and the function returns `true`.
3.  The request is **allowed** to proceed successfully.


-------------------------------------------------------

## Step-by-Step Dry Run - 2


Let's trace the rate-limiting feature with the same level of detail. The core of this feature lives in `server/ratelimit.go` and is applied in `server/handler.go`.

For this dry run, let's assume the server was started with the default rate limit settings:
*   `RATE_LIMIT_CAPACITY`: **100** (each new IP gets a bucket with 100 tokens)
*   `RATE_LIMIT_REFILL`: **1** (one token is added back to the bucket every second)

### The Journey of a Rate-Limited 'A' Record Request

#### 1. Initialization (`cmd/app/main.go` and `server/ratelimit.go`)

*   **`main()`**: The application starts.
*   **`createRateLimiter()`**: This function is called. It reads the environment variables and finds the defaults (100 capacity, 1s refill).
*   It calls **`server.NewTokenBucketRateLimiter(100, 1 * time.Second)`**.
*   **`NewTokenBucketRateLimiter()`** (`ratelimit.go`):
    *   It creates a `TokenBucketRateLimiter` struct.
    *   This struct contains a `clients` map. This map will store the token bucket for each unique client IP address.
    *   It also starts a background goroutine (`go limiter.cleanup()`) that runs periodically to remove old IP address entries from the `clients` map to prevent memory leaks.
*   **`main()`**: The returned `ratelimiter` instance is used to create the `RateLimitedHandler`, which wraps the main `dnsHandler`. The server is now ready.

---

### Scenario: A new client (`192.168.1.10`) sends its first query.

#### 2. First Request Arrives (`cmd/app/main.go` -> `server/request.go`)

*   `dig @localhost -p 5354 google.com A` is executed from client `192.168.1.10`.
*   The request travels through `serveDNS` and a new goroutine is launched for `HandleDNSRequest`.
*   **`HandleDNSRequest()`** (`request.go`): It parses the request successfully. Crucially, it creates a `context` and stores the client's IP address (`192.168.1.10`) in it. It then calls `resolveDomain`.
*   **`resolveDomain()`** (`request.go`): It calls `handler.HandleQuery(...)`. The `handler` here is our `RateLimitedHandler`.

#### 3. The Rate Limiter Intervenes (`server/handler.go`)

*   **`HandleQuery()`** (`RateLimitedHandler` in `handler.go`): This is the critical step for rate limiting.
    *   It calls `GetClientIPFromContext(ctx)` and successfully extracts `"192.168.1.10"`.
    *   It then calls **`h.limiter.AllowQuery("192.168.1.10")`**.

#### 4. The Token Bucket Logic (`server/ratelimit.go`)

*   **`AllowQuery()`** (`TokenBucketRateLimiter` in `ratelimit.go`):
    *   It locks the mutex (`limiter.mu.Lock()`) to ensure that the `clients` map is accessed safely by only one goroutine at a time.
    *   It checks if the IP `"192.168.1.10"` exists as a key in the `limiter.clients` map.
    *   **It does not exist.** This is the first time this IP has been seen.
    *   The code then creates a **new `tokenBucket` struct** for this IP. This new bucket is initialized with:
        *   `tokens`: `limiter.capacity - 1` (so, `100 - 1 = 99`). The first token is consumed immediately upon creation.
        *   `lastRefill`: The current time.
    *   This new bucket is added to the `limiter.clients` map: `limiter.clients["192.168.1.10"] = bucket`.
    *   The mutex is unlocked (`limiter.mu.Unlock()`).
    *   The function returns `true`, because the client was new and had enough capacity.

#### 5. The Request Proceeds

*   **`HandleQuery()`** (`RateLimitedHandler` in `handler.go`): Since `AllowQuery` returned `true`, the request is not blocked. It passes the call through to the wrapped `dnsHandler`.
*   From here, the request follows the exact same path as the previous dry run: it gets resolved by the `DNSResolver`, forwarded upstream, and a successful response is sent back to the client.

---

### Scenario: The same client sends 99 more queries instantly.

*   Each of the 99 queries follows the same path.
*   **`AllowQuery()`** (`ratelimit.go`): For each query, the IP `"192.168.1.10"` is now found in the `clients` map.
    *   The code calls the `refill()` method on the bucket. Since no time has passed, no new tokens are added.
    *   It checks if `bucket.tokens > 0`. It is.
    *   It decrements the token count: `bucket.tokens--`.
    *   It returns `true`.
*   After the 100th total query (the 1st + these 99), the bucket for `"192.168.1.10"` now has **`0` tokens**.

---

### Scenario: The client sends its 101st query instantly.

#### 6. The Request is Blocked

*   The request arrives at the `RateLimitedHandler`'s `HandleQuery` method as before.
*   It calls **`h.limiter.AllowQuery("192.168.1.10")`**.
*   **`AllowQuery()`** (`ratelimit.go`):
    *   It finds the bucket for `"192.168.1.10"`.
    *   It calls `refill()`. No time has passed, so no tokens are added.
    *   It checks if `bucket.tokens > 0`. **This is now false.** The bucket has 0 tokens.
    *   The function immediately returns `false`.
*   **`HandleQuery()`** (`RateLimitedHandler` in `handler.go`):
    *   The `if !h.limiter.AllowQuery(ip)` condition is now **true**.
    *   It logs the message `[RATE LIMIT] Blocked request from 192.168.1.10...`.
    *   It returns an error: `"rate limit exceeded"`.
*   **`HandleDNSRequest()`** (`request.go`): It receives this error and calls `handleError`, which sends a `SERVFAIL` response back to the `dig` client. The query never reaches the resolver.

---

### Scenario: The client waits for 2 seconds and tries again.

*   The 102nd query arrives.
*   **`AllowQuery()`** (`ratelimit.go`):
    *   It finds the bucket for `"192.168.1.10"`.
    *   It calls **`refill()`** on the bucket.
    *   Inside `refill()`, it calculates the time elapsed since `lastRefill`. It's been 2 seconds.
    *   It calculates how many new tokens to add: `(2 seconds / 1s refill rate) = 2 tokens`.
    *   It adds these to the bucket: `bucket.tokens += 2`. The bucket now has **2 tokens**.
    *   It updates `lastRefill` to the current time.
    *   Back in `AllowQuery`, it checks `bucket.tokens > 0`. This is true.
    *   It decrements the tokens to `1` and returns `true`.
*   The request is **allowed** to proceed.