package records

import (
	"bytes"
	"encoding/binary"
	"net"
	"strings"
	"testing"
)

func TestARecord_ValidIPv4(t *testing.T) {
	a := &ARecord{}
	err := a.ValidateData("192.168.1.1")
	if err != nil {
		t.Errorf("Valid IPv4 failed: %v", err)
	}
}

func TestARecord_InvalidIPv4(t *testing.T) {
	a := &ARecord{}
	err := a.ValidateData("2001:db8::1")
	if err == nil {
		t.Error("Invalid IPv4 validation passed")
	}
}

func TestARecord_BuildAnswer(t *testing.T) {
	a := &ARecord{}
	answer, err := a.BuildAnswer("example.com", "192.168.1.1", 300)
	if err != nil {
		t.Fatalf("BuildAnswer failed: %v", err)
	}

	// Validate answer structure
	expectedType := uint16(1) // A record type
	expectedClass := uint16(1)
	expectedTTL := uint32(300)
	expectedIP := net.ParseIP("192.168.1.1").To4()

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
	if dataLen != 4 {
		t.Errorf("Expected data length 4, got %d", dataLen)
	}

	// Validate IP
	ip := make([]byte, 4)
	reader.Read(ip)
	if !bytes.Equal(ip, expectedIP) {
		t.Errorf("Expected IP %v, got %v", expectedIP, ip)
	}
}

// Helper to parse domain name (simplified)
func parseDomainName(r *bytes.Reader) (string, error) {
	var labels []string
	for {
		length, _ := r.ReadByte()
		if length == 0 {
			break
		}
		label := make([]byte, length)
		r.Read(label)
		labels = append(labels, string(label))
	}
	return strings.Join(labels, "."), nil
}

func TestARecord_BuildAnswer_InvalidData(t *testing.T) {
	a := &ARecord{}
	_, err := a.BuildAnswer("example.com", "invalid-data", 3600)
	if err == nil {
		t.Error("BuildAnswer should have failed with invalid data")
	}
}

// Add decompression test
func TestParseDomainName_Compression(t *testing.T) {
	// Full message with compression pointer and target labels
	// www.example.com → 3www6example3com0
	compressed := []byte{
		3, 'w', 'w', 'w', 0xC0, 0x06, // Pointer to offset 6
		6, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
		3, 'c', 'o', 'm', 0,
	}

	r := bytes.NewReader(compressed)
	result, err := parseDomainName(r)
	if err != nil {
		t.Fatalf("Failed to parse compressed name: %v", err)
	}
	if result != "www.example.com" {
		t.Errorf("Expected www.example.com, got %s", result)
	}
}
