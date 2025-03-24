package records

import (
	"bytes"
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
	if qtype != uint16(16) {
		t.Errorf("Expected TXT type 16, got %d", qtype)
	}
	if class != uint16(1) {
		t.Errorf("Expected class %d, got %d", 1, class)
	}

	// Read TXT data
	txtLen, err := r.ReadByte()
	if err != nil {
		t.Fatalf("Failed to read TXT length: %v", err)
	}
	txtData := make([]byte, txtLen)
	if _, err := r.Read(txtData); err != nil {
		t.Fatalf("Failed to read TXT data: %v", err)
	}
	if string(txtData) != txt {
		t.Errorf("TXT mismatch: expected %q, got %q", txt, string(txtData))
	}
}

func TestTXTRecord_Integration(t *testing.T) {
	txt := &TXTRecord{}
	testData := "v=spf1 include:_spf.google.com ~all"

	answer, err := txt.BuildAnswer("example.com", testData, 300)
	if err != nil {
		t.Fatalf("Failed to build TXT answer: %v", err)
	}

	validateTXTRecord(t, answer, "example.com", testData, 300)
}

func TestTXTHandler_Registration(t *testing.T) {
	handler, ok := GetHandler(TypeTXT)
	if !ok {
		t.Fatal("TXT handler not registered")
	}

	if handler.Type() != TypeTXT {
		t.Errorf("Wrong type: got %d, want %d", handler.Type(), TypeTXT)
	}
}
