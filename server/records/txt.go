package records

import (
	"bytes"
	"errors"
)

type TXTRecord struct {
	BaseHandler
}

func (r *TXTRecord) Type() uint16       { return 16 }
func (r *TXTRecord) Class() uint16      { return 1 }
func (r *TXTRecord) DefaultTTL() uint32 { return 300 }

func (r *TXTRecord) ValidateData(data interface{}) error {
	switch v := data.(type) {
	case string:
		if len(v) > 255 {
			return errors.New("TXT record exceeds 255 characters")
		}
		return nil
	case []string:
		if len(v) == 0 {
			return errors.New("empty TXT record")
		}
		for _, txt := range v {
			if len(txt) > 255 {
				return errors.New("TXT record exceeds 255 characters")
			}
		}
		return nil
	default:
		return errors.New("invalid data type for TXT record")
	}
}

func (r *TXTRecord) BuildRecordData(data interface{}) ([]byte, error) {
	var texts []string
	switch v := data.(type) {
	case string:
		texts = []string{v}
	case []string:
		texts = v
	default:
		return nil, errors.New("invalid TXT record data type")
	}

	var buf bytes.Buffer
	for _, txt := range texts {
		if len(txt) > 255 {
			return nil, errors.New("TXT record too long")
		}
		// Write length byte
		if err := buf.WriteByte(byte(len(txt))); err != nil {
			return nil, err
		}
		// Write the string data
		if _, err := buf.WriteString(txt); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// Update BuildAnswer method to match the interface
func (r *TXTRecord) BuildAnswer(domain string, data interface{}, ttl uint32) (*bytes.Buffer, error) {
	return r.BaseHandler.BuildAnswer(r, domain, data, ttl)
}
