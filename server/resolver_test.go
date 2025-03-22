package server

import (
	"context"
	"net"
	"testing"

	"github.com/Puneet-Pal-Singh/dns-server-go/server/records"
)

func TestDNSResolver_A_Record(t *testing.T) {
	resolver := NewDNSResolver("8.8.8.8:53")
	ctx := context.Background()

	result, err := resolver.Resolve(ctx, ResolutionContext{
		Domain: "example.com",
		QType:  records.TypeA,
	})

	if err != nil {
		t.Fatalf("Resolution failed: %v", err)
	}

	if net.ParseIP(result.(string)).To4() == nil {
		t.Error("Didn't get valid IPv4 address")
	}
}

func TestDNSResolver_AAAA_Record(t *testing.T) {
	resolver := NewDNSResolver("8.8.8.8:53")
	ctx := context.Background()

	result, err := resolver.Resolve(ctx, ResolutionContext{
		Domain: "example.com",
		QType:  records.TypeAAAA,
	})

	if err != nil {
		t.Fatalf("Resolution failed: %v", err)
	}

	if net.ParseIP(result.(string)).To16() == nil {
		t.Error("Didn't get valid IPv6 address")
	}
}

func TestDNSResolver_MX_Record(t *testing.T) {
	resolver := NewDNSResolver("8.8.8.8:53")
	ctx := context.Background()

	result, err := resolver.Resolve(ctx, ResolutionContext{
		Domain: "example.com",
		QType:  records.TypeMX,
	})

	if err != nil {
		t.Fatalf("Resolution failed: %v", err)
	}

	mxData, ok := result.(records.MXData)
	if !ok {
		t.Error("Didn't get valid MX data")
	}

	if mxData.Exchange == "" {
		t.Error("MX exchange is empty")
	}
}

func TestDNSResolver_TXTRecord(t *testing.T) {
	resolver := NewDNSResolver("8.8.8.8:53")
	ctx := context.Background()

	result, err := resolver.Resolve(ctx, ResolutionContext{
		Domain: "example.com",
		QType:  records.TypeTXT,
	})

	if err != nil {
		t.Fatalf("Resolution failed: %v", err)
	}

	if len(result.([]string)) == 0 {
		t.Error("Didn't get any TXT records")
	}
}

func TestDNSResolver_InvalidRecordType(t *testing.T) {
	resolver := NewDNSResolver("8.8.8.8:53")
	ctx := context.Background()

	_, err := resolver.Resolve(ctx, ResolutionContext{
		Domain: "example.com",
		QType:  999, // Invalid record type
	})

	if err == nil {
		t.Error("Expected error for invalid record type")
	}
}

func TestDNSResolver_InvalidDomain(t *testing.T) {
	resolver := NewDNSResolver("8.8.8.8:53")
	ctx := context.Background()

	_, err := resolver.Resolve(ctx, ResolutionContext{
		Domain: "invalid-domain",
	})

	if err == nil {
		t.Error("Expected error for invalid domain")
	}

}

func TestDNSResolver_Timeout(t *testing.T) {
	resolver := NewDNSResolver("8.8.8.8:53")
	ctx := context.Background()

	_, err := resolver.Resolve(ctx, ResolutionContext{
		Domain: "example.com",
		QType:  records.TypeA,
	})

	if err == nil {
		t.Error("Expected error for timeout")
	}
}
