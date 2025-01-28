// cmd/app/main.go
package main

import (
	"log"
	"net"

	"github.com/Puneet-Pal-Singh/dns-server-go/server"
)

func main() {
	// DNS servers operate on port 53
	addr := ":5354"

	// Resolve UDP address
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %v", err)
	}

	// Listen on UDP
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("Failed to listen on UDP: %v", err)
	}
	defer conn.Close()

	log.Printf("DNS server started on %s\n", addr)

	// Buffer to read incoming requests
	buf := make([]byte, 512)

	resolver := server.NewDNSResolver()

	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Error reading from UDP: %v", err)
			continue
		}

		// Delegate request handling to a function
		go server.HandleDNSRequest(conn, clientAddr, buf[:n], resolver)
	}
}
