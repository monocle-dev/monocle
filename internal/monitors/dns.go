package monitors

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/monocle-dev/monocle/internal/types"
)

func CheckDNS(config *types.DNSConfig) error {
	timeout := config.Timeout

	if timeout == 0 {
		timeout = 5 // 5 seconds timeout by default
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	resolver := &net.Resolver{}

	switch strings.ToUpper(config.RecordType) {
	case "A":
		return checkARecord(ctx, resolver, config)
	case "AAAA":
		return checkAAAARecord(ctx, resolver, config)
	case "CNAME":
		return checkCNAMERecord(ctx, resolver, config)
	case "MX":
		return checkMXRecord(ctx, resolver, config)
	case "TXT":
		return checkTXTRecord(ctx, resolver, config)
	case "NS":
		return checkNSRecord(ctx, resolver, config)
	default:
		return errors.New("unsupported DNS record type: " + config.RecordType)
	}
}

func checkARecord(ctx context.Context, resolver *net.Resolver, config *types.DNSConfig) error {
	ips, err := resolver.LookupIPAddr(ctx, config.Domain)

	if err != nil {
		return fmt.Errorf("failed to resolve A record for %s: %v", config.Domain, err)
	}

	if len(ips) == 0 {
		return fmt.Errorf("no A records found for %s", config.Domain)
	}

	if config.Expected != "" {
		expectedIP := net.ParseIP(config.Expected)

		if expectedIP == nil {
			return fmt.Errorf("invalid expected IP: %s", config.Expected)
		}

		for _, ip := range ips {
			if ip.IP.Equal(expectedIP) {
				return nil
			}
		}

		return fmt.Errorf("expected IP %s not found in DNS response", config.Expected)
	}

	return nil
}

func checkAAAARecord(ctx context.Context, resolver *net.Resolver, config *types.DNSConfig) error {
	ips, err := resolver.LookupIPAddr(ctx, config.Domain)

	if err != nil {
		return fmt.Errorf("failed to resolve AAAA record for %s: %v", config.Domain, err)
	}

	var ipv6Found bool

	for _, ip := range ips {
		if ip.IP.To4() == nil {
			ipv6Found = true

			if config.Expected != "" {
				expectedIP := net.ParseIP(config.Expected)

				if expectedIP != nil && ip.IP.Equal(expectedIP) {
					return nil
				}
			}
		}
	}

	if !ipv6Found {
		return fmt.Errorf("no AAAA records found for %s", config.Domain)
	}

	if config.Expected != "" {
		return fmt.Errorf("expected IPv6 %s not found in DNS response", config.Expected)
	}

	return nil
}

func checkCNAMERecord(ctx context.Context, resolver *net.Resolver, config *types.DNSConfig) error {
	cname, err := resolver.LookupCNAME(ctx, config.Domain)

	if err != nil {
		return fmt.Errorf("failed to resolve CNAME for %s: %v", config.Domain, err)
	}

	if config.Expected != "" && !strings.EqualFold(cname, config.Expected) {
		return fmt.Errorf("expected CNAME %s, got %s", config.Expected, cname)
	}

	return nil
}

func checkMXRecord(ctx context.Context, resolver *net.Resolver, config *types.DNSConfig) error {
	mxRecords, err := resolver.LookupMX(ctx, config.Domain)

	if err != nil {
		return fmt.Errorf("failed to resolve MX records for %s: %v", config.Domain, err)
	}

	if len(mxRecords) == 0 {
		return fmt.Errorf("no MX records found for %s", config.Domain)
	}

	if config.Expected != "" {
		for _, mx := range mxRecords {
			if strings.EqualFold(mx.Host, config.Expected) {
				return nil
			}
		}

		return fmt.Errorf("expected MX record %s not found", config.Expected)
	}

	return nil
}

func checkTXTRecord(ctx context.Context, resolver *net.Resolver, config *types.DNSConfig) error {
	txtRecords, err := resolver.LookupTXT(ctx, config.Domain)

	if err != nil {
		return fmt.Errorf("failed to resolve TXT records for %s: %v", config.Domain, err)
	}

	if len(txtRecords) == 0 {
		return fmt.Errorf("no TXT records found for %s", config.Domain)
	}

	if config.Expected != "" {
		for _, txt := range txtRecords {
			if txt == config.Expected {
				return nil
			}
		}

		return fmt.Errorf("expected TXT record content %s not found", config.Expected)
	}

	return nil
}

func checkNSRecord(ctx context.Context, resolver *net.Resolver, config *types.DNSConfig) error {
	nsRecords, err := resolver.LookupNS(ctx, config.Domain)

	if err != nil {
		return fmt.Errorf("failed to resolve NS records for %s: %v", config.Domain, err)
	}

	if len(nsRecords) == 0 {
		return fmt.Errorf("no NS records found for %s", config.Domain)
	}

	if config.Expected != "" {
		for _, ns := range nsRecords {
			if strings.EqualFold(ns.Host, config.Expected) {
				return nil
			}
		}

		return fmt.Errorf("expected NS record %s not found", config.Expected)
	}

	return nil
}
