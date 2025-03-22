// server/request.go
package server

import (
	"context"
	"encoding/binary"
	"errors"
	"log"
	"net"

	"github.com/Puneet-Pal-Singh/dns-server-go/server/records"
)

type contextKey string

const clientIPKey = contextKey("client_ip")

const (
	responseSuccess       = 0x8180
	responseServerFailure = 0x8182
)

// HandleDNSRequest orchestrates the DNS request handling process
func HandleDNSRequest(conn *net.UDPConn, clientAddr *net.UDPAddr, request []byte, handler DNSHandler) {
	ctx := context.WithValue(context.Background(), clientIPKey, clientAddr.IP.String())

	txnID, domain, qtype, err := parseRequest(request)
	if err != nil {
		handleError(conn, clientAddr, txnID, "Request parsing", err)
		return
	}

	log.Printf("[%d] Received query for: %s", txnID, domain)

	recordHandler, data, err := resolveDomain(ctx, handler, domain, qtype)
	if err != nil {
		handleError(conn, clientAddr, txnID, "Domain resolution", err)
		return
	}

	log.Printf("[%d] Resolved %s → %s", txnID, domain, data)

	if err := buildAndSendResponse(conn, clientAddr, txnID, domain, recordHandler, data); err != nil {
		handleError(conn, clientAddr, txnID, "Response building", err)
	}
}

// parseRequest extracts transaction ID, domain, and query type from the request
func parseRequest(request []byte) (uint16, string, uint16, error) {
	if len(request) < 12 {
		return 0, "", 0, errors.New("request shorter than header size")
	}

	txnID := binary.BigEndian.Uint16(request[0:2])

	// Parse QNAME starting at offset 12
	domain, bytesRead, err := parseDomainNameWithLength(request[12:])
	if err != nil {
		return 0, "", 0, err
	}

	// QTYPE starts after QNAME (domain) + null byte
	qtypeStart := 12 + bytesRead
	if len(request) < qtypeStart+4 {
		return 0, "", 0, errors.New("request too short for qtype/qclass")
	}

	qtype := binary.BigEndian.Uint16(request[qtypeStart : qtypeStart+2])
	return txnID, domain, qtype, nil
}

// Helper function that returns parsed domain and bytes consumed
func parseDomainNameWithLength(data []byte) (string, int, error) {
	domain, err := ParseDomainName(data)
	if err != nil {
		return "", 0, err
	}

	// Calculate bytes consumed by the domain name
	pos := 0
	for {
		if pos >= len(data) || data[pos] == 0 {
			return domain, pos + 1, nil // +1 for null terminator
		}
		pos += int(data[pos]) + 1
	}
}

// resolveDomain delegates to the DNS handler
func resolveDomain(ctx context.Context, handler DNSHandler, domain string, qtype uint16) (records.RecordHandler, interface{}, error) {
	// Get handler for query type
	recordHandler, ok := records.GetHandler(qtype)
	if !ok {
		return nil, nil, errors.New("unsupported query type")
	}

	data, err := handler.HandleQuery(ctx, domain, qtype)
	if err != nil {
		return nil, nil, err
	}

	if err := recordHandler.ValidateData(data); err != nil {
		return nil, nil, err
	}

	return recordHandler, data, nil
}

// buildAndSendResponse constructs and sends the DNS response
func buildAndSendResponse(conn *net.UDPConn, addr *net.UDPAddr, txnID uint16, domain string, handler records.RecordHandler, data interface{}) error {
	response, err := BuildResponse(txnID, domain, handler, data, responseSuccess, handler.DefaultTTL())
	if err != nil {
		return err
	}
	_, err = conn.WriteToUDP(response, addr)
	return err
}

// handleError centralizes error handling and response
func handleError(conn *net.UDPConn, addr *net.UDPAddr, txnID uint16, context string, err error) {
	log.Printf("%s error: %v", context, err)
	sendErrorResponse(conn, addr, txnID, responseServerFailure)
}

func sendErrorResponse(conn *net.UDPConn, addr *net.UDPAddr, txnID uint16, flags uint16) {
	header := make([]byte, 12)
	binary.BigEndian.PutUint16(header[0:2], txnID)
	binary.BigEndian.PutUint16(header[2:4], flags)
	if _, err := conn.WriteToUDP(header, addr); err != nil {
		log.Printf("Error sending failure response: %v", err)
	}
}
