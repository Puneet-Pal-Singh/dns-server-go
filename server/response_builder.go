package server

import (
	"bytes"
	"encoding/binary"

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

// BuildResponse provides a simplified interface for response construction
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
