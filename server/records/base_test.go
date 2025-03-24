package records

import (
	"testing"
)

func TestHandlerRegistration(t *testing.T) {
	// Clear existing handlers
	handlers = make(map[uint16]RecordHandler)

	// Register handlers
	RegisterHandler(&ARecord{})
	RegisterHandler(&MXRecord{})
	RegisterHandler(&CNAMERecord{})
	RegisterHandler(&TXTRecord{})

	tests := []struct {
		name   string
		qtype  uint16
		wantOK bool
	}{
		{"A", TypeA, true},
		{"MX", TypeMX, true},
		{"CNAME", TypeCNAME, true},
		{"TXT", TypeTXT, true},
		{"Invalid", 999, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := GetHandler(tt.qtype)
			if ok != tt.wantOK {
				t.Errorf("GetHandler(%d) got ok = %v, want %v", tt.qtype, ok, tt.wantOK)
			}
		})
	}
}
