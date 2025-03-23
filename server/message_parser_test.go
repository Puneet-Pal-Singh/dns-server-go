package server

import (
	"testing"
)

func TestParseDomainName_Valid(t *testing.T) {
	tests := []struct {
		input  []byte
		expect string
	}{
		{[]byte{3, 'w', 'w', 'w', 7, 'e', 'x', 'a', 'm', 'p', 'l', 'e', 3, 'c', 'o', 'm', 0}, "www.example.com"},
		{[]byte{0}, "."}, // root domain
		{[]byte{4, 't', 'e', 's', 't', 0}, "test"},
	}

	parser := NewDomainParser()
	for _, tc := range tests {
		result, _, err := parser.Parse(tc.input)
		if err != nil {
			t.Errorf("Failed to parse valid domain %v: %v", tc.input, err)
		}
		if result != tc.expect {
			t.Errorf("Expected %q, got %q", tc.expect, result)
		}
	}
}

func TestParseDomainName_Compression(t *testing.T) {
	// Test valid compression pointer (0xC000 points to start of message)
	data := []byte{
		// First label: foo (offset 0)
		3, 'f', 'o', 'o', 0,
		// Compressed label: bar + pointer to foo (0xC000 = pointer to offset 0)
		3, 'b', 'a', 'r', 0xC0, 0x00,
	}

	parser := NewDomainParser()
	result, _, err := parser.Parse(data) // Parse entire message
	if err != nil {
		t.Fatalf("Failed to parse compressed domain: %v", err)
	}
	expected := "bar.foo"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestParseDomainName_InvalidLabelLength(t *testing.T) {
	parser := NewDomainParser()
	invalid := []byte{64, 'a'}
	_, _, err := parser.Parse(invalid)
	if err == nil {
		t.Error("Should reject label length >63")
	}
}

func TestParseDomainName_Truncated(t *testing.T) {
	parser := NewDomainParser()
	truncated := []byte{3, 'w', 'w'}
	_, _, err := parser.Parse(truncated)
	if err == nil {
		t.Error("Should detect truncated labels")
	}
}
