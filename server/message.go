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
func BuildResponse(txnID uint16, domain, ip string, flags uint16, ttl uint32) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Header Section (12 bytes)
	header := make([]byte, 12)
	binary.BigEndian.PutUint16(header[0:2], txnID) // Transaction ID
	binary.BigEndian.PutUint16(header[2:4], flags) // Flags
	binary.BigEndian.PutUint16(header[4:6], 1)     // QDCOUNT = 1
	binary.BigEndian.PutUint16(header[6:8], 1)     // ANCOUNT = 1
	buf.Write(header)

	// Question Section
	if err := WriteDomainName(buf, domain); err != nil {
		return nil, err
	}
	binary.Write(buf, binary.BigEndian, uint16(1)) // QTYPE (A record)
	binary.Write(buf, binary.BigEndian, uint16(1)) // QCLASS (IN)

	// Answer Section
	if err := WriteDomainName(buf, domain); err != nil {
		return nil, err
	}
	binary.Write(buf, binary.BigEndian, uint16(1)) // TYPE (A record)
	binary.Write(buf, binary.BigEndian, uint16(1)) // CLASS (IN)
	binary.Write(buf, binary.BigEndian, ttl)       // TTL
	binary.Write(buf, binary.BigEndian, uint16(4)) // RDLENGTH (4 bytes for IPv4)

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
