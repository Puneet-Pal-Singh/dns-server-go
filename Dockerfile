# Stage 1: Build the application
FROM golang:1.22-alpine AS builder

# Install essential build tools
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files first for better cache utilization
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application with security flags
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /go/bin/dns-server ./cmd/app

# Stage 2: Create the final minimal image
FROM gcr.io/distroless/static-debian12

# Copy the binary from builder
COPY --from=builder /go/bin/dns-server /dns-server

# Use non-root user for security
USER nonroot:nonroot

# Document the port
EXPOSE 5354/udp

# Default environment variables
ENV UPSTREAM_DNS=8.8.8.8:53 \
    RATE_LIMIT_CAPACITY=100 \
    RATE_LIMIT_REFILL=1

# Run the binary
ENTRYPOINT ["/dns-server"]