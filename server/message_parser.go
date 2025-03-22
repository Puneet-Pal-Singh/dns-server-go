// server/message.go
package server

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
)

// ParseDomainName parses the domain name from a DNS query message
func ParseDomainName(data []byte) (string, error) {
	var parts []string
	pos := 0
	maxPos := len(data)
	jumps := 0
	totalLen := 0

	for {
		if pos >= maxPos {
			return "", errors.New("buffer underflow")
		}

		// Handle compression pointers
		if data[pos]&0xC0 == 0xC0 {
			if pos+1 >= maxPos {
				return "", errors.New("truncated compression pointer")
			}
			offset := int(binary.BigEndian.Uint16(data[pos:pos+2]) & 0x3FFF)
			if offset >= maxPos {
				return "", errors.New("invalid compression pointer")
			}
			pos += 2
			jumps++
			if jumps > 10 {
				return "", errors.New("compression loop detected")
			}
			// Recursively parse the domain name at the offset
			subDomain, err := ParseDomainName(data[offset:])
			if err != nil {
				return "", err
			}
			parts = append(parts, subDomain)
			break // Exit after resolving the pointer
		}

		labelLen := int(data[pos])
		pos++

		if labelLen == 0 {
			break // End of domain name
		}

		if labelLen > 63 {
			return "", errors.New("invalid label length")
		}

		end := pos + labelLen
		if end > maxPos {
			return "", errors.New("label exceeds buffer")
		}

		totalLen += labelLen + 1
		if totalLen > 255 {
			return "", errors.New("domain exceeds 255 characters")
		}

		// Debugging output
		label := string(data[pos:end])

		parts = append(parts, label)
		pos = end
	}

	return strings.Join(parts, "."), nil
}

// [8, 102, 97, 99, 101, 98, 111, 111, 107, 3, 99, 111, 109, 0]
// Domain encoding: facebook.com → 8facebook3com0
// Breakdown:
// 08 (length) + "facebook" + 03 (length) + "com" + 00 (terminator)

// WriteDomainName encodes a domain name into the DNS message format
func WriteDomainName(buf *bytes.Buffer, domain string) error {
	if len(domain) == 0 {
		return errors.New("empty domain name")
	}

	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if len(label) < 1 {
			return errors.New("empty label in domain")
		}
		if len(label) > 63 {
			return errors.New("domain label exceeds 63 characters")
		}
		buf.WriteByte(byte(len(label)))
		buf.WriteString(label)
	}
	buf.WriteByte(0) // Null terminator
	return nil
}
