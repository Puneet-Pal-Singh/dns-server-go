// server/request.go
package server

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
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

	// Parse QNAME starting at offset 12 with byte length tracking
	parser := NewDomainParser()
	domain, bytesConsumed, err := parser.Parse(request[12:])
	if err != nil {
		return 0, "", 0, err
	}

	// Calculate QTYPE position using actual bytes consumed
	qtypeStart := 12 + bytesConsumed
	if len(request) < qtypeStart+4 {
		return 0, "", 0, errors.New("request too short for qtype/qclass")
	}

	qtype := binary.BigEndian.Uint16(request[qtypeStart : qtypeStart+2])
	return txnID, domain, qtype, nil
}

// resolveDomain delegates to the DNS handler
func resolveDomain(ctx context.Context, handler DNSHandler, domain string, qtype uint16) (records.RecordHandler, interface{}, error) {
	// Add debug logging
	log.Printf("Resolving domain %s with query type %d", domain, qtype)

	// Get handler for query type first
	recordHandler, ok := records.GetHandler(qtype)
	if !ok {
		log.Printf("No handler found for query type %d", qtype)
		return nil, nil, fmt.Errorf("unsupported query type: %d", qtype)
	}

	// Get the data from the DNS handler
	data, err := handler.HandleQuery(ctx, domain, qtype)
	if err != nil {
		log.Printf("HandleQuery error for %s (type %d): %v", domain, qtype, err)
		return nil, nil, err
	}

	// Validate the data
	if err := recordHandler.ValidateData(data); err != nil {
		log.Printf("Data validation error for %s (type %d): %v", domain, qtype, err)
		return nil, nil, fmt.Errorf("invalid data for type %d: %v", qtype, err)
	}

	// Improved Debug logging format 
	switch data := data.(type) {
	case records.MXData:
		log.Printf("[%d] Resolved %s → MX {preference: %d, exchange: %s}",
			qtype, domain, data.Preference, data.Exchange)
	case []string:
		log.Printf("[%d] Resolved %s → TXT %q", qtype, domain, data[0])
	default:
		log.Printf("[%d] Resolved %s → %v", qtype, domain, data)
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
