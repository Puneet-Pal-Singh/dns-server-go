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
	return 28 // AAAA type code
}

func (r *AAAARecord) Class() uint16 {
	return 1
}

func (r *AAAARecord) DefaultTTL() uint32 {
	return 300
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
	return r.BaseHandler.BuildAnswer(r, domain, data, ttl)
}
