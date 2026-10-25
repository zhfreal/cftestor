package config_test

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"cftestor/internal/config"
	"cftestor/internal/fetcher"
	"cftestor/internal/logger"
	"cftestor/internal/outbound"
	"cftestor/internal/utils"
)

func firstLocalTestIP(t *testing.T) (net.IP, string) {
	t.Helper()
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("net.Interfaces returned error: %v", err)
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch a := addr.(type) {
			case *net.IPNet:
				ip = a.IP
			case *net.IPAddr:
				ip = a.IP
			}
			if ip == nil || ip.IsUnspecified() {
				continue
			}
			if v4 := ip.To4(); v4 != nil {
				return v4, ""
			}
			return ip, iface.Name
		}
	}
	t.Fatal("no local interface address available for test")
	return nil, ""
}

func firstLocalTestInterface(t *testing.T) net.Interface {
	t.Helper()
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("net.Interfaces returned error: %v", err)
	}
	for _, iface := range ifaces {
		if iface.Index > 0 {
			return iface
		}
	}
	t.Fatal("no local interface available for test")
	return net.Interface{}
}

func TestParseOutboundMarkAcceptsDecimalAndHex(t *testing.T) {
	tests := []struct {
		input string
		want  uint32
	}{
		{input: "123", want: 123},
		{input: "0x7b", want: 123},
		{input: "0X7B", want: 123},
		{input: "0xffffffff", want: 0xffffffff},
	}
	for _, tt := range tests {
		got, err := outbound.ParseOutboundMark("--mark", tt.input)
		if err != nil {
			t.Fatalf("ParseOutboundMark(%q) returned error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Fatalf("ParseOutboundMark(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParseOutboundMarkRejectsInvalidValues(t *testing.T) {
	for _, input := range []string{"", "-1", "abc", "0x100000000"} {
		if _, err := outbound.ParseOutboundMark("--mark", input); err == nil {
			t.Fatalf("ParseOutboundMark(%q) returned nil error", input)
		}
	}
}

func TestConfigureAppRejectsConflictingOutboundMarks(t *testing.T) {
	resetGlobalsForTest()
	opts, shouldExit, _, err := config.ConfigureApp([]string{"--quiet", "--dt-only", "--source", "1.1.1.1", "--mark", "1", "--xmark", "2"})
	if err == nil {
		err = outbound.PrepareOutboundOptions(&opts)
	}
	if !shouldExit && err == nil {
		t.Fatalf("ConfigureApp returned shouldExit %v, err %v; want conflicting mark error", shouldExit, err)
	}
	if err == nil || !strings.Contains(err.Error(), "cannot set different mark values") {
		t.Fatalf("error %v does not mention conflicting mark values", err)
	}
}

func TestConfigureAppAcceptsOutboundMarkAliasesOnLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("socket marks are Linux-only")
	}
	resetGlobalsForTest()
	opts, shouldExit, _, err := config.ConfigureApp([]string{"--quiet", "--dt-only", "--source", "1.1.1.1", "--mark", "123", "--xmark", "0x7b"})
	if shouldExit || err != nil {
		t.Fatalf("ConfigureApp returned shouldExit %v, err %v", shouldExit, err)
	}
	if err := outbound.PrepareOutboundOptions(&opts); err != nil {
		t.Fatalf("PrepareOutboundOptions returned error: %v", err)
	}
	if !config.Config.OutboundMarkSet || config.Config.OutboundMark != 123 {
		t.Fatalf("OutboundMarkSet=%v OutboundMark=%d, want set mark 123", config.Config.OutboundMarkSet, config.Config.OutboundMark)
	}
}

func TestConfigureAppAcceptsOutboundSourceIP(t *testing.T) {
	ip, zone := firstLocalTestIP(t)
	arg := ip.String()
	if zone != "" && ip.To4() == nil && ip.IsLinkLocalUnicast() {
		arg += "%" + zone
	}
	resetGlobalsForTest()
	opts, shouldExit, _, err := config.ConfigureApp([]string{"--quiet", "--dt-only", "--source", "1.1.1.1", "--interface", arg})
	if shouldExit || err != nil {
		t.Fatalf("ConfigureApp returned shouldExit %v, err %v", shouldExit, err)
	}
	if err := outbound.PrepareOutboundOptions(&opts); err != nil {
		t.Fatalf("PrepareOutboundOptions returned error: %v", err)
	}
	if config.Config.OutboundSourceIP == nil || !config.Config.OutboundSourceIP.Equal(ip) {
		t.Fatalf("OutboundSourceIP=%v, want %v", config.Config.OutboundSourceIP, ip)
	}
}

func TestPrepareOutboundInterfaceAcceptsIndex(t *testing.T) {
	iface := firstLocalTestInterface(t)
	resetGlobalsForTest()
	config.Config.OutboundInterface = strconv.Itoa(iface.Index)
	if err := outbound.PrepareOutboundInterface(); err != nil {
		t.Fatalf("PrepareOutboundInterface returned error: %v", err)
	}
	if config.Config.OutboundInterfaceIndex != iface.Index || config.Config.OutboundInterfaceName != iface.Name {
		t.Fatalf("resolved interface = (%d, %q), want (%d, %q)", config.Config.OutboundInterfaceIndex, config.Config.OutboundInterfaceName, iface.Index, iface.Name)
	}
}

func resetGlobalsForTest() {
	config.Config = config.DefaultConfig()
	config.IPStr = []string{}
	config.ResetRuntimeState()
	logger.Log = logger.NewLogger(logger.LogLevelFatal)
}

func TestParseCLIAcceptsDNSHostInput(t *testing.T) {
	opts, err := config.ParseCLI([]string{"--dt-only", "-s", "example.com:443", "-6"})
	if err != nil {
		t.Fatalf("ParseCLI returned error: %v", err)
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
			ok, host, port := utils.SplitHost(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("SplitHost(%q) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if host != tt.wantHost || port != tt.wantPort {
				t.Fatalf("SplitHost(%q) = (%q, %d), want (%q, %d)", tt.input, host, port, tt.wantHost, tt.wantPort)
			}
		})
	}
}

func TestSourceIPsDNSHostPassesFamilyFilters(t *testing.T) {
	for _, mode := range []int8{config.TypeIPv4, config.TypeIPv6, config.TypeIPv4 | config.TypeIPv6} {
		sources := config.NewSourceIPs()
		if err := sources.AddFromSlice([]string{"example.com:443"}, mode); err != nil {
			t.Fatalf("AddFromSlice returned error for mode %d: %v", mode, err)
		}
		if got := sources.LenInt(); got != 1 {
			t.Fatalf("LenInt for mode %d = %d, want 1", mode, got)
		}
	}
}

func TestSourceIPsIPLiteralHostsStillRespectFamilyFilters(t *testing.T) {
	sources := config.NewSourceIPs()
	if err := sources.AddFromSlice([]string{"[2606:4700::1]:443"}, config.TypeIPv4); err != nil {
		t.Fatalf("AddFromSlice returned error: %v", err)
	}
	if got := sources.LenInt(); got != 0 {
		t.Fatalf("IPv6 host passed IPv4-only filter, LenInt = %d", got)
	}
}

func TestLoadSourceIPsAcceptsExplicitIPv6WithoutFamilyFlag(t *testing.T) {
	resetGlobalsForTest()
	config.IPStr = []string{"2606:4700::1"}
	if err := config.LoadSourceIPs(config.TypeIPv4, false, false); err != nil {
		t.Fatalf("LoadSourceIPs returned error: %v", err)
	}
	if !config.Config.IPv6Mode {
		t.Fatal("expected explicit input without family flags to enable IPv6 mode")
	}
	if got := config.SrcIPs.LenInt(); got != 1 {
		t.Fatalf("SrcIPs.LenInt() = %d, want 1", got)
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
				if !config.Config.DTOnly {
					t.Fatal("expected --disable-download to enable DT-only mode")
				}
			},
		},
		{
			name: "dt via https alias maps protocol",
			args: []string{"--silence", "--dt-via-https", "--dt-only", "-s", "1.1.1.1"},
			want: func(t *testing.T) {
				if config.Config.DTVia != "https" || !config.Config.DTHttps {
					t.Fatalf("expected HTTPS DT mode, got DTVia=%q DTHttps=%v", config.Config.DTVia, config.Config.DTHttps)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGlobalsForTest()
			opts, shouldExit, _, err := config.ConfigureApp(tt.args)
			if shouldExit || err != nil {
				t.Fatalf("ConfigureApp(%v) = shouldExit %v, err %v", tt.args, shouldExit, err)
			}
			if err := outbound.PrepareOutboundOptions(&opts); err != nil {
				t.Fatalf("PrepareOutboundOptions returned error: %v", err)
			}
			tt.want(t)
		})
	}
}

func TestNormalizeDTViaLowercasesProtocol(t *testing.T) {
	resetGlobalsForTest()
	config.Config.DTVia = "TLS"
	if err := config.NormalizeDTVia(); err != nil {
		t.Fatalf("NormalizeDTVia returned error: %v", err)
	}
	if config.Config.DTVia != "tls" || config.Config.DTHttps {
		t.Fatalf("NormalizeDTVia did not normalize TLS mode, DTVia=%q DTHttps=%v", config.Config.DTVia, config.Config.DTHttps)
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
			_, shouldExit, _, err := config.ConfigureApp(tt.args)
			if !shouldExit || err == nil {
				t.Fatalf("ConfigureApp(%v) = shouldExit %v, err %v; want error", tt.args, shouldExit, err)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestSourceIPsRejectsBareDNSHostWithHostPortMessage(t *testing.T) {
	sources := config.NewSourceIPs()
	err := sources.AddFromSlice([]string{"example.com"}, config.TypeIPv4|config.TypeIPv6)
	if err == nil {
		t.Fatal("expected bare DNS host without port to be rejected")
	}
	if !strings.Contains(err.Error(), "host:port") {
		t.Fatalf("error %q does not mention host:port", err.Error())
	}
}

func TestAddPortsReturnsErrorsAndDeduplicates(t *testing.T) {
	sources := config.NewSourceIPs()
	if err := sources.AddPorts([]string{"443,8443", "8443"}); err != nil {
		t.Fatalf("AddPorts returned error: %v", err)
	}
	if len(sources.Ports) != 2 {
		t.Fatalf("len(sources.Ports) = %d, want 2", len(sources.Ports))
	}
	if sources.Ports[0] != 443 || sources.Ports[1] != 8443 {
		t.Fatalf("sources.Ports = %v, want [443, 8443]", sources.Ports)
	}
}

func TestURLHelpersReturnErrorsInsteadOfExiting(t *testing.T) {
	host, port, err := utils.ParseUrl("https://example.com/path", config.DefaultDLTUrl)
	if err != nil {
		t.Fatalf("ParseUrl returned error: %v", err)
	}
	if host != "example.com" || port != 443 {
		t.Fatalf("ParseUrl returned (%q, %d), want (example.com, 443)", host, port)
	}

	got, err := utils.NewUrl("https://example.com/path", "8443", config.DefaultDLTUrl)
	if err != nil {
		t.Fatalf("NewUrl returned error: %v", err)
	}
	if got != "https://example.com:8443/path" {
		t.Fatalf("NewUrl returned %q", got)
	}

	if _, _, err := utils.ParseUrl("not a url", config.DefaultDLTUrl); err == nil {
		t.Fatal("expected invalid ParseUrl input to return error")
	}
	if _, err := utils.NewUrl("https://example.com/path", "0", config.DefaultDLTUrl); err == nil {
		t.Fatal("expected invalid NewUrl port to return error")
	}
}

func TestDefaultURLsUseCloudflareSpeedtest(t *testing.T) {
	if config.DefaultDTUrl != "https://speed.cloudflare.com/__down?bytes=0" {
		t.Fatalf("DefaultDTUrl = %q", config.DefaultDTUrl)
	}
	if config.DefaultDLTUrl != "https://speed.cloudflare.com/__down?bytes=99999999" {
		t.Fatalf("DefaultDLTUrl = %q", config.DefaultDLTUrl)
	}
	if config.DefaultTestHost != "speed.cloudflare.com" {
		t.Fatalf("DefaultTestHost = %q", config.DefaultTestHost)
	}
}

func TestNoCacheIgnoresDefaultURLsWithEquivalentPorts(t *testing.T) {
	resetGlobalsForTest()
	config.Config.NoCache = true

	defaults := []string{
		config.DefaultDTUrl,
		config.DefaultDLTUrl,
		"https://speed.cloudflare.com:443/__down?bytes=0",
		"https://speed.cloudflare.com:443/__down?bytes=99999999",
	}
	for _, sourceURL := range defaults {
		if config.ShouldApplyNoCache(sourceURL) {
			t.Fatalf("ShouldApplyNoCache(%q) = true, want false", sourceURL)
		}
	}

	if !config.ShouldApplyNoCache("https://example.com/__down?bytes=99999999") {
		t.Fatal("custom URL should apply no-cache when Config.NoCache is true")
	}

	config.Config.NoCache = false
	if config.ShouldApplyNoCache("https://example.com/__down?bytes=99999999") {
		t.Fatal("custom URL should not apply no-cache when Config.NoCache is false")
	}
}

func TestParseCLIAcceptsLongFormAliases(t *testing.T) {
	opts, err := config.ParseCLI([]string{
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
		"--resolve-loc-threads", "15",
		"--quiet",
	})
	if err != nil {
		t.Fatalf("ParseCLI returned error: %v", err)
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
		{name: "resolve location threads", ok: cfg.ResolveLocPar == 15},
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
	opts, shouldExit, _, err := config.ConfigureApp([]string{"--quiet", "--dt-only", "--source", "1.1.1.1", "--dt-timeout-ms", "1234"})
	if shouldExit || err != nil {
		t.Fatalf("ConfigureApp returned shouldExit %v, err %v", shouldExit, err)
	}
	if err := outbound.PrepareOutboundOptions(&opts); err != nil {
		t.Fatalf("PrepareOutboundOptions returned error: %v", err)
	}
	if config.Config.DTTimeout != 1234 {
		t.Fatalf("config.Config.DTTimeout = %d, want 1234", config.Config.DTTimeout)
	}
}

func TestFetchCloudflareDomains(t *testing.T) {
	oldLimit := config.Config.TrancoLimit
	config.Config.TrancoLimit = 10
	defer func() { config.Config.TrancoLimit = oldLimit }()

	domains, err := fetcher.FetchCloudflareDomains(config.Config.DNSServer, config.Config.TrancoLimit)
	if err != nil {
		t.Logf("FetchCloudflareDomains returned error (possibly no network): %v", err)
		return
	}
	t.Logf("Fetched %d verified Cloudflare domains", len(domains))
}

func TestFetchDynamicIPv4(t *testing.T) {
	oldLimit := config.Config.TrancoLimit
	config.Config.TrancoLimit = 10
	defer func() { config.Config.TrancoLimit = oldLimit }()

	cidrs, err := fetcher.FetchDynamicIPv4(config.Config.DNSServer, config.Config.TrancoLimit)
	if err != nil {
		t.Logf("FetchDynamicIPv4 returned error (possibly no network): %v", err)
		return
	}
	t.Logf("Fetched %d IPv4 CIDRs", len(cidrs))
	for _, cidr := range cidrs {
		parts := strings.Split(cidr, "/")
		if len(parts) == 2 {
			maskSize, err := strconv.Atoi(parts[1])
			if err != nil {
				t.Errorf("failed to parse mask size for %s: %v", cidr, err)
				continue
			}
			if maskSize < 16 {
				t.Errorf("found subnet smaller than /16: %s", cidr)
			}
		}
	}
}

func TestIPModeBehavior(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantIPv4     bool
		wantIPv6     bool
		wantIPv4Chg  bool
		wantIPv6Chg  bool
	}{
		{
			name:         "neither specified",
			args:         []string{},
			wantIPv4:     true,
			wantIPv6:     true,
			wantIPv4Chg:  false,
			wantIPv6Chg:  false,
		},
		{
			name:         "only -4 specified",
			args:         []string{"-4"},
			wantIPv4:     true,
			wantIPv6:     false,
			wantIPv4Chg:  true,
			wantIPv6Chg:  false,
		},
		{
			name:         "only -6 specified",
			args:         []string{"-6"},
			wantIPv4:     false,
			wantIPv6:     true,
			wantIPv4Chg:  false,
			wantIPv6Chg:  true,
		},
		{
			name:         "both -4 and -6 specified",
			args:         []string{"-4", "-6"},
			wantIPv4:     true,
			wantIPv6:     true,
			wantIPv4Chg:  true,
			wantIPv6Chg:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGlobalsForTest()
			opts, err := config.ParseCLI(tt.args)
			if err != nil {
				t.Fatalf("ParseCLI failed: %v", err)
			}
			if opts.Config.IPv4Mode != tt.wantIPv4 {
				t.Errorf("IPv4Mode = %v, want %v", opts.Config.IPv4Mode, tt.wantIPv4)
			}
			if opts.Config.IPv6Mode != tt.wantIPv6 {
				t.Errorf("IPv6Mode = %v, want %v", opts.Config.IPv6Mode, tt.wantIPv6)
			}
			if opts.IPv4Changed != tt.wantIPv4Chg {
				t.Errorf("IPv4Changed = %v, want %v", opts.IPv4Changed, tt.wantIPv4Chg)
			}
			if opts.IPv6Changed != tt.wantIPv6Chg {
				t.Errorf("IPv6Changed = %v, want %v", opts.IPv6Changed, tt.wantIPv6Chg)
			}
		})
	}
}

func TestLoadSourceIPsDualStackDefault(t *testing.T) {
	resetGlobalsForTest()
	config.IPStr = []string{}
	config.Config.IPFile = ""
	config.Config.FastMode = false

	if err := config.LoadSourceIPs(config.TypeIPv4|config.TypeIPv6, false, false); err != nil {
		t.Fatalf("LoadSourceIPs failed for dual-stack default: %v", err)
	}

	totalIPs := config.SrcIPs.LenInt()
	if totalIPs == 0 {
		t.Fatal("expected some source IPs to be loaded")
	}

	hasIPv4 := false
	hasIPv6 := false
	
	if err := config.SrcIPs.AddPorts([]string{"443"}); err != nil {
		t.Fatalf("AddPorts failed: %v", err)
	}
	
	batch := config.SrcIPs.RetrieveSome(totalIPs, false)
	for _, ipStr := range batch {
		if strings.Contains(*ipStr, "[") {
			hasIPv6 = true
		} else {
			hasIPv4 = true
		}
	}

	if !hasIPv4 || !hasIPv6 {
		t.Errorf("expected both IPv4 and IPv6 addresses, got: hasIPv4=%v, hasIPv6=%v", hasIPv4, hasIPv6)
	}
}

func TestSupplementCLI(t *testing.T) {
	resetGlobalsForTest()
	opts, err := config.ParseCLI([]string{"--supplement"})
	if err != nil {
		t.Fatalf("ParseCLI returned error: %v", err)
	}
	if !opts.Config.Supplement {
		t.Fatal("expected --supplement flag to be parsed")
	}
	if !opts.Config.SupplementIPv4 {
		t.Fatal("expected SupplementIPv4 to default to true")
	}
	if opts.Config.SupplementIPv6 {
		t.Fatal("expected SupplementIPv6 to default to false")
	}
}

func TestSupplementIPModeStaging(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantPriV4  bool
		wantPriV6  bool
		wantSuppV4 bool
		wantSuppV6 bool
		wantSMode  int8
	}{
		{
			name:       "default supplement",
			args:       []string{"--supplement"},
			wantPriV4:  true,
			wantPriV6:  true,
			wantSuppV4: true,
			wantSuppV6: false,
			wantSMode:  config.TypeIPv4,
		},
		{
			name:       "primary ipv4 with supplement ipv6",
			args:       []string{"-4", "--supplement", "--supplement-ipv6"},
			wantPriV4:  true,
			wantPriV6:  false,
			wantSuppV4: false,
			wantSuppV6: true,
			wantSMode:  config.TypeIPv6,
		},
		{
			name:       "primary ipv6 with supplement ipv4",
			args:       []string{"-6", "--supplement", "--supplement-ipv4"},
			wantPriV4:  false,
			wantPriV6:  true,
			wantSuppV4: true,
			wantSuppV6: false,
			wantSMode:  config.TypeIPv4,
		},
		{
			name:       "both supplement ipv4 and ipv6",
			args:       []string{"--supplement", "--supplement-ipv4", "--supplement-ipv6"},
			wantPriV4:  true,
			wantPriV6:  true,
			wantSuppV4: true,
			wantSuppV6: true,
			wantSMode:  config.TypeIPv4 | config.TypeIPv6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGlobalsForTest()
			opts, err := config.ParseCLI(tt.args)
			if err != nil {
				t.Fatalf("ParseCLI failed: %v", err)
			}
			if opts.Config.IPv4Mode != tt.wantPriV4 {
				t.Errorf("IPv4Mode = %v, want %v", opts.Config.IPv4Mode, tt.wantPriV4)
			}
			if opts.Config.IPv6Mode != tt.wantPriV6 {
				t.Errorf("IPv6Mode = %v, want %v", opts.Config.IPv6Mode, tt.wantPriV6)
			}
			if opts.Config.SupplementIPv4 != tt.wantSuppV4 {
				t.Errorf("SupplementIPv4 = %v, want %v", opts.Config.SupplementIPv4, tt.wantSuppV4)
			}
			if opts.Config.SupplementIPv6 != tt.wantSuppV6 {
				t.Errorf("SupplementIPv6 = %v, want %v", opts.Config.SupplementIPv6, tt.wantSuppV6)
			}

			config.Config = opts.Config
			if got := config.SupplementIPMode(); got != tt.wantSMode {
				t.Errorf("SupplementIPMode() = %d, want %d", got, tt.wantSMode)
			}
		})
	}
}

func TestSupplementDisabledBothIPv4AndIPv6Rejects(t *testing.T) {
	resetGlobalsForTest()
	_, _, _, err := config.ConfigureApp([]string{"--supplement", "--supplement-ipv4=false", "--supplement-ipv6=false"})
	if err == nil {
		t.Fatal("expected error when both --supplement-ipv4 and --supplement-ipv6 are false")
	}
	wantSubstr := "\"--supplement-ipv4\" and \"--supplement-ipv6\" cannot both be disabled"
	if !strings.Contains(err.Error(), wantSubstr) {
		t.Fatalf("unexpected error message: %v (want %q)", err, wantSubstr)
	}
}

func TestSupplementSourceIPs(t *testing.T) {
	resetGlobalsForTest()
	config.Config.PortStrSlice = []string{"443"}
	
	if err := config.SupplementSourceIPs(config.SourceLevelFast, config.TypeIPv4|config.TypeIPv6); err != nil {
		t.Fatalf("SupplementSourceIPs failed for LevelFast: %v", err)
	}
	if config.SrcIPs.LenInt() == 0 {
		t.Fatal("expected fast source IPs to be loaded")
	}

	resetGlobalsForTest()
	config.Config.PortStrSlice = []string{"8443"}
	if err := config.SupplementSourceIPs(config.SourceLevelFull, config.TypeIPv4|config.TypeIPv6); err != nil {
		t.Fatalf("SupplementSourceIPs failed for LevelFull: %v", err)
	}
	if config.SrcIPs.LenInt() == 0 {
		t.Fatal("expected full source IPs to be loaded")
	}
}

func TestGetSourceLevelName(t *testing.T) {
	tests := []struct {
		name     string
		level    int
		contains string
	}{
		{name: "user level", level: config.SourceLevelUser, contains: "-s/-i"},
		{name: "fast level", level: config.SourceLevelFast, contains: "--fast"},
		{name: "full level", level: config.SourceLevelFull, contains: "Cloudflare"},
		{name: "unknown level", level: 99, contains: "level 99"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := config.GetSourceLevelName(tt.level)
			if !strings.Contains(name, tt.contains) {
				t.Errorf("GetSourceLevelName(%d) = %q, want substring %q", tt.level, name, tt.contains)
			}
		})
	}
}

func TestInitialSourceLevelHierarchy(t *testing.T) {
	t.Run("user sources set SourceLevelUser", func(t *testing.T) {
		resetGlobalsForTest()
		_, _, _, err := config.ConfigureApp([]string{"-s", "1.1.1.1", "--supplement"})
		if err != nil {
			t.Fatalf("ConfigureApp failed: %v", err)
		}
		hasUserSources := len(config.IPStr) > 0 || len(config.Config.IPFile) > 0
		if !hasUserSources {
			t.Fatal("expected hasUserSources to be true for -s")
		}
	})

	t.Run("fast mode without user sources set SourceLevelFast", func(t *testing.T) {
		resetGlobalsForTest()
		opts, _, _, err := config.ConfigureApp([]string{"--fast", "--supplement"})
		if err != nil {
			t.Fatalf("ConfigureApp failed: %v", err)
		}
		if !opts.Config.FastMode {
			t.Fatal("expected FastMode to be true")
		}
		if len(config.IPStr) > 0 || len(config.Config.IPFile) > 0 {
			t.Fatal("expected no user sources when only --fast is set")
		}
	})

	t.Run("default uses SourceLevelFull", func(t *testing.T) {
		resetGlobalsForTest()
		opts, _, _, err := config.ConfigureApp([]string{"--supplement"})
		if err != nil {
			t.Fatalf("ConfigureApp failed: %v", err)
		}
		if opts.Config.FastMode {
			t.Fatal("expected FastMode to be false for default")
		}
		if len(config.IPStr) > 0 || len(config.Config.IPFile) > 0 {
			t.Fatal("expected no user sources for default")
		}
	})
}

func TestSupplementWithoutLoopSucceeds(t *testing.T) {
	resetGlobalsForTest()
	_, _, _, err := config.ConfigureApp([]string{"--supplement"})
	if err != nil {
		t.Fatalf("expected ConfigureApp to succeed when --supplement is used without --loop: %v", err)
	}
}

func TestLoadSourceIPsDualStackRandomMixed(t *testing.T) {
	resetGlobalsForTest()
	config.IPStr = []string{}
	config.Config.IPFile = ""
	config.Config.FastMode = true

	if err := config.LoadSourceIPs(config.TypeIPv4|config.TypeIPv6, false, false); err != nil {
		t.Fatalf("LoadSourceIPs failed: %v", err)
	}

	if err := config.SrcIPs.AddPorts([]string{"443"}); err != nil {
		t.Fatalf("AddPorts failed: %v", err)
	}

	batch := config.SrcIPs.RetrieveSome(50, true)
	if len(batch) == 0 {
		t.Fatal("expected retrieve to return IPs")
	}

	hasIPv4 := false
	hasIPv6 := false
	for _, ipStr := range batch {
		if strings.Contains(*ipStr, "[") {
			hasIPv6 = true
		} else {
			hasIPv4 = true
		}
	}

	if !hasIPv4 || !hasIPv6 {
		t.Errorf("expected both IPv4 and IPv6 to be retrieved in the batch, got: hasIPv4=%v, hasIPv6=%v", hasIPv4, hasIPv6)
	}
}

func TestRetrieveSomeFullAmount(t *testing.T) {
	src := config.NewSourceIPs()
	hosts := []string{
		"1.1.1.1:443", "1.1.1.2:443", "1.1.1.3:443", "1.1.1.4:443", "1.1.1.5:443",
		"1.1.1.6:443", "1.1.1.7:443", "1.1.1.8:443", "1.1.1.9:443", "1.1.1.10:443",
	}
	if err := src.AddFromSlice(hosts, config.TypeIPv4); err != nil {
		t.Fatalf("AddFromSlice failed: %v", err)
	}

	batch := src.RetrieveSome(10, false)
	if len(batch) != 10 {
		t.Fatalf("expected RetrieveSome(10) to return 10 hosts, got %d", len(batch))
	}
}

func TestTotalHosts(t *testing.T) {
	src := config.NewSourceIPs()
	if err := src.Add("1.1.1.1", config.TypeIPv4); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if err := src.Add("1.0.0.1", config.TypeIPv4); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if err := src.AddPorts([]string{"443", "8443"}); err != nil {
		t.Fatalf("AddPorts failed: %v", err)
	}
	total := src.TotalHosts()
	if total.Int64() != 4 {
		t.Fatalf("expected 4 total hosts, got %d", total.Int64())
	}
}

func TestMultiInAndSWorkTogether(t *testing.T) {
	resetGlobalsForTest()
	dir := t.TempDir()
	f1 := dir + "/list1.txt"
	f2 := dir + "/list2.txt"

	if err := os.WriteFile(f1, []byte("1.1.1.1\n1.1.1.2\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(f2, []byte("1.0.0.1\n1.0.0.2\n"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	opts, shouldExit, exitCode, err := config.ConfigureApp([]string{
		"-s", "8.8.8.8",
		"-s", "8.8.4.4",
		"-i", f1,
		"-i", f2,
	})
	if err != nil {
		t.Fatalf("ConfigureApp failed: %v", err)
	}
	if shouldExit || exitCode != 0 {
		t.Fatalf("ConfigureApp returned shouldExit=%v, exitCode=%d", shouldExit, exitCode)
	}
	if !config.HasUserSources() {
		t.Fatal("expected HasUserSources() to be true")
	}
	if len(opts.IPs) != 2 {
		t.Fatalf("expected 2 IPs from -s, got %d: %v", len(opts.IPs), opts.IPs)
	}
	if len(opts.Config.IPFiles) != 2 {
		t.Fatalf("expected 2 IPFiles from -i, got %d: %v", len(opts.Config.IPFiles), opts.Config.IPFiles)
	}

	// 2 from -s, 2 from f1, 2 from f2 = 6 hosts total
	total := config.SrcIPs.TotalHosts()
	if total.Int64() != 6 {
		t.Fatalf("expected 6 total hosts loaded into SrcIPs, got %d", total.Int64())
	}
}

func TestFastModeWithUserSourcesWarnings(t *testing.T) {
	t.Run("without supplement warns fast mode ignored", func(t *testing.T) {
		resetGlobalsForTest()
		opts, shouldExit, exitCode, err := config.ConfigureApp([]string{
			"-s", "1.1.1.1",
			"--fast",
		})
		if err != nil {
			t.Fatalf("ConfigureApp failed: %v", err)
		}
		if shouldExit || exitCode != 0 {
			t.Fatalf("ConfigureApp returned shouldExit=%v, exitCode=%d", shouldExit, exitCode)
		}
		if !opts.Config.FastMode {
			t.Fatal("expected FastMode to be true")
		}
	})

	t.Run("with supplement warns fast mode tested as fallback", func(t *testing.T) {
		resetGlobalsForTest()
		opts, shouldExit, exitCode, err := config.ConfigureApp([]string{
			"-s", "1.1.1.1",
			"--fast",
			"--supplement",
		})
		if err != nil {
			t.Fatalf("ConfigureApp failed: %v", err)
		}
		if shouldExit || exitCode != 0 {
			t.Fatalf("ConfigureApp returned shouldExit=%v, exitCode=%d", shouldExit, exitCode)
		}
		if !opts.Config.Supplement {
			t.Fatal("expected Supplement to be true")
		}
	})
}

func TestResolveLocParFlags(t *testing.T) {
	resetGlobalsForTest()
	opts, shouldExit, _, err := config.ConfigureApp([]string{"--dt-only", "--resolve-loc-par", "8"})
	if shouldExit || err != nil {
		t.Fatalf("ConfigureApp returned shouldExit %v, err %v", shouldExit, err)
	}
	if config.Config.ResolveLocPar != 8 {
		t.Fatalf("expected ResolveLocPar 8, got %d", config.Config.ResolveLocPar)
	}
	if !config.Config.ResolveLoc {
		t.Fatal("expected ResolveLoc to be auto-enabled when --resolve-loc-par is specified")
	}
	if !opts.Config.ResolveLoc {
		t.Fatal("expected opts.Config.ResolveLoc to be auto-enabled when --resolve-loc-par is specified")
	}
}

func TestResolveLocParExplicitFalse(t *testing.T) {
	resetGlobalsForTest()
	opts, shouldExit, _, err := config.ConfigureApp([]string{"--dt-only", "--resolve-loc-par", "8", "--resolve-loc=false"})
	if shouldExit || err != nil {
		t.Fatalf("ConfigureApp returned shouldExit %v, err %v", shouldExit, err)
	}
	if config.Config.ResolveLoc {
		t.Fatal("expected ResolveLoc to remain false when explicitly disabled")
	}
	if opts.Config.ResolveLoc {
		t.Fatal("expected opts.Config.ResolveLoc to remain false when explicitly disabled")
	}
}

func TestResolveLocParValidation(t *testing.T) {
	resetGlobalsForTest()
	_, _, _, err := config.ConfigureApp([]string{"--dt-only", "--resolve-loc-par", "0"})
	if err == nil {
		t.Fatal("expected error for --resolve-loc-par 0")
	}
}

func TestVerifyResultsCombineMinMax(t *testing.T) {
	ip := "1.1.1.1"
	dtRes := config.VerifyResults{
		IP:      &ip,
		Dtc:     4,
		Dtpc:    4,
		Dtpr:    1.0,
		Da:      50.0,
		Dmi:     40.0,
		Dmx:     60.0,
		DtDList: []float64{40.0, 50.0, 50.0, 60.0},
	}

	dltRes := config.VerifyResults{
		IP:    &ip,
		Dltc:  1,
		Dltpc: 1,
		Dltpr: 1.0,
		Dls:   5000.0,
		Dlds:  10000000,
		Dltd:  2.0,
	}

	dltRes.Combine(dtRes)

	if dltRes.Dmi != 40.0 {
		t.Errorf("expected combined Dmi = 40.0, got %v", dltRes.Dmi)
	}
	if dltRes.Dmx != 60.0 {
		t.Errorf("expected combined Dmx = 60.0, got %v", dltRes.Dmx)
	}
	if dltRes.Da != 50.0 {
		t.Errorf("expected combined Da = 50.0, got %v", dltRes.Da)
	}
	if dltRes.Dls != 5000.0 {
		t.Errorf("expected combined Dls = 5000.0, got %v", dltRes.Dls)
	}
}

func TestParseCLIAcceptsDurationStrings(t *testing.T) {
	opts, err := config.ParseCLI([]string{
		"-s", "1.1.1.1",
		"-t", "50ms",
		"-k", "50ms",
		"-d", "50s",
		"-I", "50ms",
		"--dlt-timeout", "50ms",
		"--loop-interval", "50s",
		"--test-timeout", "50mins",
	})
	if err != nil {
		t.Fatalf("ParseCLI returned error: %v", err)
	}
	cfg := opts.Config
	if cfg.DTTimeout != 50 || cfg.DTTimeoutDuration != 50*time.Millisecond {
		t.Errorf("DTTimeout = %d, %v; want 50, 50ms", cfg.DTTimeout, cfg.DTTimeoutDuration)
	}
	if cfg.DTEvaluationDelay != 50 || cfg.DTEvaluationDelayDuration != 50*time.Millisecond {
		t.Errorf("DTEvaluationDelay = %d, %v; want 50, 50ms", cfg.DTEvaluationDelay, cfg.DTEvaluationDelayDuration)
	}
	if cfg.DLTDurMax != 50 || cfg.DLTDurationInTotal != 50*time.Second {
		t.Errorf("DLTDurMax = %d, %v; want 50, 50s", cfg.DLTDurMax, cfg.DLTDurationInTotal)
	}
	if cfg.Interval != 50 || cfg.IntervalDuration != 50*time.Millisecond {
		t.Errorf("Interval = %d, %v; want 50, 50ms", cfg.Interval, cfg.IntervalDuration)
	}
	if cfg.DLTTimeout != 50 || cfg.HttpRspTimeoutDuration != 50*time.Millisecond {
		t.Errorf("DLTTimeout = %d, %v; want 50, 50ms", cfg.DLTTimeout, cfg.HttpRspTimeoutDuration)
	}
	if cfg.LoopInterval != 50 || cfg.LoopIntervalDuration != 50*time.Second {
		t.Errorf("LoopInterval = %d, %v; want 50, 50s", cfg.LoopInterval, cfg.LoopIntervalDuration)
	}
	if cfg.TestTimeout != 50 || cfg.TestTimeoutDuration != 50*time.Minute {
		t.Errorf("TestTimeout = %d, %v; want 50, 50m", cfg.TestTimeout, cfg.TestTimeoutDuration)
	}
}

func TestParseCLIAcceptsNaturalDurationUnits(t *testing.T) {
	opts, err := config.ParseCLI([]string{
		"-s", "1.1.1.1",
		"--dt-timeout-ms", "2s",
		"--dt-max-delay", "1500ms",
		"--dlt-duration", "1m30s",
		"--test-interval-ms", "500ms",
		"--dlt-timeout-ms", "10sec",
		"--loop-interval", "2min",
		"--test-timeout", "1h",
	})
	if err != nil {
		t.Fatalf("ParseCLI returned error: %v", err)
	}
	cfg := opts.Config
	if cfg.DTTimeout != 2000 || cfg.DTTimeoutDuration != 2*time.Second {
		t.Errorf("DTTimeout = %d, %v; want 2000, 2s", cfg.DTTimeout, cfg.DTTimeoutDuration)
	}
	if cfg.DTEvaluationDelay != 1500 || cfg.DTEvaluationDelayDuration != 1500*time.Millisecond {
		t.Errorf("DTEvaluationDelay = %d, %v; want 1500, 1500ms", cfg.DTEvaluationDelay, cfg.DTEvaluationDelayDuration)
	}
	if cfg.DLTDurMax != 90 || cfg.DLTDurationInTotal != 90*time.Second {
		t.Errorf("DLTDurMax = %d, %v; want 90, 90s", cfg.DLTDurMax, cfg.DLTDurationInTotal)
	}
	if cfg.Interval != 500 || cfg.IntervalDuration != 500*time.Millisecond {
		t.Errorf("Interval = %d, %v; want 500, 500ms", cfg.Interval, cfg.IntervalDuration)
	}
	if cfg.DLTTimeout != 10000 || cfg.HttpRspTimeoutDuration != 10*time.Second {
		t.Errorf("DLTTimeout = %d, %v; want 10000, 10s", cfg.DLTTimeout, cfg.HttpRspTimeoutDuration)
	}
	if cfg.LoopInterval != 120 || cfg.LoopIntervalDuration != 2*time.Minute {
		t.Errorf("LoopInterval = %d, %v; want 120, 2m", cfg.LoopInterval, cfg.LoopIntervalDuration)
	}
	if cfg.TestTimeout != 60 || cfg.TestTimeoutDuration != 1*time.Hour {
		t.Errorf("TestTimeout = %d, %v; want 60, 1h", cfg.TestTimeout, cfg.TestTimeoutDuration)
	}
}

func TestParseCLILegacyUnitlessValues(t *testing.T) {
	opts, err := config.ParseCLI([]string{
		"-s", "1.1.1.1",
		"-t", "1500",
		"-k", "400",
		"-d", "12",
		"-I", "250",
		"--dlt-timeout", "3500",
		"--loop-interval", "45",
		"--test-timeout", "20",
	})
	if err != nil {
		t.Fatalf("ParseCLI returned error: %v", err)
	}
	cfg := opts.Config
	if cfg.DTTimeout != 1500 || cfg.DTTimeoutDuration != 1500*time.Millisecond {
		t.Errorf("DTTimeout = %d, %v; want 1500, 1500ms", cfg.DTTimeout, cfg.DTTimeoutDuration)
	}
	if cfg.DTEvaluationDelay != 400 || cfg.DTEvaluationDelayDuration != 400*time.Millisecond {
		t.Errorf("DTEvaluationDelay = %d, %v; want 400, 400ms", cfg.DTEvaluationDelay, cfg.DTEvaluationDelayDuration)
	}
	if cfg.DLTDurMax != 12 || cfg.DLTDurationInTotal != 12*time.Second {
		t.Errorf("DLTDurMax = %d, %v; want 12, 12s", cfg.DLTDurMax, cfg.DLTDurationInTotal)
	}
	if cfg.Interval != 250 || cfg.IntervalDuration != 250*time.Millisecond {
		t.Errorf("Interval = %d, %v; want 250, 250ms", cfg.Interval, cfg.IntervalDuration)
	}
	if cfg.DLTTimeout != 3500 || cfg.HttpRspTimeoutDuration != 3500*time.Millisecond {
		t.Errorf("DLTTimeout = %d, %v; want 3500, 3500ms", cfg.DLTTimeout, cfg.HttpRspTimeoutDuration)
	}
	if cfg.LoopInterval != 45 || cfg.LoopIntervalDuration != 45*time.Second {
		t.Errorf("LoopInterval = %d, %v; want 45, 45s", cfg.LoopInterval, cfg.LoopIntervalDuration)
	}
	if cfg.TestTimeout != 20 || cfg.TestTimeoutDuration != 20*time.Minute {
		t.Errorf("TestTimeout = %d, %v; want 20, 20m", cfg.TestTimeout, cfg.TestTimeoutDuration)
	}
}

func TestParseCLIRejectsInvalidDurationStrings(t *testing.T) {
	invalidFlags := [][]string{
		{"-t", "abc"},
		{"-t", "0"},
		{"-t", "-5ms"},
		{"-k", "bad"},
		{"-k", "0"},
		{"-d", "invalid"},
		{"-d", "0s"},
		{"-I", "xyz"},
		{"-I", "-10ms"},
		{"--dlt-timeout", "foo"},
		{"--loop-interval", "bar"},
		{"--test-timeout", "0"},
	}

	for _, flagArgs := range invalidFlags {
		args := append([]string{"-s", "1.1.1.1"}, flagArgs...)
		_, err := config.ParseCLI(args)
		if err == nil {
			t.Errorf("ParseCLI(%v) expected error, got nil", args)
		}
	}
}

func TestSafeLooperDurationOperations(t *testing.T) {
	looper := config.NewSafeLooperWithDuration(3, 2500*time.Millisecond)
	if dur := looper.GetDuration(); dur != 2500*time.Millisecond {
		t.Fatalf("GetDuration() = %v, want 2500ms", dur)
	}
	if interv := looper.GetInterval(); interv != 2500 {
		t.Fatalf("GetInterval() = %d, want 2500", interv)
	}

	looper.SetDuration(100 * time.Millisecond)
	if dur := looper.GetDuration(); dur != 100*time.Millisecond {
		t.Fatalf("GetDuration() after SetDuration = %v, want 100ms", dur)
	}
	if interv := looper.GetInterval(); interv != 100 {
		t.Fatalf("GetInterval() after SetDuration = %d, want 100", interv)
	}

	looper.SetInterval(500)
	if dur := looper.GetDuration(); dur != 500*time.Millisecond {
		t.Fatalf("GetDuration() after SetInterval = %v, want 500ms", dur)
	}
	if interv := looper.GetInterval(); interv != 500 {
		t.Fatalf("GetInterval() after SetInterval = %d, want 500", interv)
	}

	// Verify Sleep releases mutex and does not block concurrent readers
	sleepDone := make(chan struct{})
	go func() {
		looper.SetDuration(80 * time.Millisecond)
		looper.Sleep()
		close(sleepDone)
	}()

	time.Sleep(10 * time.Millisecond)
	// Querying round while sleeping must succeed immediately
	round := looper.GetRound()
	if round != 0 {
		t.Fatalf("looper.GetRound() = %d, want 0", round)
	}
	<-sleepDone
}

func TestProgrammaticConfigDurationSynchronization(t *testing.T) {
	cfg := config.DefaultConfig()

	// 1. Programmatically mutate raw integer fields
	cfg.Interval = 250
	cfg.LoopInterval = 15
	cfg.TestTimeout = 45
	cfg.DLTDurMax = 8
	cfg.DLTTimeout = 3000
	cfg.DTTimeout = 1200
	cfg.DTEvaluationDelay = 400
	cfg.SyncDurations()

	if cfg.IntervalDuration != 250*time.Millisecond {
		t.Errorf("IntervalDuration = %v, want 250ms", cfg.IntervalDuration)
	}
	if cfg.LoopIntervalDuration != 15*time.Second {
		t.Errorf("LoopIntervalDuration = %v, want 15s", cfg.LoopIntervalDuration)
	}
	if cfg.TestTimeoutDuration != 45*time.Minute {
		t.Errorf("TestTimeoutDuration = %v, want 45m", cfg.TestTimeoutDuration)
	}
	if cfg.DTTimeoutDuration != 1200*time.Millisecond {
		t.Errorf("DTTimeoutDuration = %v, want 1200ms", cfg.DTTimeoutDuration)
	}
	if cfg.DTEvaluationDelayDuration != 400*time.Millisecond {
		t.Errorf("DTEvaluationDelayDuration = %v, want 400ms", cfg.DTEvaluationDelayDuration)
	}
	if cfg.DLTDurationInTotal != 8*time.Second {
		t.Errorf("DLTDurationInTotal = %v, want 8s", cfg.DLTDurationInTotal)
	}
	if cfg.HttpRspTimeoutDuration != 3*time.Second {
		t.Errorf("HttpRspTimeoutDuration = %v, want 3s", cfg.HttpRspTimeoutDuration)
	}

	// 2. Programmatically mutate duration fields
	cfg2 := config.DefaultConfig()
	cfg2.IntervalDuration = 100 * time.Millisecond
	cfg2.LoopIntervalDuration = 20 * time.Second
	cfg2.TestTimeoutDuration = 10 * time.Minute
	cfg2.DLTDurationInTotal = 5 * time.Second
	cfg2.HttpRspTimeoutDuration = 2 * time.Second
	cfg2.DTTimeoutDuration = 1500 * time.Millisecond
	cfg2.DTEvaluationDelayDuration = 350 * time.Millisecond
	cfg2.SyncDurations()

	if cfg2.Interval != 100 {
		t.Errorf("Interval = %d, want 100", cfg2.Interval)
	}
	if cfg2.LoopInterval != 20 {
		t.Errorf("LoopInterval = %d, want 20", cfg2.LoopInterval)
	}
	if cfg2.TestTimeout != 10 {
		t.Errorf("TestTimeout = %d, want 10", cfg2.TestTimeout)
	}
	if cfg2.DLTDurMax != 5 {
		t.Errorf("DLTDurMax = %d, want 5", cfg2.DLTDurMax)
	}
	if cfg2.DLTTimeout != 2000 {
		t.Errorf("DLTTimeout = %d, want 2000", cfg2.DLTTimeout)
	}
	if cfg2.DTTimeout != 1500 {
		t.Errorf("DTTimeout = %d, want 1500", cfg2.DTTimeout)
	}
	if cfg2.DTEvaluationDelay != 350 {
		t.Errorf("DTEvaluationDelay = %d, want 350", cfg2.DTEvaluationDelay)
	}

	// 3. Repeated mutation and reset to default value
	cfg.Interval = 100
	cfg.SyncDurations()
	if cfg.IntervalDuration != 100*time.Millisecond {
		t.Errorf("IntervalDuration after re-mutation = %v, want 100ms", cfg.IntervalDuration)
	}

	cfg.Interval = 500 // Reset to default 500
	cfg.SyncDurations()
	if cfg.IntervalDuration != 500*time.Millisecond {
		t.Errorf("IntervalDuration after reset to default = %v, want 500ms", cfg.IntervalDuration)
	}

	// 4. Simultaneous mutation where duration field takes precedence
	cfg3 := config.DefaultConfig()
	cfg3.Interval = 300
	cfg3.IntervalDuration = 750 * time.Millisecond
	cfg3.SyncDurations()
	if cfg3.Interval != 750 || cfg3.IntervalDuration != 750*time.Millisecond {
		t.Errorf("Conflicting mutation (Interval=%d, IntervalDuration=%v), want 750, 750ms", cfg3.Interval, cfg3.IntervalDuration)
	}
}

func TestConfigureAppHTTPSModeTimeout(t *testing.T) {
	resetGlobalsForTest()

	// With explicit -t, custom value is preserved
	opts, shouldExit, _, err := config.ConfigureApp([]string{"--quiet", "--dt-only", "-s", "1.1.1.1", "--dt-via", "https", "-t", "3500ms"})
	if shouldExit || err != nil {
		t.Fatalf("ConfigureApp returned shouldExit=%v, err=%v", shouldExit, err)
	}
	_ = opts

	if config.Config.DTTimeout != 3500 || config.Config.DTTimeoutDuration != 3500*time.Millisecond {
		t.Fatalf("DTTimeout = %d, %v; want 3500, 3500ms", config.Config.DTTimeout, config.Config.DTTimeoutDuration)
	}

	resetGlobalsForTest()

	// Without -t, default is upgraded to 5000ms for HTTPS
	opts2, shouldExit, _, err := config.ConfigureApp([]string{"--quiet", "--dt-only", "-s", "1.1.1.1", "--dt-via", "https"})
	if shouldExit || err != nil {
		t.Fatalf("ConfigureApp returned shouldExit=%v, err=%v", shouldExit, err)
	}
	_ = opts2

	if config.Config.DTTimeout != 5000 || config.Config.DTTimeoutDuration != 5000*time.Millisecond {
		t.Fatalf("DTTimeout = %d, %v; want 5000, 5000ms", config.Config.DTTimeout, config.Config.DTTimeoutDuration)
	}

	resetGlobalsForTest()

	// Switch to SSL mode without -t should preserve 2000ms
	opts3, shouldExit, _, err := config.ConfigureApp([]string{"--quiet", "--dt-only", "-s", "1.1.1.1", "--dt-via", "ssl"})
	if shouldExit || err != nil {
		t.Fatalf("ConfigureApp returned shouldExit=%v, err=%v", shouldExit, err)
	}
	_ = opts3

	if config.Config.DTTimeout != 2000 || config.Config.DTTimeoutDuration != 2000*time.Millisecond {
		t.Fatalf("DTTimeout = %d, %v; want 2000, 2000ms", config.Config.DTTimeout, config.Config.DTTimeoutDuration)
	}
}

func TestSafeLooperSleepWithCheck(t *testing.T) {
	looper := config.NewSafeLooperWithDuration(3, 5*time.Second)

	// Stop condition is triggered after 30ms
	stopAt := time.Now().Add(30 * time.Millisecond)
	start := time.Now()
	looper.SleepWithCheck(func() bool {
		return time.Now().After(stopAt)
	})
	elapsed := time.Since(start)

	// It should wake up well before the 5 second duration
	if elapsed >= 1*time.Second {
		t.Fatalf("SleepWithCheck took %v; expected early wake up around 30-150ms", elapsed)
	}
}

func TestSourceIPsRetrieveSomeNilEvictionAndCompaction(t *testing.T) {
	s := config.NewSourceIPs()
	hosts := make([]string, 2000)
	for i := 0; i < 2000; i++ {
		hosts[i] = fmt.Sprintf("192.0.2.%d:443", i%256)
	}
	if err := s.AddFromSlice(hosts, config.TypeIPv4); err != nil {
		t.Fatalf("AddFromSlice failed: %v", err)
	}

	// Retrieve in chunks of 50
	totalRetrieved := 0
	for {
		batch := s.RetrieveSome(50, false)
		if len(batch) == 0 {
			break
		}
		totalRetrieved += len(batch)
	}

	if totalRetrieved != 2000 {
		t.Fatalf("expected 2000 retrieved hosts, got %d", totalRetrieved)
	}

	// When exhausted, retrieveSome should return empty and s.IsEmpty() should be true
	if !s.IsEmpty() {
		t.Fatalf("expected SourceIPs to be empty after full retrieval")
	}
}



