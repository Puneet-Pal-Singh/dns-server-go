package records

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
)

func TestAAAARecord_ValidIPv6(t *testing.T) {
	aaaa := &AAAARecord{}
	err := aaaa.ValidateData("2001:db8::1")
	if err != nil {
		t.Errorf("Valid IPv6 failed: %v", err)
	}
}

func TestAAAARecord_InvalidIPv6(t *testing.T) {
	aaaa := &AAAARecord{}
	err := aaaa.ValidateData("192.168.1.1")
	if err == nil {
		t.Error("Invalid IPv6 validation passed")
	}
}

func TestAAAARecord_BuildAnswer(t *testing.T) {
	aaaa := &AAAARecord{}
	answer, err := aaaa.BuildAnswer("example.com", "2001:db8::1", 300)
	if err != nil {
		t.Fatalf("BuildAnswer failed: %v", err)
	}

	// Validation logic similar to A record test...
	// (Omitted for brevity, should check IPv6-specific details)

	// Validate answer structure
	expectedType := uint16(28) // AAAA record type
	expectedClass := uint16(1)
	expectedTTL := uint32(300)
	expectedIP := net.ParseIP("2001:db8::1").To16()
	expectedDataLen := 16 // IPv6 is 16 bytes

	// Read the answer bytes
	buf := answer.Bytes()
	reader := bytes.NewReader(buf)

	// Skip domain name (variable length)
	_, _ = parseDomainName(reader)

	// Read type, class, TTL, and data length
	var (
		qtype, class uint16
		ttl          uint32
		dataLen      uint16
	)
	binary.Read(reader, binary.BigEndian, &qtype)
	binary.Read(reader, binary.BigEndian, &class)
	binary.Read(reader, binary.BigEndian, &ttl)
	binary.Read(reader, binary.BigEndian, &dataLen)

	// Validate fields
	if qtype != expectedType {
		t.Errorf("Expected type %d, got %d", expectedType, qtype)
	}
	if class != expectedClass {
		t.Errorf("Expected class %d, got %d", expectedClass, class)
	}
	if ttl != expectedTTL {
		t.Errorf("Expected TTL %d, got %d", expectedTTL, ttl)
	}
	if dataLen != 16 {
		t.Errorf("Expected data length %d, got %d", expectedDataLen, dataLen)
	}

	// Validate IP
	ip := make([]byte, 16)
	reader.Read(ip)
	if !bytes.Equal(ip, expectedIP) {
		t.Errorf("Expected IP %v, got %v", expectedIP, ip)
	}
}
