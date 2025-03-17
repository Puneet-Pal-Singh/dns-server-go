// server/message.go
package server

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"

	"github.com/Puneet-Pal-Singh/dns-server-go/server/records"
)

// DNSResponseBuilder constructs DNS responses through composition
type DNSResponseBuilder struct {
	buf      *bytes.Buffer
	header   []byte
	question []byte
	answer   []byte
}

// NewDNSResponseBuilder creates a new response builder
func NewDNSResponseBuilder(txnID uint16, flags uint16) *DNSResponseBuilder {
	b := &DNSResponseBuilder{
		buf:    new(bytes.Buffer),
		header: make([]byte, 12),
	}

	// Initialize header
	binary.BigEndian.PutUint16(b.header[0:2], txnID)
	binary.BigEndian.PutUint16(b.header[2:4], flags)
	binary.BigEndian.PutUint16(b.header[4:6], 1) // QDCOUNT
	binary.BigEndian.PutUint16(b.header[6:8], 1) // ANCOUNT

	return b
}

// WithQuestion adds the question section
func (b *DNSResponseBuilder) WithQuestion(domain string) error {
	var qBuf bytes.Buffer
	if err := WriteDomainName(&qBuf, domain); err != nil {
		return err
	}
	qBuf.Write([]byte{0x00, 0x01, 0x00, 0x01}) // QTYPE and QCLASS
	b.question = qBuf.Bytes()
	return nil
}

// WithAnswer adds the answer section
func (b *DNSResponseBuilder) WithAnswer(domain string, handler records.RecordHandler, data interface{}, ttl uint32) error {
	answerBuf, err := handler.BuildAnswer(domain, data, ttl)
	if err != nil {
		return err
	}
	b.answer = answerBuf.Bytes()
	return nil
}

// Build constructs the final DNS response
func (b *DNSResponseBuilder) Build() []byte {
	b.buf.Write(b.header)
	b.buf.Write(b.question)
	b.buf.Write(b.answer)
	return b.buf.Bytes()
}

// BuildResponse (simplified interface)
func BuildResponse(txnID uint16, domain string, handler records.RecordHandler, data interface{}, flags uint16, ttl uint32) ([]byte, error) {
	builder := NewDNSResponseBuilder(txnID, flags)

	if err := builder.WithQuestion(domain); err != nil {
		return nil, err
	}

	if err := builder.WithAnswer(domain, handler, data, ttl); err != nil {
		return nil, err
	}

	return builder.Build(), nil
}

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
