// server/message.go
package server

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"strings"
)

// ParseDomainName parses the domain name from a DNS query message
func ParseDomainName(question []byte) (string, error) {
	var domainParts []string
	offset := 0

	for {
		if offset >= len(question) {
			return "", errors.New("malformed question section")
		}

		length := int(question[offset])
		if length == 0 {
			break
		}

		offset++
		if offset+length > len(question) {
			return "", errors.New("label length exceeds question size")
		}

		domainParts = append(domainParts, string(question[offset:offset+length]))
		offset += length
	}

	return strings.Join(domainParts, "."), nil
}

// [8, 102, 97, 99, 101, 98, 111, 111, 107, 3, 99, 111, 109, 0]
// www.facebook.com

// BuildResponse constructs a binary DNS response
func BuildResponse(txnID uint16, domain string, ip string) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write Header
	header := make([]byte, 12)
	binary.BigEndian.PutUint16(header[0:2], txnID)  // Transaction ID
	binary.BigEndian.PutUint16(header[2:4], 0x8180) // Flags (standard response, no error)
	binary.BigEndian.PutUint16(header[4:6], 1)      // Question Count
	binary.BigEndian.PutUint16(header[6:8], 1)      // Answer Count
	binary.BigEndian.PutUint16(header[8:10], 0)     // Authority Count
	binary.BigEndian.PutUint16(header[10:12], 0)    // Additional Count
	buf.Write(header)

	// Write Question Section
	if err := WriteDomainName(buf, domain); err != nil {
		return nil, err
	}
	buf.Write([]byte{0, 1, 0, 1}) // Type A, Class IN

	// Write Answer Section
	if err := WriteDomainName(buf, domain); err != nil {
		return nil, err
	}
	buf.Write([]byte{0, 1, 0, 1})  // Type A, Class IN
	buf.Write([]byte{0, 0, 1, 44}) // TTL (300 seconds)
	buf.Write([]byte{0, 4})        // Data length (4 bytes for IPv4)
	ipBytes := net.ParseIP(ip).To4()
	if ipBytes == nil {
		return nil, errors.New("invalid IPv4 address")
	}
	buf.Write(ipBytes)

	return buf.Bytes(), nil
}

// WriteDomainName encodes a domain name into the DNS message format
func WriteDomainName(buf *bytes.Buffer, domain string) error {
	for _, part := range strings.Split(domain, ".") {
		if len(part) > 63 {
			return errors.New("domain label exceeds 63 characters")
		}
		buf.WriteByte(byte(len(part)))
		buf.WriteString(part)
	}
	buf.WriteByte(0) // Null terminator
	return nil
}
