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
	qtype, class, _, _, err := readDNSFields(r)
	if err != nil {
		t.Fatalf("Failed to read DNS fields: %v", err)
	}

	// Validate fields
	if qtype != uint16(15) {
		t.Errorf("Expected MX type 15, got %d", qtype)
	}
	if class != uint16(1) {
		t.Errorf("Expected class %d, got %d", 1, class)
	}

	// Read preference
	var preference uint16
	if err := binary.Read(r, binary.BigEndian, &preference); err != nil {
		t.Fatalf("Failed to read preference: %v", err)
	}
	if preference != mx.Preference {
		t.Errorf("Expected preference %d, got %d", mx.Preference, preference)
	}

	// Read exchange
	exchange, err := parseDomainName(r)
	if err != nil {
		t.Fatalf("Failed to read exchange: %v", err)
	}
	if exchange != mx.Exchange {
		t.Errorf("Expected exchange %q, got %q", mx.Exchange, exchange)
	}
}

func TestMXRecord_Integration(t *testing.T) {
	mx := &MXRecord{}
	data := MXData{
		Preference: 10,
		Exchange:   "mail.example.com",
	}

	answer, err := mx.BuildAnswer("example.com", data, 300)
	if err != nil {
		t.Fatalf("Failed to build MX answer: %v", err)
	}

	validateMXRecord(t, answer, "example.com", data, 300)
}

func TestMXHandler_Registration(t *testing.T) {
	handler, ok := GetHandler(TypeMX)
	if !ok {
		t.Fatal("MX handler not registered")
	}

	if handler.Type() != TypeMX {
		t.Errorf("Wrong type: got %d, want %d", handler.Type(), TypeMX)
	}
}
