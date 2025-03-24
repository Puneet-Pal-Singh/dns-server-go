package records

import (
	"bytes"
	"testing"
)

func TestCNAMERecord_Comprehensive(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		target  string
		ttl     uint32
		wantErr bool
	}{
		{"valid", "www.example.com", "example.com", 300, false},
		{"invalid_target", "www.example.com", "invalid..domain", 300, true},
		{"empty_target", "www.example.com", "", 300, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cname := &CNAMERecord{}
			answer, err := cname.BuildAnswer(tt.domain, tt.target, tt.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildAnswer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				validateCNAMERecord(t, answer, tt.domain, tt.target, tt.ttl)
			}
		})
	}
}

func validateCNAMERecord(t *testing.T, buf *bytes.Buffer, domain, target string, ttl uint32) {
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
	if qtype != uint16(5) {
		t.Errorf("Expected CNAME type 5, got %d", qtype)
	}
	if class != uint16(1) {
		t.Errorf("Expected class %d, got %d", 1, class)
	}

	// Read target
	parsedTarget, err := parseDomainName(r)
	if err != nil {
		t.Fatalf("Failed to parse target domain: %v", err)
	}
	if parsedTarget != target {
		t.Errorf("Target mismatch: expected %q, got %q", target, parsedTarget)
	}
}
