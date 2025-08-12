// server/response_builder.go
package server

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/Puneet-Pal-Singh/dns-server-go/server/records"
)

// DNSResponseBuilder constructs DNS responses through composition
type DNSResponseBuilder struct {
	buf      *bytes.Buffer
	header   []byte
	question []byte
	answer   []byte
	records.BaseHandler
	position int
}

// NewDNSResponseBuilder creates a new response builder
func NewDNSResponseBuilder(txnID uint16, flags uint16) *DNSResponseBuilder {
	b := &DNSResponseBuilder{
		buf:      new(bytes.Buffer),
		header:   make([]byte, 12),
		position: 12, // Initialize position after header
	}

	// Initialize header
	binary.BigEndian.PutUint16(b.header[0:2], txnID)
	binary.BigEndian.PutUint16(b.header[2:4], flags)
	binary.BigEndian.PutUint16(b.header[4:6], records.QDCOUNT)
	binary.BigEndian.PutUint16(b.header[6:8], records.ANCOUNT)

	return b
}

// WithQuestion adds the question section
func (b *DNSResponseBuilder) WithQuestion(domain string, qtype uint16) error {
	var qBuf bytes.Buffer

	b.Writer = &records.DomainNameWriter{
		Offsets: make(map[string]int),
		Pos:     12,
	}

	if err := b.WriteDomainName(&qBuf, domain); err != nil {
		return err
	}

	// Fix: Use binary.Write for both QTYPE and QCLASS
	if err := binary.Write(&qBuf, binary.BigEndian, qtype); err != nil {
		return err
	}
	if err := binary.Write(&qBuf, binary.BigEndian, uint16(records.ClassIN)); err != nil {
		return err
	}

	b.question = qBuf.Bytes()
	b.Writer.Pos += len(b.question)
	return nil
}

// WithAnswer adds the answer section
func (b *DNSResponseBuilder) WithAnswer(domain string, handler records.RecordHandler, data interface{}, ttl uint32) error {
	// Share compression state with any handler implementing DomainNameCompressor
	if compressor, ok := handler.(records.DomainNameCompressor); ok {
		compressor.SetWriter(b.Writer)
	}

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
	b.position += len(b.header)

	b.buf.Write(b.question)
	b.position += len(b.question)

	b.buf.Write(b.answer)
	b.position += len(b.answer)

	return b.buf.Bytes()
}

// BuildResponse provides a simplified interface for response construction
func BuildResponse(txnID uint16, domain string, handler records.RecordHandler, data interface{}, flags uint16, ttl uint32) ([]byte, error) {
	// Validate inputs
	if domain == "" {
		return nil, errors.New("empty domain name")
	}
	if handler == nil {
		return nil, errors.New("nil record handler")
	}

	builder := NewDNSResponseBuilder(txnID, flags)

	// Add question section
	if err := builder.WithQuestion(domain, handler.Type()); err != nil {
		return nil, fmt.Errorf("failed to add question: %w", err)
	}

	// Add answer section
	if err := builder.WithAnswer(domain, handler, data, ttl); err != nil {
		return nil, fmt.Errorf("failed to add answer: %w", err)
	}

	return builder.Build(), nil
}
