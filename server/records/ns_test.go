package records

import (
	"bytes"
	"testing"
)

func TestNSRecord_ValidateData(t *testing.T) {
	ns := &NSRecord{}

	t.Run("Valid NS", func(t *testing.T) {
		if err := ns.ValidateData("ns1.example.com"); err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("Invalid Type", func(t *testing.T) {
		err := ns.ValidateData(123)
		if err == nil || err.Error() != "invalid NS data type, expected string" {
			t.Errorf("Expected type error, got: %v", err)
		}
	})

	t.Run("Invalid Domain", func(t *testing.T) {
		err := ns.ValidateData(".invalid.")
		if err == nil {
			t.Error("Expected domain validation error")
		}
	})
}

func TestNSRecord_BuildRecordData(t *testing.T) {
	ns := &NSRecord{}
	expected := []byte{
		3, 'n', 's', '1',
		7, 'e', 'x', 'a', 'm', 'p', 'l', 'e',
		3, 'c', 'o', 'm',
		0,
	}

	data, err := ns.BuildRecordData("ns1.example.com")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !bytes.Equal(data, expected) {
		t.Errorf("Expected:\n%x\nGot:\n%x", expected, data)
	}
}