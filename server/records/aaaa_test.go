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

	// Use the helper function to validate the AAAA record structure
	validateAAAARecord(t, answer, "example.com", "2001:db8::1", 300)
}

func TestAAAARecord_Comprehensive(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		ip      string
		ttl     uint32
		wantErr bool
	}{
		{"valid_ipv6", "example.com", "2001:db8::1", 300, false},
		{"invalid_ipv6", "example.com", "2001:xyz::1", 300, true},
		{"ipv4_address", "example.com", "192.168.1.1", 300, true},
		{"empty_ip", "example.com", "", 300, true},
		{"zero_ttl", "example.com", "2001:db8::1", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aaaa := &AAAARecord{}
			answer, err := aaaa.BuildAnswer(tt.domain, tt.ip, tt.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildAnswer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				validateAAAARecord(t, answer, tt.domain, tt.ip, tt.ttl)
			}
		})
	}
}

func validateAAAARecord(t *testing.T, buf *bytes.Buffer, domain, ip string, ttl uint32) {
	data := buf.Bytes()
	if len(data) < 22 { // Minimum size for AAAA record
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
	if qtype != uint16(28) { // AAAA record type
		t.Errorf("Expected type %d, got %d", 28, qtype)
	}
	if class != uint16(1) { // IN class
		t.Errorf("Expected class %d, got %d", 1, class)
	}
	if readTTL != ttl {
		t.Errorf("Expected TTL %d, got %d", ttl, readTTL)
	}
	if dataLen != 16 { // AAAA record data length should be 16 bytes
		t.Errorf("Expected data length 16, got %d", dataLen)
	}

	// Validate IP
	ipData := make([]byte, 16)
	copy(ipData, data[22:38]) // Assuming the IP starts after the header and fields
	if !bytes.Equal(ipData, net.ParseIP(ip).To16()) {
		t.Errorf("Expected IP %v, got %v", ip, ipData)
	}
}
