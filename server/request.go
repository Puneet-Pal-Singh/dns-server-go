// server/request.go
package server

import (
	"encoding/binary"
	"log"
	"net"
)

// HandleDNSRequest processes incoming DNS requests
func HandleDNSRequest(conn *net.UDPConn, clientAddr *net.UDPAddr, request []byte, resolver *DNSResolver) {
	if len(request) < 12 {
		log.Println("Invalid DNS request: too short")
		return
	}

	// Extract the transaction ID (bytes 0-1 of the request)
	txnID := binary.BigEndian.Uint16(request[0:2])
	log.Printf("Transaction ID: %d", txnID)

	// Extract domain name from the request
	domain, err := ParseDomainName(request[12:])
	if err != nil {
		log.Printf("Error parsing domain name: %v", err)
		return
	}

	log.Printf("Query for domain: %s", domain)

	// Resolve the domain to an IP
	ip, err := resolver.ResolveDomain(domain)
	if err != nil {
		log.Printf("Error resolving domain: %v", err)
		return
	}

	log.Printf("Resolved IP for domain %s: %s", domain, ip)

	// Build the DNS response
	response, err := BuildResponse(txnID, domain, ip)
	if err != nil {
		log.Printf("Error building response: %v", err)
		return
	}

	// Send the response
	if _, err := conn.WriteToUDP(response, clientAddr); err != nil {
		log.Printf("Error sending response: %v", err)
	}
}
