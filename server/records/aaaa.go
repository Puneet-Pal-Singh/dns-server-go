// server/records/aaaa.go
package records

import (
	"bytes"
	"errors"
	"net"
)

type AAAARecord struct {
	BaseHandler
}

func (r *AAAARecord) Type() uint16 {
	return TypeAAAA
}

func (r *AAAARecord) Class() uint16 {
	return ClassIN
}

func (r *AAAARecord) DefaultTTL() uint32 {
	return DefaultTTL
}

func (r *AAAARecord) ValidateData(data interface{}) error {
	return r.ValidateIP(data, true)
}

func (r *AAAARecord) BuildRecordData(data interface{}) ([]byte, error) {
	ip := net.ParseIP(data.(string)).To16()
	if ip == nil {
		return nil, errors.New("invalid IPv6 address")
	}
	return ip, nil
}

func (r *AAAARecord) BuildAnswer(domain string, data interface{}, ttl uint32) (*bytes.Buffer, error) {
	// Validate the data first
	if err := r.ValidateData(data); err != nil {
		return nil, err
	}

	// Pass the original data to BaseHandler, not the processed bytes
	return r.BaseHandler.BuildAnswer(r, domain, data, ttl)
}
