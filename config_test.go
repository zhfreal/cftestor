package main

import "testing"

func resetGlobalsForTest() {
	Config = DefaultConfig()
	ipStr = []string{}
	resetRuntimeState()
	myLogger = myLogger.newLogger(logLevelFatal)
}

func TestParseCLIAcceptsDNSHostInput(t *testing.T) {
	opts, err := parseCLI([]string{"--dt-only", "-s", "example.com:443", "-6"})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if len(opts.IPs) != 1 || opts.IPs[0] != "example.com:443" {
		t.Fatalf("unexpected IP inputs: %#v", opts.IPs)
	}
	if !opts.Config.DTOnly {
		t.Fatal("expected --dt-only to be parsed")
	}
	if !opts.Config.IPv6Mode || !opts.IPv6Changed {
		t.Fatal("expected -6 to enable IPv6 mode and mark the flag changed")
	}
}

func TestSplitHostAcceptsDNSAndIPHosts(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantOK   bool
		wantHost string
		wantPort int
	}{
		{name: "dns", input: "example.com:443", wantOK: true, wantHost: "example.com", wantPort: 443},
		{name: "dns trailing dot", input: "example.com.:443", wantOK: true, wantHost: "example.com.", wantPort: 443},
		{name: "ipv4", input: "1.1.1.1:443", wantOK: true, wantHost: "1.1.1.1", wantPort: 443},
		{name: "ipv6", input: "[2606:4700::1]:443", wantOK: true, wantHost: "2606:4700::1", wantPort: 443},
		{name: "bad dns", input: "bad_host!:443", wantOK: false},
		{name: "bad ipv4 literal", input: "999.999.999.999:443", wantOK: false},
		{name: "missing port", input: "example.com", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, host, port := splitHost(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("splitHost(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if host != tt.wantHost || port != tt.wantPort {
				t.Fatalf("splitHost(%q) = (%q, %d), want (%q, %d)", tt.input, host, port, tt.wantHost, tt.wantPort)
			}
		})
	}
}

func TestSourceIPsDNSHostPassesFamilyFilters(t *testing.T) {
	for _, mode := range []int8{TypeIPv4, TypeIPv6, TypeIPv4 | TypeIPv6} {
		sources := NewSourceIPs()
		if err := sources.AddFromSlice([]string{"example.com:443"}, mode); err != nil {
			t.Fatalf("AddFromSlice returned error for mode %d: %v", mode, err)
		}
		if got := sources.LenInt(); got != 1 {
			t.Fatalf("LenInt for mode %d = %d, want 1", mode, got)
		}
	}
}

func TestSourceIPsIPLiteralHostsStillRespectFamilyFilters(t *testing.T) {
	sources := NewSourceIPs()
	if err := sources.AddFromSlice([]string{"[2606:4700::1]:443"}, TypeIPv4); err != nil {
		t.Fatalf("AddFromSlice returned error: %v", err)
	}
	if got := sources.LenInt(); got != 0 {
		t.Fatalf("IPv6 host passed IPv4-only filter, LenInt = %d", got)
	}
}

func TestLoadSourceIPsAcceptsExplicitIPv6WithoutFamilyFlag(t *testing.T) {
	resetGlobalsForTest()
	ipStr = []string{"2606:4700::1"}
	if err := loadSourceIPs(TypeIPv4, false, false); err != nil {
		t.Fatalf("loadSourceIPs returned error: %v", err)
	}
	if !Config.IPv6Mode {
		t.Fatal("expected explicit input without family flags to enable IPv6 mode")
	}
	if got := srcIPs.LenInt(); got != 1 {
		t.Fatalf("srcIPs.LenInt() = %d, want 1", got)
	}
}
