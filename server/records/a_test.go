package records

import (
	"bytes"
	"encoding/binary"
	"net"
	"strings"
	"testing"
)

func TestARecord_Essential(t *testing.T) {
	t.Run("ValidIPv4", TestARecord_ValidIPv4)
	t.Run("InvalidIPv4", TestARecord_InvalidIPv4)
	t.Run("AnswerStructure", testAnswerStructure)
}

func TestARecord_ValidIPv4(t *testing.T) {
	a := &ARecord{}
	err := a.ValidateData("192.168.1.1")
	if err != nil {
		t.Errorf("Valid IPv4 failed: %v", err)
	}
}

func TestARecord_InvalidIPv4(t *testing.T) {
	a := &ARecord{}
	cases := []interface{}{"invalid", "2001:db8::1", 12345}
	for _, c := range cases {
		if err := a.ValidateData(c); err == nil {
			t.Errorf("Expected error for: %v", c)
		}
	}
}

func testAnswerStructure(t *testing.T) {
	a := &ARecord{}
	answer, err := a.BuildAnswer("test.com", "192.168.1.1", 300)
	if err != nil {
		t.Fatalf("Failed to build answer: %v", err)
	}

	data := answer.Bytes() // Extract []byte from *bytes.Buffer

	// Quick validation of critical components
	if bytes.Index(data, []byte{0x00, 0x01}) == -1 { // A record type
		t.Error("Missing A record type in answer")
	}
	if bytes.Index(data, net.ParseIP("192.168.1.1").To4()) == -1 {
		t.Error("Missing correct IPv4 in answer")
	}
}

func TestARecord_BuildAnswer(t *testing.T) {
	a := &ARecord{}
	answer, err := a.BuildAnswer("example.com", "192.168.1.1", 300)
	if err != nil {
		t.Fatalf("BuildAnswer failed: %v", err)
	}

	// Use the helper function to validate the A record structure
	validateARecord(t, answer, "example.com", "192.168.1.1", 300)
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

func TestARecord_Comprehensive(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		ip      string
		ttl     uint32
		wantErr bool
	}{
		{"valid_ipv4", "example.com", "192.168.1.1", 300, false},
		{"invalid_ipv4", "example.com", "256.256.256.256", 300, true},
		{"ipv6_address", "example.com", "2001:db8::1", 300, true},
		{"empty_ip", "example.com", "", 300, true},
		{"zero_ttl", "example.com", "192.168.1.1", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &ARecord{}
			answer, err := a.BuildAnswer(tt.domain, tt.ip, tt.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildAnswer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				validateARecord(t, answer, tt.domain, tt.ip, tt.ttl)
			}
		})
	}
}

func validateARecord(t *testing.T, buf *bytes.Buffer, domain, ip string, ttl uint32) {
	data := buf.Bytes()
	if len(data) < 14 { // Minimum size for A record
		t.Error("Response too short")
		return
	}

	// Skip domain name (variable length)
	_, _ = parseDomainName(bytes.NewReader(data[12:]))

	// Read type, class, TTL, and data length
	var (
		qtype, class uint16
		readTTL      uint32
		dataLen      uint16
	)
	binary.Read(bytes.NewReader(data[12:]), binary.BigEndian, &qtype)
	binary.Read(bytes.NewReader(data[14:]), binary.BigEndian, &class)
	binary.Read(bytes.NewReader(data[16:]), binary.BigEndian, &readTTL)
	binary.Read(bytes.NewReader(data[20:]), binary.BigEndian, &dataLen)

	// Validate fields
	if qtype != uint16(1) { // A record type
		t.Errorf("Expected type %d, got %d", 1, qtype)
	}
	if class != uint16(1) { // IN class
		t.Errorf("Expected class %d, got %d", 1, class)
	}
	if readTTL != ttl {
		t.Errorf("Expected TTL %d, got %d", ttl, readTTL)
	}
	if dataLen != 4 { // A record data length should be 4 bytes
		t.Errorf("Expected data length 4, got %d", dataLen)
	}

	// Validate IP
	ipData := make([]byte, 4)
	copy(ipData, data[24:28]) // Assuming the IP starts after the header and fields
	if !bytes.Equal(ipData, net.ParseIP(ip).To4()) {
		t.Errorf("Expected IP %v, got %v", ip, ipData)
	}
}
