// cmd/app/main.go
package main

import (
	"log"
	"net"
	"os"

	"github.com/Puneet-Pal-Singh/dns-server-go/server"
)

func main() {
	addr := ":5354"
	upstreamDNS := getUpstreamDNS()

	conn := setupUDP(addr)
	defer conn.Close()

	log.Printf("DNS server started on %s", addr)

	resolver := server.NewDNSResolver(upstreamDNS)
	handler := server.NewDNSHandler(resolver)

	serveDNS(conn, handler)
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
