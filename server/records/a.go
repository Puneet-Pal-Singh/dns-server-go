// server/records/a.go
package records

import (
	"bytes"
	"errors"
	"net"
)

type ARecord struct {
	BaseHandler
}

func (r *ARecord) Type() uint16       { return TypeA }
func (r *ARecord) Class() uint16      { return ClassIN }
func (r *ARecord) DefaultTTL() uint32 { return DefaultTTL }

func (r *ARecord) ValidateData(data interface{}) error {
	return r.ValidateIP(data, false)
}

func (r *ARecord) BuildRecordData(data interface{}) ([]byte, error) {
	ip := net.ParseIP(data.(string)).To4()
	if ip == nil {
		return nil, errors.New("invalid IPv4 address")
	}
	return ip, nil
}

func (r *ARecord) BuildAnswer(domain string, data interface{}, ttl uint32) (*bytes.Buffer, error) {
	if err := r.ValidateData(data); err != nil {
		return nil, err
	}

	return r.BaseHandler.BuildAnswer(r, domain, data, ttl)
}
