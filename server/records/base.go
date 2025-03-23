package records

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
)

const (
	ClassIN    = 1   // Internet class
	QDCOUNT    = 1   // Questions count
	ANCOUNT    = 1   // Answers count
	TypeA      = 1   // A record type
	TypeAAAA   = 28  // AAAA record type
	TypeMX     = 15  // MX record type
	TypeTXT    = 16  // TXT record type
	DefaultTTL = 300 // Default TTL value
)

// Add unified registration
func init() {
	RegisterHandler(&ARecord{})
	RegisterHandler(&AAAARecord{})
	RegisterHandler(&TXTRecord{})
	RegisterHandler(&MXRecord{})
}

var handlers = make(map[uint16]RecordHandler)

func RegisterHandler(h RecordHandler) {
	handlers[h.Type()] = h
}

func GetHandler(qtype uint16) (RecordHandler, bool) {
	h, ok := handlers[qtype]
	return h, ok
}

// RecordHandler interface for all DNS records
type RecordHandler interface {
	ValidateData(data interface{}) error
	BuildRecordData(data interface{}) ([]byte, error)
	BuildAnswer(domain string, data interface{}, ttl uint32) (*bytes.Buffer, error)
	Type() uint16
	Class() uint16
	DefaultTTL() uint32
}

// Change this
type DomainNameWriter struct {
	Offsets map[string]int
	Pos     int
}

// To this
type BaseHandler struct {
	Writer *DomainNameWriter
}

// Add this interface definition
type DomainNameCompressor interface {
	SetWriter(*DomainNameWriter)
}

// Add the SetWriter method to BaseHandler
func (b *BaseHandler) SetWriter(w *DomainNameWriter) {
	b.Writer = w
}

// WriteDomainName with compression support
func (b *BaseHandler) WriteDomainName(buf *bytes.Buffer, domain string) error {
	if len(domain) == 0 {
		return errors.New("empty domain name")
	}

	// Initialize writer if needed
	if b.Writer == nil {
		b.Writer = &DomainNameWriter{
			Offsets: make(map[string]int),
		}
	}

	startPos := b.Writer.Pos

	// Check if we've seen this domain before
	if offset, exists := b.Writer.Offsets[domain]; exists {
		// Use compression pointer (0xC0 | offset)
		pointer := uint16(0xC000 | offset)
		return binary.Write(buf, binary.BigEndian, pointer)
	}

	labels := strings.Split(domain, ".")
	for i, label := range labels {
		if len(label) == 0 {
			return errors.New("empty label in domain name")
		}
		if len(label) > 63 {
			return errors.New("label exceeds 63 characters")
		}

		// Store offset for this subdomain
		remainingDomain := strings.Join(labels[i:], ".")
		b.Writer.Offsets[remainingDomain] = b.Writer.Pos

		if err := buf.WriteByte(byte(len(label))); err != nil {
			return fmt.Errorf("failed to write label length: %w", err)
		}
		if _, err := buf.Write([]byte(label)); err != nil {
			return fmt.Errorf("failed to write label: %w", err)
		}

		b.Writer.Pos += 1 + len(label)
	}

	// Write the terminating null byte
	if err := buf.WriteByte(0); err != nil {
		return fmt.Errorf("failed to write terminator: %w", err)
	}
	b.Writer.Pos++

	// Store the complete domain name offset
	b.Writer.Offsets[domain] = startPos

	return nil
}

// ValidateCommon checks base requirements
func (b *BaseHandler) ValidateCommon(domain string, data interface{}) error {
	if err := validateDomain(domain); err != nil {
		return err
	}
	if data == nil {
		return errors.New("nil record data")
	}
	return nil
}

// BuildCommonAnswer handles common answer structure
func (b *BaseHandler) BuildCommonAnswer(h RecordHandler, domain string, data interface{}, ttl uint32) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)

	// Write domain name using shared implementation
	if err := b.WriteDomainName(buf, domain); err != nil {
		return nil, fmt.Errorf("failed to write domain name: %w", err)
	}

	// Write record type
	if err := binary.Write(buf, binary.BigEndian, h.Type()); err != nil {
		return nil, fmt.Errorf("failed to write type: %w", err)
	}

	// Write class
	if err := binary.Write(buf, binary.BigEndian, h.Class()); err != nil {
		return nil, fmt.Errorf("failed to write class: %w", err)
	}

	// Use default TTL if not specified
	if ttl == 0 {
		ttl = h.DefaultTTL()
	}

	// Write TTL
	if err := binary.Write(buf, binary.BigEndian, ttl); err != nil {
		return nil, fmt.Errorf("failed to write TTL: %w", err)
	}

	// Build record data
	recordData, err := h.BuildRecordData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to build record data: %w", err)
	}

	// Write record data length
	if err := binary.Write(buf, binary.BigEndian, uint16(len(recordData))); err != nil {
		return nil, fmt.Errorf("failed to write data length: %w", err)
	}

	// Write record data
	if _, err := buf.Write(recordData); err != nil {
		return nil, fmt.Errorf("failed to write record data: %w", err)
	}

	return buf, nil
}

// ValidateIP checks IP address validity
func (b *BaseHandler) ValidateIP(data interface{}, ipv6 bool) error {
	ip, ok := data.(string)
	if !ok {
		return errors.New("invalid data type, expected string")
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return errors.New("invalid IP format")
	}

	if ipv6 && parsed.To4() != nil {
		return errors.New("IPv4 address not allowed in AAAA record, expected IPv6 address")
	}

	if !ipv6 && parsed.To4() == nil {
		return errors.New("IPv6 address not allowed in A record, expected IPv4 address")
	}
	return nil
}

// Common validation functions
func validateDomain(domain string) error {
	if len(domain) == 0 {
		return errors.New("empty domain name")
	}
	if len(domain) > 255 {
		return errors.New("domain name exceeds maximum length")
	}
	if strings.Contains(domain, "..") {
		return errors.New("invalid domain name format")
	}
	return nil
}

// Add common response building logic
func (b *BaseHandler) BuildAnswer(h RecordHandler, domain string, data interface{}, ttl uint32) (*bytes.Buffer, error) {
	if err := h.ValidateData(data); err != nil {
		return nil, err
	}
	return b.BuildCommonAnswer(h, domain, data, ttl)
}

// Add type validation in base handler
func (b *BaseHandler) ValidateRecordType(data interface{}, expectedType string) error {
	switch expectedType {
	case "A":
		return b.ValidateIP(data, false)
	case "AAAA":
		return b.ValidateIP(data, true)
	case "MX":
		_, ok := data.(MXData)
		if !ok {
			return errors.New("invalid MX data")
		}
		return nil
	default:
		return errors.New("unsupported record type")
	}
}
