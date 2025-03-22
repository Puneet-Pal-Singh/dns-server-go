package server

import (
	"testing"
)

func TestParseDomainName_InvalidLabelLength(t *testing.T) {
	// Label length 64 (invalid)
	invalid := []byte{64, 'a'}
	_, err := ParseDomainName(invalid)
	if err == nil {
		t.Error("Should reject label length >63")
	}
}

func TestParseDomainName_Truncated(t *testing.T) {
	// Truncated data
	truncated := []byte{3, 'w', 'w'}
	_, err := ParseDomainName(truncated)
	if err == nil {
		t.Error("Should detect truncated labels")
	}
}
