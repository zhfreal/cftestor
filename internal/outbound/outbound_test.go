package outbound

import (
	"bytes"
	"io"
	"testing"
)

func TestParseOutboundMark(t *testing.T) {
	tests := []struct {
		input    string
		expected uint32
		wantErr  bool
	}{
		{"0", 0, false},
		{"123", 123, false},
		{"0x7b", 123, false},
		{"0x100", 256, false},
		{"4294967295", 4294967295, false},
		{"0xffffffff", 4294967295, false},
		{"", 0, true},
		{"-1", 0, true},
		{"4294967296", 0, true},
		{"abc", 0, true},
	}

	for _, tt := range tests {
		got, err := ParseOutboundMark("--mark", tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseOutboundMark(%q) err = %v, wantErr = %v", tt.input, err, tt.wantErr)
		}
		if err == nil && got != tt.expected {
			t.Errorf("ParseOutboundMark(%q) = %d, expected %d", tt.input, got, tt.expected)
		}
	}
}

func TestParseOutboundSourceIP(t *testing.T) {
	tests := []struct {
		raw          string
		expectedIP   string
		expectedZone string
		ok           bool
	}{
		{"192.168.1.100", "192.168.1.100", "", true},
		{"fe80::1", "fe80::1", "", true},
		{"fe80::1%eth0", "fe80::1", "eth0", true},
		{"eth0", "", "", false},
		{"", "", "", false},
	}

	for _, tt := range tests {
		ip, zone, ok := parseOutboundSourceIP(tt.raw)
		if ok != tt.ok {
			t.Errorf("parseOutboundSourceIP(%q) ok = %v, want %v", tt.raw, ok, tt.ok)
		}
		if ok {
			if ip.String() != tt.expectedIP {
				t.Errorf("parseOutboundSourceIP(%q) ip = %v, want %v", tt.raw, ip, tt.expectedIP)
			}
			if zone != tt.expectedZone {
				t.Errorf("parseOutboundSourceIP(%q) zone = %v, want %v", tt.raw, zone, tt.expectedZone)
			}
		}
	}
}

func TestParseOutboundInterfaceIndex(t *testing.T) {
	tests := []struct {
		raw      string
		expected int
		ok       bool
		wantErr  bool
	}{
		{"1", 1, true, false},
		{"42", 42, true, false},
		{"0", 0, false, true},
		{"-1", 0, false, false},
		{"eth0", 0, false, false},
	}

	for _, tt := range tests {
		index, ok, err := parseOutboundInterfaceIndex(tt.raw)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseOutboundInterfaceIndex(%q) err = %v, wantErr = %v", tt.raw, err, tt.wantErr)
		}
		if ok != tt.ok {
			t.Errorf("parseOutboundInterfaceIndex(%q) ok = %v, want %v", tt.raw, ok, tt.ok)
		}
		if ok && err == nil && index != tt.expected {
			t.Errorf("parseOutboundInterfaceIndex(%q) index = %d, want %d", tt.raw, index, tt.expected)
		}
	}
}

func TestGetLocFromCFResp(t *testing.T) {
	sampleTrace := `fl=123f45
h=speed.cloudflare.com
ip=1.1.1.1
ts=1721111111.111
visit_scheme=https
uag=Mozilla/5.0
colo=SFO
sliver=none
http=http/2
loc=US
tls=TLSv1.3
sni=plaintext
warp=off
gateway=off
rbi=off
kex=X25519
`

	reader := io.NopCloser(bytes.NewBufferString(sampleTrace))
	loc, err := get_loc_from_cf_resp(reader)
	if err != nil {
		t.Fatalf("get_loc_from_cf_resp failed: %v", err)
	}

	if loc == "" {
		t.Fatalf("expected non-empty location from trace")
	}
}
