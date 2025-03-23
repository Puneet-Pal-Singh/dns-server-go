// server/message.go
package server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// DomainParser interface for dependency injection
type DomainParser interface {
	Parse(data []byte) (string, int, error)
}

// domainParser implements DomainParser with state encapsulation
type domainParser struct {
	data     []byte
	pos      int
	maxPos   int
	jumps    int
	parts    []string
	totalLen int
}

// NewDomainParser factory function
func NewDomainParser() DomainParser {
	return &domainParser{}
}

// Parse implements DomainParser interface
func (dp *domainParser) Parse(data []byte) (string, int, error) {
	dp.reset(data)
	domain, err := dp.parseDomain()
	return domain, dp.pos, err
}

// Private implementation methods
func (dp *domainParser) reset(data []byte) {
	dp.data = data
	dp.pos = 0
	dp.maxPos = len(data)
	dp.jumps = 0
	dp.parts = make([]string, 0)
	dp.totalLen = 0
}

func (dp *domainParser) parseDomain() (string, error) {
	for {
		if err := dp.checkBufferBounds(); err != nil {
			return "", err
		}

		if dp.isCompressionPointer() {
			return dp.handleCompression()
		}

		label, err := dp.readLabel()
		if err != nil {
			return "", err
		}
		if label == "" {
			break
		}

		dp.parts = append(dp.parts, label)
	}

	return strings.Join(dp.parts, "."), nil
}

func (dp *domainParser) checkBufferBounds() error {
	if dp.pos >= dp.maxPos {
		return fmt.Errorf("buffer underflow at position %d", dp.pos)
	}
	return nil
}

func (dp *domainParser) isCompressionPointer() bool {
	return dp.data[dp.pos]&0xC0 == 0xC0
}

func (dp *domainParser) handleCompression() (string, error) {
	if dp.pos+1 >= dp.maxPos {
		return "", errors.New("truncated compression pointer")
	}

	offset := int(binary.BigEndian.Uint16(dp.data[dp.pos:dp.pos+2]) & 0x3FFF)
	dp.pos += 2

	if err := dp.validateCompressionOffset(offset); err != nil {
		return "", err
	}

	return dp.parseCompressedDomain(offset)
}

func (dp *domainParser) validateCompressionOffset(offset int) error {
	if offset >= dp.maxPos {
		return fmt.Errorf("invalid compression offset %d", offset)
	}
	if dp.jumps > 10 {
		return errors.New("compression loop detected")
	}
	dp.jumps++
	return nil
}

func (dp *domainParser) parseCompressedDomain(offset int) (string, error) {
	subParser := NewDomainParser()
	subDomain, _, err := subParser.Parse(dp.data[offset:])
	if err != nil {
		return "", fmt.Errorf("compressed domain resolution failed: %w", err)
	}
	dp.parts = append(dp.parts, subDomain)
	return strings.Join(dp.parts, "."), nil
}

func (dp *domainParser) readLabel() (string, error) {
	labelLen := int(dp.data[dp.pos])
	dp.pos++

	if labelLen == 0 {
		return "", nil // End of domain
	}

	if err := dp.validateLabelLength(labelLen); err != nil {
		return "", err
	}

	end := dp.pos + labelLen
	if err := dp.validateLabelBounds(end); err != nil {
		return "", err
	}

	label := string(dp.data[dp.pos:end])
	dp.pos = end

	if err := dp.validateTotalLength(labelLen); err != nil {
		return "", err
	}

	return label, nil
}

func (dp *domainParser) validateLabelLength(length int) error {
	if length > 63 {
		return fmt.Errorf("invalid label length %d at position %d", length, dp.pos-1)
	}
	return nil
}

func (dp *domainParser) validateLabelBounds(end int) error {
	if end > dp.maxPos {
		return fmt.Errorf("label exceeds buffer at position %d", dp.pos)
	}
	return nil
}

func (dp *domainParser) validateTotalLength(labelLen int) error {
	dp.totalLen += labelLen + 1
	if dp.totalLen > 255 {
		return errors.New("domain exceeds 255 characters")
	}
	return nil
}

// [8, 102, 97, 99, 101, 98, 111, 111, 107, 3, 99, 111, 109, 0]
// Domain encoding: facebook.com → 8facebook3com0
// Breakdown:
// 08 (length) + "facebook" + 03 (length) + "com" + 00 (terminator)

