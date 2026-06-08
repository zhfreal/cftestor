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

func TestParseCLIAcceptsLongFormAliases(t *testing.T) {
	opts, err := parseCLI([]string{
		"--source", "example.com:443",
		"--source", "1.1.1.1:443",
		"--source-file", "sources.txt",
		"--result-count", "7",
		"--dt-workers", "3",
		"--dt-timeout-ms", "1234",
		"--dt-attempts", "2",
		"--dt-protocol", "tls",
		"--sni-hostname", "speed.cloudflare.com",
		"--dt-status-code", "204",
		"--dt-evaluate",
		"--dt-max-delay", "111",
		"--dt-min-pass-rate", "90",
		"--dt-max-stddev", "12.5",
		"--dlt-workers", "4",
		"--dlt-duration", "8",
		"--dlt-attempts", "2",
		"--dlt-timeout-ms", "3000",
		"--test-interval-ms", "10",
		"--min-speed", "123.5",
		"--to-csv",
		"--csv-file", "out.csv",
		"--to-sqlite",
		"--sqlite-file", "out.db",
		"--record-label", "edge",
		"--resolve-location",
		"--quiet",
	})
	if err != nil {
		t.Fatalf("parseCLI returned error: %v", err)
	}
	if len(opts.IPs) != 2 || opts.IPs[0] != "example.com:443" || opts.IPs[1] != "1.1.1.1:443" {
		t.Fatalf("alias sources were not merged into IPs: %#v", opts.IPs)
	}
	cfg := opts.Config
	checks := []struct {
		name string
		ok   bool
	}{
		{name: "source file", ok: cfg.IPFile == "sources.txt"},
		{name: "result count", ok: cfg.ResultMin == 7},
		{name: "dt workers", ok: cfg.DTWorkerThread == 3},
		{name: "dt timeout", ok: cfg.DTTimeout == 1234},
		{name: "dt attempts", ok: cfg.DTCount == 2},
		{name: "dt protocol", ok: cfg.DTVia == "tls"},
		{name: "sni hostname", ok: cfg.HostName == "speed.cloudflare.com"},
		{name: "dt status", ok: cfg.DTHttpRspReturnCodeExpected == 204},
		{name: "dt evaluate", ok: cfg.EnableDTEvaluation},
		{name: "dt max delay", ok: cfg.DTEvaluationDelay == 111},
		{name: "dt min pass rate", ok: cfg.DTEvaluationDTPR == 90},
		{name: "dt max stddev", ok: cfg.DTStdExp == 12.5},
		{name: "dlt workers", ok: cfg.DLTWorkerThread == 4},
		{name: "dlt duration", ok: cfg.DLTDurMax == 8},
		{name: "dlt attempts", ok: cfg.DLTCount == 2},
		{name: "dlt timeout", ok: cfg.DLTTimeout == 3000},
		{name: "interval", ok: cfg.Interval == 10},
		{name: "min speed", ok: cfg.DLTEvaluationSpeed == 123.5},
		{name: "to csv", ok: cfg.StoreToFile},
		{name: "csv file", ok: cfg.ResultFile == "out.csv"},
		{name: "to sqlite", ok: cfg.StoreToDB},
		{name: "sqlite file", ok: cfg.DBFile == "out.db"},
		{name: "record label", ok: cfg.SuffixLabel == "edge"},
		{name: "resolve location", ok: cfg.ResolveLoc},
		{name: "quiet", ok: cfg.SilenceMode},
	}
	for _, check := range checks {
		if !check.ok {
			t.Fatalf("alias check failed: %s; config = %#v", check.name, cfg)
		}
	}
	if !opts.DTTimeoutChanged {
		t.Fatal("--dt-timeout-ms should mark DTTimeoutChanged")
	}
}

func TestConfigureAppKeepsDTTimeoutAliasValue(t *testing.T) {
	resetGlobalsForTest()
	shouldExit, _, err := configureApp([]string{"--quiet", "--dt-only", "--source", "1.1.1.1", "--dt-timeout-ms", "1234"})
	if shouldExit || err != nil {
		t.Fatalf("configureApp returned shouldExit %v, err %v", shouldExit, err)
	}
	if Config.DTTimeout != 1234 {
		t.Fatalf("Config.DTTimeout = %d, want 1234", Config.DTTimeout)
	}
}
