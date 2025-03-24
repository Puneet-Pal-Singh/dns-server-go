package records

import (
	"bytes"
	"errors"
)

type CNAMERecord struct {
	BaseHandler
}

func (r *CNAMERecord) Type() uint16       { return 5 }
func (r *CNAMERecord) Class() uint16      { return 1 }
func (r *CNAMERecord) DefaultTTL() uint32 { return 300 }

func (r *CNAMERecord) ValidateData(data interface{}) error {
	target, ok := data.(string)
	if !ok {
		return errors.New("invalid data type for CNAME record")
	}
	return validateDomain(target)
}

func (r *CNAMERecord) BuildRecordData(data interface{}) ([]byte, error) {
	target, ok := data.(string)
	if !ok {
		return nil, errors.New("invalid CNAME data type")
	}

	var buf bytes.Buffer
	// Remove trailing dot if present
	// target = strings.TrimSuffix(target, ".")

	// // Write domain name in DNS wire format
	// labels := strings.Split(target, ".")
	// for _, label := range labels {
	// 	if len(label) > 63 {
	// 		return nil, errors.New("label exceeds 63 characters")
	// 	}
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

	if err := r.WriteDomainName(&buf, target); err != nil {
        return nil, err
    }

	return buf.Bytes(), nil
}

func (r *CNAMERecord) BuildAnswer(domain string, data interface{}, ttl uint32) (*bytes.Buffer, error) {
	return r.BaseHandler.BuildAnswer(r, domain, data, ttl)
}
