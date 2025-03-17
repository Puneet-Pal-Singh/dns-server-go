package records

import (
	"errors"
)

type TXTRecord struct {
	BaseHandler
}

func (r *TXTRecord) Type() uint16       { return 16 }
func (r *TXTRecord) Class() uint16      { return 1 }
func (r *TXTRecord) DefaultTTL() uint32 { return 300 }

func (r *TXTRecord) ValidateData(data interface{}) error {
	txt, ok := data.(string)
	if !ok {
		return errors.New("invalid data type for TXT record")
	}
	if len(txt) > 255 {
		return errors.New("TXT record exceeds 255 characters")
	}
	return nil
}

func (r *TXTRecord) BuildRecordData(data interface{}) ([]byte, error) {
	txt := data.(string)
	if len(txt) > 255 {
		return nil, errors.New("TXT record too long")
	}
	return append([]byte{byte(len(txt))}, txt...), nil
}
