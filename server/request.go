// server/request.go
package server

import (
	"context"
	"encoding/binary"
	"errors"
	"log"
	"net"
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

	txnID, domain, err := parseRequest(request)
	if err != nil {
		handleError(conn, clientAddr, txnID, "Request parsing", err)
		return
	}

	log.Printf("[%d] Received query for: %s", txnID, domain)

	ip, err := resolveDomain(ctx, handler, domain)
	if err != nil {
		handleError(conn, clientAddr, txnID, "Domain resolution", err)
		return
	}

	log.Printf("[%d] Resolved %s → %s", txnID, domain, ip)

	if err := buildAndSendResponse(conn, clientAddr, txnID, domain, ip); err != nil {
		handleError(conn, clientAddr, txnID, "Response building", err)
	}
}

// parseRequest extracts transaction ID and domain from the request
func parseRequest(request []byte) (uint16, string, error) {
	if len(request) < 12 {
		return 0, "", errors.New("request too short")
	}

	txnID := binary.BigEndian.Uint16(request[0:2])
	domain, err := ParseDomainName(request[12:])
	return txnID, domain, err
}

// resolveDomain delegates to the DNS handler
func resolveDomain(ctx context.Context, handler DNSHandler, domain string) (string, error) {
	return handler.HandleQuery(ctx, domain)
}

// buildAndSendResponse constructs and sends the DNS response
func buildAndSendResponse(conn *net.UDPConn, addr *net.UDPAddr, txnID uint16, domain, ip string) error {
	response, err := BuildResponse(txnID, domain, ip, responseSuccess, 300)
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
