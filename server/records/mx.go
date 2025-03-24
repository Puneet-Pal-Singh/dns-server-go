package records

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type MXRecord struct {
	BaseHandler
}

func (r *MXRecord) Type() uint16       { return 15 }
func (r *MXRecord) Class() uint16      { return 1 }
func (r *MXRecord) DefaultTTL() uint32 { return 300 }

type MXData struct {
	Preference uint16
	Exchange   string
}

func (r *MXRecord) ValidateData(data interface{}) error {
	mx, ok := data.(MXData)
	if !ok {
		return errors.New("invalid MX data format")
	}
	return validateDomain(mx.Exchange)
}

func (r *MXRecord) BuildRecordData(data interface{}) ([]byte, error) {
	mx, ok := data.(MXData)
	if !ok {
		return nil, errors.New("invalid MX data format")
	}

	var buf bytes.Buffer
	// Write preference (2 bytes)
	if err := binary.Write(&buf, binary.BigEndian, mx.Preference); err != nil {
		return nil, err
	}

	// // Write exchange domain name
	// exchange := strings.TrimSuffix(mx.Exchange, ".")
	// labels := strings.Split(exchange, ".")
	// for _, label := range labels {
	// 	if err := buf.WriteByte(byte(len(label))); err != nil {
	// 		return nil, err
	// 	}
	// 	if _, err := buf.WriteString(label); err != nil {
	// 		return nil, err
	// 	}
	// }
	// // Terminate with zero byte
	// if err := buf.WriteByte(0); err != nil {
	// 	return nil, err
	// }

	// Write exchange domain
    if err := r.WriteDomainName(&buf, mx.Exchange); err != nil {
        return nil, err
    }

	return buf.Bytes(), nil
}

// Update BuildAnswer method to match the interface
func (r *MXRecord) BuildAnswer(domain string, data interface{}, ttl uint32) (*bytes.Buffer, error) {
	return r.BaseHandler.BuildAnswer(r, domain, data, ttl)
}
