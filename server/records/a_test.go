package records

import (
	"bytes"
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

// Update domain name parser to handle compression pointers properly
func parseDomainName(r *bytes.Reader) (string, error) {
	var labels []string
	originalPos, _ := r.Seek(0, 1) // Track original position for compression pointers

	for {
		length, err := r.ReadByte()
		if err != nil {
			return "", err
		}

		// Handle compression pointer (two high bits set)
		if (length & 0xC0) == 0xC0 {
			nextByte, _ := r.ReadByte()
			pointer := uint16(length&^0xC0)<<8 | uint16(nextByte)

			// Save current position and jump to pointer
			currentPos, _ := r.Seek(0, 1)
			r.Seek(int64(pointer), 0)
			name, _ := parseDomainName(r) // Recursively parse compressed name
			r.Seek(currentPos, 0)         // Restore original position
			return strings.Join(labels, ".") + "." + name, nil
		}

		if length == 0 {
			break
		}
		label := make([]byte, length)
		r.Read(label)
		labels = append(labels, string(label))
	}

	// Reset reader position if no compression was used
	r.Seek(originalPos, 0)
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
	if len(data) < 14 {
		t.Error("Response too short")
		return
	}

	r := bytes.NewReader(data)
	_, err := parseDomainName(r)
	if err != nil {
		t.Fatalf("Failed to parse domain name: %v", err)
	}

	// Read DNS fields
	qtype, class, readTTL, dataLen, err := readDNSFields(r)
	if err != nil {
		t.Fatalf("Failed to read DNS fields: %v", err)
	}

	// Validate fields
	if qtype != uint16(1) {
		t.Errorf("Expected type %d, got %d", 1, qtype)
	}
	if class != uint16(1) {
		t.Errorf("Expected class %d, got %d", 1, class)
	}
	if readTTL != ttl {
		t.Errorf("Expected TTL %d, got %d", ttl, readTTL)
	}
	if dataLen != 4 {
		t.Errorf("Expected data length 4, got %d", dataLen)
	}

	// Read IP address
	ipBytes := make([]byte, 4)
	if _, err := r.Read(ipBytes); err != nil {
		t.Fatalf("Failed to read IP: %v", err)
	}
	parsedIP := net.IP(ipBytes).String()
	if parsedIP != ip {
		t.Errorf("IP mismatch: expected %s, got %s", ip, parsedIP)
	}
}
