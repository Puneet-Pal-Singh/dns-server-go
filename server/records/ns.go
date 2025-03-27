package records

import (
	"bytes"
	"errors"
	"fmt"
)

// NSRecord handles name server records
type NSRecord struct {
	BaseHandler
}

func (n *NSRecord) Type() uint16       { return TypeNS }
func (n *NSRecord) Class() uint16      { return ClassIN }
func (n *NSRecord) DefaultTTL() uint32 { return DefaultTTL }

func (n *NSRecord) ValidateData(data interface{}) error {
	ns, ok := data.(string)
	if !ok {
		return errors.New("invalid NS data type, expected string")
	}
	return validateDomain(ns)
}

func (n *NSRecord) BuildRecordData(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := n.WriteDomainName(&buf, data.(string)); err != nil {
		return nil, fmt.Errorf("failed to write NS record: %w", err)
	}
	return buf.Bytes(), nil
}

func (n *NSRecord) BuildAnswer(domain string, data interface{}, ttl uint32) (*bytes.Buffer, error) {
	return n.BuildCommonAnswer(n, domain, data, ttl)
}
