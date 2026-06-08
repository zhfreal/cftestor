package main

import (
	"strings"
	"testing"
)

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

func TestDeprecatedFlagsStillMapToCanonicalBehavior(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want func(t *testing.T)
	}{
		{
			name: "disable download maps to dt only",
			args: []string{"--silence", "--disable-download", "-s", "1.1.1.1"},
			want: func(t *testing.T) {
				if !Config.DTOnly {
					t.Fatal("expected --disable-download to enable DT-only mode")
				}
			},
		},
		{
			name: "dt via https alias maps protocol",
			args: []string{"--silence", "--dt-via-https", "--dt-only", "-s", "1.1.1.1"},
			want: func(t *testing.T) {
				if Config.DTVia != "https" || !Config.DTHttps {
					t.Fatalf("expected HTTPS DT mode, got DTVia=%q DTHttps=%v", Config.DTVia, Config.DTHttps)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGlobalsForTest()
			shouldExit, _, err := configureApp(tt.args)
			if shouldExit || err != nil {
				t.Fatalf("configureApp(%v) = shouldExit %v, err %v", tt.args, shouldExit, err)
			}
			tt.want(t)
		})
	}
}

func TestNormalizeDTViaLowercasesProtocol(t *testing.T) {
	resetGlobalsForTest()
	Config.DTVia = "TLS"
	if err := normalizeDTVia(); err != nil {
		t.Fatalf("normalizeDTVia returned error: %v", err)
	}
	if Config.DTVia != "tls" || Config.DTHttps {
		t.Fatalf("normalizeDTVia did not normalize TLS mode, DTVia=%q DTHttps=%v", Config.DTVia, Config.DTHttps)
	}
}

func TestValidationMessagesMatchThresholds(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "result", args: []string{"--silence", "-s", "1.1.1.1", "--result", "0"}, wantErr: "must be greater than 0"},
		{name: "port", args: []string{"--silence", "-s", "1.1.1.1", "--port", "0"}, wantErr: "invalid value for \"-p|--port\""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGlobalsForTest()
			shouldExit, _, err := configureApp(tt.args)
			if !shouldExit || err == nil {
				t.Fatalf("configureApp(%v) = shouldExit %v, err %v; want error", tt.args, shouldExit, err)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestSourceIPsRejectsBareDNSHostWithHostPortMessage(t *testing.T) {
	sources := NewSourceIPs()
	err := sources.AddFromSlice([]string{"example.com"}, TypeIPv4|TypeIPv6)
	if err == nil {
		t.Fatal("expected bare DNS host without port to be rejected")
	}
	if !strings.Contains(err.Error(), "host:port") {
		t.Fatalf("error %q does not mention host:port", err.Error())
	}
}

func TestAddPortsReturnsErrorsAndDeduplicates(t *testing.T) {
	sources := NewSourceIPs()
	if err := sources.AddPorts([]string{"443,8443", "8443"}); err != nil {
		t.Fatalf("AddPorts returned error: %v", err)
	}
	if len(sources.ports) != 2 || sources.ports[0] != 443 || sources.ports[1] != 8443 {
		t.Fatalf("ports = %#v, want [443 8443]", sources.ports)
	}

	badSources := NewSourceIPs()
	if err := badSources.AddPorts([]string{"65536"}); err == nil {
		t.Fatal("expected invalid port to return error")
	}
}

func TestURLHelpersReturnErrorsInsteadOfExiting(t *testing.T) {
	host, port, err := parseUrl("https://example.com/path")
	if err != nil {
		t.Fatalf("parseUrl returned error: %v", err)
	}
	if host != "example.com" || port != 443 {
		t.Fatalf("parseUrl returned (%q, %d), want (example.com, 443)", host, port)
	}

	got, err := newUrl("https://example.com/path", "8443")
	if err != nil {
		t.Fatalf("newUrl returned error: %v", err)
	}
	if got != "https://example.com:8443/path" {
		t.Fatalf("newUrl returned %q", got)
	}

	if _, _, err := parseUrl("not a url"); err == nil {
		t.Fatal("expected invalid parseUrl input to return error")
	}
	if _, err := newUrl("https://example.com/path", "0"); err == nil {
		t.Fatal("expected invalid newUrl port to return error")
	}
}

func TestDefaultURLsUseCloudflareSpeedtest(t *testing.T) {
	if defaultDTUrl != "https://speed.cloudflare.com/__down?bytes=0" {
		t.Fatalf("defaultDTUrl = %q", defaultDTUrl)
	}
	if defaultDLTUrl != "https://speed.cloudflare.com/__down?bytes=250000000" {
		t.Fatalf("defaultDLTUrl = %q", defaultDLTUrl)
	}
	if DefaultTestHost != "speed.cloudflare.com" {
		t.Fatalf("DefaultTestHost = %q", DefaultTestHost)
	}
}

func TestNoCacheIgnoresDefaultURLsWithEquivalentPorts(t *testing.T) {
	resetGlobalsForTest()
	Config.NoCache = true

	defaults := []string{
		defaultDTUrl,
		defaultDLTUrl,
		"https://speed.cloudflare.com:443/__down?bytes=0",
		"https://speed.cloudflare.com:443/__down?bytes=250000000",
	}
	for _, sourceURL := range defaults {
		if shouldApplyNoCache(sourceURL) {
			t.Fatalf("shouldApplyNoCache(%q) = true, want false", sourceURL)
		}
	}

	if !shouldApplyNoCache("https://example.com/__down?bytes=250000000") {
		t.Fatal("custom URL should apply no-cache when Config.NoCache is true")
	}

	Config.NoCache = false
	if shouldApplyNoCache("https://example.com/__down?bytes=250000000") {
		t.Fatal("custom URL should not apply no-cache when Config.NoCache is false")
	}
}
