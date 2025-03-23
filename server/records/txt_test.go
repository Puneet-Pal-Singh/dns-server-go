package records

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestTXTRecord_Comprehensive(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		txt     string
		ttl     uint32
		wantErr bool
	}{
		{"valid_short", "example.com", "v=spf1 include:_spf.example.com ~all", 300, false},
		{"empty_string", "example.com", "", 300, false},
		{"max_length", "example.com", strings.Repeat("a", 255), 300, false},
		{"too_long", "example.com", strings.Repeat("a", 256), 300, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txt := &TXTRecord{}
			answer, err := txt.BuildAnswer(tt.domain, tt.txt, tt.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildAnswer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				validateTXTRecord(t, answer, tt.domain, tt.txt, tt.ttl)
			}
		})
	}
}

func validateTXTRecord(t *testing.T, buf *bytes.Buffer, domain string, txt string, ttl uint32) {
	data := buf.Bytes()
	if len(data) < 14 { // Minimum size for TXT record
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
	if qtype != uint16(16) { // TXT record type
		t.Errorf("Expected type %d, got %d", 16, qtype)
	}
	if class != uint16(1) { // IN class
		t.Errorf("Expected class %d, got %d", 1, class)
	}
	if readTTL != ttl {
		t.Errorf("Expected TTL %d, got %d", ttl, readTTL)
	}
	if dataLen > 255 { // TXT record data length should not exceed 255 bytes
		t.Errorf("Expected data length <= 255, got %d", dataLen)
	}

	// Validate TXT data
	txtData := string(data[22 : 22+dataLen])
	if txtData != txt {
		t.Errorf("Expected TXT data %s, got %s", txt, txtData)
	}
}
