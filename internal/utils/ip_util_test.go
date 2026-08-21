package utils

import (
	"math/rand"
	"testing"
)

func TestIPRangeFromCIDR(t *testing.T) {
	cidr := "192.168.1.0/24"
	ipr := NewIPRangeFromCIDR(&cidr)
	if ipr == nil {
		t.Fatalf("NewIPRangeFromCIDR failed for %s", cidr)
	}

	if ipr.Length().Int64() != 256 {
		t.Errorf("expected length 256 for /24, got %v", ipr.Length())
	}
	if !ipr.IsV4() {
		t.Errorf("expected IPRange to be IPv4")
	}

	// Extract 10 IPs
	extracted := ipr.Extract(10)
	if len(extracted) != 10 {
		t.Fatalf("expected 10 extracted IPs, got %d", len(extracted))
	}
	if extracted[0].String() != "192.168.1.0" || extracted[9].String() != "192.168.1.9" {
		t.Errorf("unexpected extracted IPs: %v", extracted)
	}

	// Remaining length should be 246
	if ipr.Length().Int64() != 246 {
		t.Errorf("expected remaining length 246, got %v", ipr.Length())
	}

	// Extract all remaining
	rest := ipr.ExtractAll(300)
	if len(rest) != 246 {
		t.Errorf("expected 246 rest IPs, got %d", len(rest))
	}
	if ipr.Length().Int64() != 0 {
		t.Errorf("expected 0 remaining length after ExtractAll, got %v", ipr.Length())
	}
}

func TestIPRangeIPv6RandomExtraction(t *testing.T) {
	cidr := "2001:db8::/120" // 256 IPs
	ipr := NewIPRangeFromCIDR(&cidr)
	if ipr == nil {
		t.Fatalf("NewIPRangeFromCIDR failed for %s", cidr)
	}

	if !ipr.IsV6() {
		t.Errorf("expected IPRange to be IPv6")
	}
	if ipr.Length().Int64() != 256 {
		t.Errorf("expected length 256 for /120, got %v", ipr.Length())
	}

	rng := rand.New(rand.NewSource(42))
	randomIPs := ipr.GetRandomX(rng, 5)
	if len(randomIPs) != 5 {
		t.Fatalf("expected 5 random IPs, got %d", len(randomIPs))
	}
	for _, ip := range randomIPs {
		if ip.To4() != nil {
			t.Errorf("expected IPv6 address, got %v", ip)
		}
	}
}

func TestIPValidationAndVersionDetection(t *testing.T) {
	tests := []struct {
		input     string
		validCIDR bool
		validIP   bool
		validHost bool
		v4        bool
		v6        bool
	}{
		{"1.1.1.1", false, true, false, true, false},
		{"1.1.1.1/24", true, false, false, true, false},
		{"2606:4700::1", false, true, false, false, true},
		{"2606:4700::/32", true, false, false, false, true},
		{"1.1.1.1:443", false, false, true, true, false},
		{"[2606:4700::1]:443", false, false, true, false, true},
		{"cloudflare.com:443", false, false, true, true, true}, // DNS host matches both
		{"invalid.ip", false, false, false, false, false},
	}

	for _, tt := range tests {
		if IsValidCIDR(tt.input) != tt.validCIDR {
			t.Errorf("IsValidCIDR(%q) = %v, expected %v", tt.input, IsValidCIDR(tt.input), tt.validCIDR)
		}
		if IsValidIP(tt.input) != tt.validIP {
			t.Errorf("IsValidIP(%q) = %v, expected %v", tt.input, IsValidIP(tt.input), tt.validIP)
		}
		if IsValidHost(tt.input) != tt.validHost {
			t.Errorf("IsValidHost(%q) = %v, expected %v", tt.input, IsValidHost(tt.input), tt.validHost)
		}
	}
}

func TestGenHostFromIPStrPort(t *testing.T) {
	if got := GenHostFromIPStrPort("1.1.1.1", 443); got != "1.1.1.1:443" {
		t.Errorf("expected 1.1.1.1:443, got %s", got)
	}
	if got := GenHostFromIPStrPort("2606:4700::1", 443); got != "[2606:4700::1]:443" {
		t.Errorf("expected [2606:4700::1]:443, got %s", got)
	}
	if got := GenHostFromIPStrPort("invalid-ip", 443); got != "" {
		t.Errorf("expected empty string for invalid IP, got %s", got)
	}
	if got := GenHostFromIPStrPort("1.1.1.1", 0); got != "" {
		t.Errorf("expected empty string for invalid port 0, got %s", got)
	}
}

func TestUniqueIntSlice(t *testing.T) {
	in := []int{443, 80, 443, 8443, 80, 2053}
	out := UniqueIntSlice(in)
	if len(out) != 4 {
		t.Fatalf("expected 4 unique ports, got %d (%v)", len(out), out)
	}
}

func TestMathHelpers(t *testing.T) {
	vals := []float64{10.0, 20.0, 30.0}
	if m := Mean(vals); m != 20.0 {
		t.Errorf("expected mean 20.0, got %v", m)
	}
	if v := Variance(vals); v != 100.0 {
		t.Errorf("expected sample variance 100.0, got %v", v)
	}
	if s := Std(vals); s != 10.0 {
		t.Errorf("expected std 10.0, got %v", s)
	}
	if max := MaxInt(5, 12, 3, 9); max != 12 {
		t.Errorf("expected max 12, got %d", max)
	}
	if min := MinInt(5, 12, 3, 9); min != 3 {
		t.Errorf("expected min 3, got %d", min)
	}
}

func TestIataMap(t *testing.T) {
	if country, ok := IataMap["SFO"]; !ok || country != "US" {
		t.Errorf("expected SFO -> US in IataMap, got %s (ok=%v)", country, ok)
	}
	if country, ok := IataMap["HKG"]; !ok || country != "HK" {
		t.Errorf("expected HKG -> HK in IataMap, got %s (ok=%v)", country, ok)
	}
	if country, ok := IataMap["NRT"]; !ok || country != "JP" {
		t.Errorf("expected NRT -> JP in IataMap, got %s (ok=%v)", country, ok)
	}
}

func TestUrlHelpers(t *testing.T) {
	host, port, err := ParseUrl("https://speed.cloudflare.com/100mb", "https://speed.cloudflare.com/__down?bytes=100000000")
	if err != nil {
		t.Fatalf("ParseUrl failed: %v", err)
	}
	if host != "speed.cloudflare.com" || port != 443 {
		t.Errorf("ParseUrl got host=%s, port=%d", host, port)
	}

	newUrl, err := NewUrl("https://speed.cloudflare.com/__down?bytes=100000000", "8443", "https://speed.cloudflare.com")
	if err != nil {
		t.Fatalf("NewUrl failed: %v", err)
	}
	if newUrl != "https://speed.cloudflare.com:8443/__down?bytes=100000000" {
		t.Errorf("NewUrl unexpected URL: %s", newUrl)
	}
}
