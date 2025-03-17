package records

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"strings"
)

const (
	TypeA    = 1
	TypeAAAA = 28
	TypeTXT  = 16
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

// BaseHandler implements common functionality
type BaseHandler struct{}

// WriteDomainName common implementation
func (b *BaseHandler) WriteDomainName(buf *bytes.Buffer, domain string) error {
	if strings.HasSuffix(domain, ".") {
		domain = domain[:len(domain)-1]
	}

	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if len(label) > 63 {
			return errors.New("label exceeds 63 characters")
		}
		buf.WriteByte(byte(len(label)))
		buf.WriteString(label)
	}
	buf.WriteByte(0) // Null terminator
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
	if ttl == 0 {
		ttl = h.DefaultTTL()
	}

	var buf bytes.Buffer
	if err := b.WriteDomainName(&buf, domain); err != nil {
		return nil, err
	}

	binary.Write(&buf, binary.BigEndian, h.Type())
	binary.Write(&buf, binary.BigEndian, h.Class())
	binary.Write(&buf, binary.BigEndian, ttl)

	recordData, err := h.BuildRecordData(data)
	if err != nil {
		return nil, err
	}

	binary.Write(&buf, binary.BigEndian, uint16(len(recordData)))
	buf.Write(recordData)

	return &buf, nil
}

// ValidateIP checks IP address validity
func (b *BaseHandler) ValidateIP(data interface{}, ipv6 bool) error {
	ip, ok := data.(string)
	if !ok {
		return errors.New("invalid data type")
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return errors.New("invalid IP format")
	}

	if ipv6 && parsed.To4() != nil {
		return errors.New("expected IPv6 address")
	}

	if !ipv6 && parsed.To4() == nil {
		return errors.New("expected IPv4 address")
	}
	return nil
}

// Common validation functions
func validateDomain(domain string) error {
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
