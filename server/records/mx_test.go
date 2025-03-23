package records

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestMXRecord_Comprehensive(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		mx      MXData
		ttl     uint32
		wantErr bool
	}{
		{
			"valid_mx",
			"example.com",
			MXData{Preference: 10, Exchange: "mail.example.com"},
			300,
			false,
		},
		{
			"empty_exchange",
			"example.com",
			MXData{Preference: 10, Exchange: ""},
			300,
			true,
		},
		{
			"invalid_exchange",
			"example.com",
			MXData{Preference: 10, Exchange: "invalid..domain"},
			300,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mx := &MXRecord{}
			answer, err := mx.BuildAnswer(tt.domain, tt.mx, tt.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildAnswer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				validateMXRecord(t, answer, tt.domain, tt.mx, tt.ttl)
			}
		})
	}
}

func validateMXRecord(t *testing.T, buf *bytes.Buffer, domain string, mx MXData, ttl uint32) {
	data := buf.Bytes()
	if len(data) < 14 { // Minimum size for MX record
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
	if qtype != uint16(15) { // MX record type
		t.Errorf("Expected type %d, got %d", 15, qtype)
	}
	if class != uint16(1) { // IN class
		t.Errorf("Expected class %d, got %d", 1, class)
	}
	if readTTL != ttl {
		t.Errorf("Expected TTL %d, got %d", ttl, readTTL)
	}
	if dataLen < 3 { // MX record data length should be at least 3 bytes (preference + exchange)
		t.Errorf("Expected data length at least 3, got %d", dataLen)
	}

	// Validate preference
	preference := binary.BigEndian.Uint16(data[22:24])
	if preference != mx.Preference {
		t.Errorf("Expected preference %d, got %d", mx.Preference, preference)
	}

	// Validate exchange
	exchange, _ := parseDomainName(bytes.NewReader(data[24:]))
	if exchange != mx.Exchange {
		t.Errorf("Expected exchange %s, got %s", mx.Exchange, exchange)
	}
}
