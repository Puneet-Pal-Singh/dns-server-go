// cmd/app/main.go
package main

import (
	"log"
	"net"
	"os"
	"time"
	"strconv"

	"github.com/Puneet-Pal-Singh/dns-server-go/server"
)

func main() {
	addr := ":5354"
	upstreamDNS := getUpstreamDNS()

	conn := setupUDP(addr)
	defer conn.Close()

	log.Printf("DNS server started on %s", addr)

	resolver := server.NewDNSResolver(upstreamDNS)
	baseHandler := server.NewDNSHandler(resolver)

	// Initialize rate limiting
	ratelimiter := createRateLimiter()

	// Wrap handler with rate limiting
	rateLimitedHandler := server.NewRateLimitedHandler(baseHandler, ratelimiter)

	serveDNS(conn, rateLimitedHandler)
}

// getUpstreamDNS handles environment configuration
func getUpstreamDNS() string {
	if up := os.Getenv("UPSTREAM_DNS"); up != "" {
		return up
	}
	return "8.8.8.8:53" // Default to Google DNS
}

// setupUDP handles network configuration
func setupUDP(addr string) *net.UDPConn {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Fatalf("Resolve error: %v", err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("Listen error: %v", err)
	}
	return conn
}

// serveDNS handles the request loop
func serveDNS(conn *net.UDPConn, handler server.DNSHandler) {
	buf := make([]byte, 512)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Read error: %v", err)
			continue
		}
		go server.HandleDNSRequest(conn, clientAddr, buf[:n], handler)
	}
}

func createRateLimiter() server.RateLimiter {
	capacity := getIntEnv("RATE_LIMIT_CAPACITY", 100)
	refillSec := getIntEnv("RATE_LIMIT_REFILL", 1)

	return server.NewTokenBucketRateLimiter(
		capacity,
		time.Duration(refillSec)*time.Second,
	)
}

func getIntEnv(name string, defaultValue int) int {
	if value := os.Getenv(name); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
