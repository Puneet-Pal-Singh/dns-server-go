// server/message.go
package server

import (
	"bytes"
	"errors"
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

		// Check for pointer (compressed name)
		if (length & 0xC0) == 0xC0 {
			if offset+1 >= len(question) {
				return "", errors.New("invalid compression pointer")
			}
			// Skip compressed name for this implementation
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
// Domain encoding: facebook.com → 8facebook3com0
// Breakdown:
// 08 (length) + "facebook" + 03 (length) + "com" + 00 (terminator)

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
