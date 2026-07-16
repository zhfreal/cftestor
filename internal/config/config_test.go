package config_test

import (
	"net"
	"runtime"
	"strconv"
	"strings"
	"testing"

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

