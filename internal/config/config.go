package config

import (
	"fmt"
	"math"
	"math/big"
	"net/url"
	"os"
	"strings"
	"time"

	"cftestor/internal/fetcher"
	"cftestor/internal/logger"
	"cftestor/internal/utils"
	utls "github.com/metacubex/utls"
	flag "github.com/spf13/pflag"
)

type CliOptions struct {
	Config                AppConfig
	IPs                   []string
	SourceIPs             []string
	IPFiles               []string
	SourceFiles           []string
	Mark                  string
	XMark                 string
	PrintVersion          bool
	TLSHelloFirefox       bool
	TLSHelloChrome        bool
	TLSHelloEdge          bool
	TLSHelloSafari        bool
	IPv4Changed           bool
	IPv6Changed           bool
	DTTimeoutChanged      bool
	SupplementIPv4Changed bool
	SupplementIPv6Changed bool
	MarkChanged           bool
	XMarkChanged          bool
	ResolveLocChanged     bool
	ResolveLocParChanged  bool

	DTTimeoutRaw         string
	DLTTimeoutRaw        string
	DLTDurMaxRaw         string
	IntervalRaw          string
	DTEvaluationDelayRaw string
	LoopIntervalRaw      string
	TestTimeoutRaw       string
}

func init() {
	fetcher.FallbackPrefixes = append(append([]string{}, CFIPV4FULL...), CFIPV6FULL...)
}

func (cfg *AppConfig) initLastSync() {
	cfg.lastInterval = cfg.Interval
	cfg.lastIntervalDuration = cfg.IntervalDuration
	cfg.lastLoopInterval = cfg.LoopInterval
	cfg.lastLoopIntervalDuration = cfg.LoopIntervalDuration
	cfg.lastTestTimeout = cfg.TestTimeout
	cfg.lastTestTimeoutDuration = cfg.TestTimeoutDuration
	cfg.lastDTTimeout = cfg.DTTimeout
	cfg.lastDTTimeoutDuration = cfg.DTTimeoutDuration
	cfg.lastDTEvaluationDelay = cfg.DTEvaluationDelay
	cfg.lastDTEvaluationDelayDuration = cfg.DTEvaluationDelayDuration
	cfg.lastDLTDurMax = cfg.DLTDurMax
	cfg.lastDLTDurationInTotal = cfg.DLTDurationInTotal
	cfg.lastDLTTimeout = cfg.DLTTimeout
	cfg.lastHttpRspTimeoutDuration = cfg.HttpRspTimeoutDuration
}

func DefaultConfig() AppConfig {
	cfg := AppConfig{
		DTCount:                     4,
		DTWorkerThread:              20,
		DLTDurMax:                   10,
		DLTDurationInTotal:          10 * time.Second,
		DLTWorkerThread:             1,
		DLTCount:                    1,
		ResultMin:                   10,
		Interval:                    500,
		IntervalDuration:            500 * time.Millisecond,
		DTEvaluationDelay:           600,
		DTEvaluationDelayDuration:   600 * time.Millisecond,
		DTTimeout:                   2000,
		DTTimeoutDuration:           2000 * time.Millisecond,
		DTStdExp:                    30,
		HostName:                    DefaultTestHost,
		DLTUrl:                      DefaultDLTUrl,
		DTUrl:                       DefaultDTUrl,
		DLTTimeout:                  5000,
		HttpRspTimeoutDuration:      5000 * time.Millisecond,
		Loop:                        -1,
		TestTimeout:                 30,
		TestTimeoutDuration:         30 * time.Minute,
		LoopInterval:                60,
		LoopIntervalDuration:        60 * time.Second,
		DTEvaluationDTPR:            100,
		DLTEvaluationSpeed:          6000,
		DTVia:                       "https",
		DTHttpRspReturnCodeExpected: 200,
		IPv4Mode:                    true,
		IPv6Mode:                    true,
		SupplementIPv4:              true,
		SupplementIPv6:              false,
		TLSClientID:                 utls.HelloChrome_Auto,
		UserAgent:                   UserAgentChrome,
		PortStrSlice:                []string{},
		DNSServer:                   "1.1.1.1:53",
		TrancoLimit:                 1000,
		ResolveLocPar:               10,
	}
	cfg.initLastSync()
	return cfg
}

func durationToMs(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	ms64 := int64(d / time.Millisecond)
	if ms64 > math.MaxInt {
		return math.MaxInt
	}
	if ms64 == 0 {
		return 1
	}
	return int(ms64)
}

func durationToSeconds(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	s64 := int64(d / time.Second)
	if s64 > math.MaxInt {
		return math.MaxInt
	}
	if s64 == 0 {
		return 1
	}
	return int(s64)
}

func durationToMinutes(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	m64 := int64(d / time.Minute)
	if m64 > math.MaxInt {
		return math.MaxInt
	}
	if m64 == 0 {
		return 1
	}
	return int(m64)
}

func syncMs(dur *time.Duration, ms *int, lastDur *time.Duration, lastMs *int) {
	durChanged := *dur != *lastDur
	msChanged := *ms != *lastMs

	if durChanged && msChanged {
		*ms = durationToMs(*dur)
	} else if durChanged {
		*ms = durationToMs(*dur)
	} else if msChanged {
		if *ms > 0 {
			*dur = time.Duration(*ms) * time.Millisecond
		} else {
			*dur = 0
		}
	} else if *dur <= 0 && *ms > 0 {
		*dur = time.Duration(*ms) * time.Millisecond
	} else if *ms <= 0 && *dur > 0 {
		*ms = durationToMs(*dur)
	}

	*lastDur = *dur
	*lastMs = *ms
}

func syncSeconds(dur *time.Duration, s *int, lastDur *time.Duration, lastS *int) {
	durChanged := *dur != *lastDur
	sChanged := *s != *lastS

	if durChanged && sChanged {
		*s = durationToSeconds(*dur)
	} else if durChanged {
		*s = durationToSeconds(*dur)
	} else if sChanged {
		if *s > 0 {
			*dur = time.Duration(*s) * time.Second
		} else {
			*dur = 0
		}
	} else if *dur <= 0 && *s > 0 {
		*dur = time.Duration(*s) * time.Second
	} else if *s <= 0 && *dur > 0 {
		*s = durationToSeconds(*dur)
	}

	*lastDur = *dur
	*lastS = *s
}

func syncMinutes(dur *time.Duration, m *int, lastDur *time.Duration, lastM *int) {
	durChanged := *dur != *lastDur
	mChanged := *m != *lastM

	if durChanged && mChanged {
		*m = durationToMinutes(*dur)
	} else if durChanged {
		*m = durationToMinutes(*dur)
	} else if mChanged {
		if *m > 0 {
			*dur = time.Duration(*m) * time.Minute
		} else {
			*dur = 0
		}
	} else if *dur <= 0 && *m > 0 {
		*dur = time.Duration(*m) * time.Minute
	} else if *m <= 0 && *dur > 0 {
		*m = durationToMinutes(*dur)
	}

	*lastDur = *dur
	*lastM = *m
}

func syncDurationAndMs(dur *time.Duration, ms *int, defaultVal int) {
	if dur == &Config.IntervalDuration || dur == &Config.DTTimeoutDuration || dur == &Config.DTEvaluationDelayDuration || dur == &Config.HttpRspTimeoutDuration {
		Config.SyncDurations()
		return
	}
	defaultDur := time.Duration(defaultVal) * time.Millisecond
	lastDur := defaultDur
	lastMs := defaultVal
	syncMs(dur, ms, &lastDur, &lastMs)
}

func syncDurationAndSeconds(dur *time.Duration, s *int, defaultVal int) {
	if dur == &Config.LoopIntervalDuration || dur == &Config.DLTDurationInTotal {
		Config.SyncDurations()
		return
	}
	defaultDur := time.Duration(defaultVal) * time.Second
	lastDur := defaultDur
	lastS := defaultVal
	syncSeconds(dur, s, &lastDur, &lastS)
}

func syncDurationAndMinutes(dur *time.Duration, m *int, defaultVal int) {
	if dur == &Config.TestTimeoutDuration {
		Config.SyncDurations()
		return
	}
	defaultDur := time.Duration(defaultVal) * time.Minute
	lastDur := defaultDur
	lastM := defaultVal
	syncMinutes(dur, m, &lastDur, &lastM)
}

// SyncDurations synchronizes any differences between duration fields and their integer companion fields in AppConfig.
func (cfg *AppConfig) SyncDurations() {
	syncMs(&cfg.IntervalDuration, &cfg.Interval, &cfg.lastIntervalDuration, &cfg.lastInterval)
	syncSeconds(&cfg.LoopIntervalDuration, &cfg.LoopInterval, &cfg.lastLoopIntervalDuration, &cfg.lastLoopInterval)
	syncMinutes(&cfg.TestTimeoutDuration, &cfg.TestTimeout, &cfg.lastTestTimeoutDuration, &cfg.lastTestTimeout)
	syncMs(&cfg.DTTimeoutDuration, &cfg.DTTimeout, &cfg.lastDTTimeoutDuration, &cfg.lastDTTimeout)
	syncMs(&cfg.DTEvaluationDelayDuration, &cfg.DTEvaluationDelay, &cfg.lastDTEvaluationDelayDuration, &cfg.lastDTEvaluationDelay)
	syncSeconds(&cfg.DLTDurationInTotal, &cfg.DLTDurMax, &cfg.lastDLTDurationInTotal, &cfg.lastDLTDurMax)
	syncMs(&cfg.HttpRspTimeoutDuration, &cfg.DLTTimeout, &cfg.lastHttpRspTimeoutDuration, &cfg.lastDLTTimeout)
}

func ParseCLI(args []string) (CliOptions, error) {
	opts := CliOptions{
		Config:      DefaultConfig(),
		IPs:         []string{},
		SourceIPs:   []string{},
		IPFiles:     []string{},
		SourceFiles: []string{},
	}
	fs := flag.NewFlagSet(RunTime, flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Print(Help)
	}
	RegisterCLIFlags(fs, &opts)
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	opts.IPs = append(opts.IPs, opts.SourceIPs...)
	opts.IPFiles = append(opts.IPFiles, opts.SourceFiles...)
	opts.Config.IPFiles = opts.IPFiles
	if len(opts.IPFiles) > 0 {
		opts.Config.IPFile = opts.IPFiles[0]
	}
	opts.IPv4Changed = FlagChanged(fs, "ipv4")
	opts.IPv6Changed = FlagChanged(fs, "ipv6")
	opts.DTTimeoutChanged = FlagChanged(fs, "dt-timeout", "dt-timeout-ms")
	opts.MarkChanged = FlagChanged(fs, "mark")
	opts.XMarkChanged = FlagChanged(fs, "xmark")
	opts.SupplementIPv4Changed = FlagChanged(fs, "supplement-ipv4")
	opts.SupplementIPv6Changed = FlagChanged(fs, "supplement-ipv6")
	opts.ResolveLocChanged = FlagChanged(fs, "resolve-loc", "resolve-location")
	opts.ResolveLocParChanged = FlagChanged(fs, "resolve-loc-par", "resolve-loc-threads")

	if opts.DTTimeoutChanged {
		d, err := utils.ParseDurationString(opts.DTTimeoutRaw, time.Millisecond)
		if err != nil {
			return opts, fmt.Errorf("invalid value for \"-t|--dt-timeout\": %w", err)
		}
		if d <= 0 {
			return opts, fmt.Errorf("%q must be greater than 0 (got %s)", "-t|--dt-timeout", opts.DTTimeoutRaw)
		}
		opts.Config.DTTimeoutDuration = d
		opts.Config.DTTimeout = durationToMs(d)
	}
	if FlagChanged(fs, "dlt-timeout", "dlt-timeout-ms") {
		d, err := utils.ParseDurationString(opts.DLTTimeoutRaw, time.Millisecond)
		if err != nil {
			return opts, fmt.Errorf("invalid value for \"--dlt-timeout\": %w", err)
		}
		if d <= 0 {
			return opts, fmt.Errorf("%q must be greater than 0 (got %s)", "--dlt-timeout", opts.DLTTimeoutRaw)
		}
		opts.Config.HttpRspTimeoutDuration = d
		opts.Config.DLTTimeout = durationToMs(d)
	}
	if FlagChanged(fs, "dlt-period", "dlt-duration") {
		d, err := utils.ParseDurationString(opts.DLTDurMaxRaw, time.Second)
		if err != nil {
			return opts, fmt.Errorf("invalid value for \"-d|--dlt-period\": %w", err)
		}
		if d <= 0 {
			return opts, fmt.Errorf("%q must be greater than 0 (got %s)", "-d|--dlt-period", opts.DLTDurMaxRaw)
		}
		opts.Config.DLTDurationInTotal = d
		opts.Config.DLTDurMax = durationToSeconds(d)
	}
	if FlagChanged(fs, "interval", "test-interval-ms") {
		d, err := utils.ParseDurationString(opts.IntervalRaw, time.Millisecond)
		if err != nil {
			return opts, fmt.Errorf("invalid value for \"-I|--interval\": %w", err)
		}
		if d <= 0 {
			return opts, fmt.Errorf("%q must be greater than 0 (got %s)", "-I|--interval", opts.IntervalRaw)
		}
		opts.Config.IntervalDuration = d
		opts.Config.Interval = durationToMs(d)
	}
	if FlagChanged(fs, "ev-dt-delay", "dt-max-delay") {
		d, err := utils.ParseDurationString(opts.DTEvaluationDelayRaw, time.Millisecond)
		if err != nil {
			return opts, fmt.Errorf("invalid value for \"-k|--ev-dt-delay\": %w", err)
		}
		if d <= 0 {
			return opts, fmt.Errorf("%q must be greater than 0 (got %s)", "-k|--ev-dt-delay", opts.DTEvaluationDelayRaw)
		}
		opts.Config.DTEvaluationDelayDuration = d
		opts.Config.DTEvaluationDelay = durationToMs(d)
	}
	if FlagChanged(fs, "loop-interval") {
		d, err := utils.ParseDurationString(opts.LoopIntervalRaw, time.Second)
		if err != nil {
			return opts, fmt.Errorf("invalid value for \"--loop-interval\": %w", err)
		}
		if d <= 0 {
			return opts, fmt.Errorf("%q must be greater than 0 (got %s)", "--loop-interval", opts.LoopIntervalRaw)
		}
		opts.Config.LoopIntervalDuration = d
		opts.Config.LoopInterval = durationToSeconds(d)
	}
	if FlagChanged(fs, "test-timeout") {
		d, err := utils.ParseDurationString(opts.TestTimeoutRaw, time.Minute)
		if err != nil {
			return opts, fmt.Errorf("invalid value for \"--test-timeout\": %w", err)
		}
		if d <= 0 {
			return opts, fmt.Errorf("%q must be greater than 0 (got %s)", "--test-timeout", opts.TestTimeoutRaw)
		}
		opts.Config.TestTimeoutDuration = d
		opts.Config.TestTimeout = durationToMinutes(d)
	}
	if opts.IPv4Changed && !opts.IPv6Changed {
		opts.Config.IPv6Mode = false
	} else if opts.IPv6Changed && !opts.IPv4Changed {
		opts.Config.IPv4Mode = false
	}
	if opts.SupplementIPv6Changed && !opts.SupplementIPv4Changed {
		opts.Config.SupplementIPv4 = false
		opts.Config.SupplementIPv6 = true
	}
	ApplyTLSFingerprint(&opts)
	opts.Config.SyncDurations()
	return opts, nil
}

func RegisterCLIFlags(fs *flag.FlagSet, opts *CliOptions) {
	cfg := &opts.Config

	fs.BoolVar(&cfg.FastMode, "fast", cfg.FastMode, "Use a limited set of internal Cloudflare IPs for quick scanning.")
	fs.StringSliceVarP(&opts.IPs, "ip", "s", opts.IPs, "IP, CIDR, or host:port candidate to test. Can be provided multiple times.")
	fs.StringSliceVar(&opts.SourceIPs, "source", opts.SourceIPs, "Alias for --ip.")
	fs.StringSliceVarP(&opts.IPFiles, "in", "i", opts.IPFiles, "Path to a file containing IPs, CIDRs, or host:port entries. Can be provided multiple times.")
	fs.StringSliceVar(&opts.SourceFiles, "source-file", opts.SourceFiles, "Alias for --in.")

	fs.IntVarP(&cfg.DTWorkerThread, "dt-thread", "m", cfg.DTWorkerThread, "Number of concurrent Delay Test (DT) workers.")
	fs.IntVar(&cfg.DTWorkerThread, "dt-workers", cfg.DTWorkerThread, "Alias for --dt-thread.")
	fs.StringVarP(&opts.DTTimeoutRaw, "dt-timeout", "t", opts.DTTimeoutRaw, "Timeout for a single DT attempt in ms or duration (e.g. 50ms, 2s).")
	fs.StringVar(&opts.DTTimeoutRaw, "dt-timeout-ms", opts.DTTimeoutRaw, "Alias for --dt-timeout.")
	fs.IntVarP(&cfg.DTCount, "dt-count", "c", cfg.DTCount, "Number of DT attempts per candidate.")
	fs.IntVar(&cfg.DTCount, "dt-attempts", cfg.DTCount, "Alias for --dt-count.")
	fs.StringSliceVarP(&cfg.PortStrSlice, "port", "p", cfg.PortStrSlice, "Port(s) for IP/CIDR inputs. Supports single ports, ranges, and lists.")
	fs.StringVar(&cfg.HostName, "hostname", cfg.HostName, "SNI hostname for TLS/SSL DT.")
	fs.StringVar(&cfg.HostName, "sni-hostname", cfg.HostName, "Alias for --hostname.")
	fs.StringVar(&cfg.DTVia, "dt-via", cfg.DTVia, "Delay-test protocol: https, tls, or ssl.")
	fs.StringVar(&cfg.DTVia, "dt-protocol", cfg.DTVia, "Alias for --dt-via.")
	fs.IntVar(&cfg.DTHttpRspReturnCodeExpected, "dt-expect-code", cfg.DTHttpRspReturnCodeExpected, "HTTP status code expected for DT test.")
	fs.IntVar(&cfg.DTHttpRspReturnCodeExpected, "dt-status-code", cfg.DTHttpRspReturnCodeExpected, "Alias for --dt-expect-code.")
	fs.BoolVar(&cfg.DTHttps, "dt-via-https", cfg.DTHttps, "Deprecated alias for --dt-via https.")
	fs.StringVar(&cfg.DTUrl, "dt-url", cfg.DTUrl, "URL to use for HTTPS-based DT.")

	fs.IntVarP(&cfg.DLTWorkerThread, "dlt-thread", "n", cfg.DLTWorkerThread, "Number of concurrent Download Test (DLT) workers.")
	fs.IntVar(&cfg.DLTWorkerThread, "dlt-workers", cfg.DLTWorkerThread, "Alias for --dlt-thread.")
	fs.StringVarP(&opts.DLTDurMaxRaw, "dlt-period", "d", opts.DLTDurMaxRaw, "Maximum duration for one DLT attempt in seconds or duration (e.g. 50s, 10s, 10000ms).")
	fs.StringVar(&opts.DLTDurMaxRaw, "dlt-duration", opts.DLTDurMaxRaw, "Alias for --dlt-period.")
	fs.IntVarP(&cfg.DLTCount, "dlt-count", "b", cfg.DLTCount, "Number of DLT attempts per candidate.")
	fs.IntVar(&cfg.DLTCount, "dlt-attempts", cfg.DLTCount, "Alias for --dlt-count.")
	fs.StringVarP(&cfg.DLTUrl, "dlt-url", "u", cfg.DLTUrl, "URL to use for DLT.")
	fs.StringVar(&opts.DLTTimeoutRaw, "dlt-timeout", opts.DLTTimeoutRaw, "HTTP response timeout for DLT in ms or duration (e.g. 50ms, 5s).")
	fs.StringVar(&opts.DLTTimeoutRaw, "dlt-timeout-ms", opts.DLTTimeoutRaw, "Alias for --dlt-timeout.")
	fs.StringVarP(&opts.IntervalRaw, "interval", "I", opts.IntervalRaw, "Interval between test attempts in ms or duration (e.g. 50ms, 500ms, 1s).")
	fs.StringVar(&opts.IntervalRaw, "test-interval-ms", opts.IntervalRaw, "Alias for --interval.")

	fs.BoolVar(&cfg.EnableDTEvaluation, "ev-dt", cfg.EnableDTEvaluation, "Enable DT evaluation using all attempts.")
	fs.BoolVar(&cfg.EnableDTEvaluation, "dt-evaluate", cfg.EnableDTEvaluation, "Alias for --ev-dt.")
	fs.StringVarP(&opts.DTEvaluationDelayRaw, "ev-dt-delay", "k", opts.DTEvaluationDelayRaw, "Maximum allowed average DT delay in ms or duration (e.g. 50ms, 600ms).")
	fs.StringVar(&opts.DTEvaluationDelayRaw, "dt-max-delay", opts.DTEvaluationDelayRaw, "Alias for --ev-dt-delay.")
	fs.Float64Var(&cfg.DTEvaluationDTPR, "ev-dt-dtpr", cfg.DTEvaluationDTPR, "Minimum required DT pass rate percentage.")
	fs.Float64Var(&cfg.DTEvaluationDTPR, "dt-min-pass-rate", cfg.DTEvaluationDTPR, "Alias for --ev-dt-dtpr.")
	fs.Float64Var(&cfg.DTStdExp, "ev-dt-std", cfg.DTStdExp, "Maximum allowed DT standard deviation when enabled.")
	fs.Float64Var(&cfg.DTStdExp, "dt-max-stddev", cfg.DTStdExp, "Alias for --ev-dt-std.")
	fs.Float64VarP(&cfg.DLTEvaluationSpeed, "speed", "l", cfg.DLTEvaluationSpeed, "Minimum required download speed in KB/s.")
	fs.Float64Var(&cfg.DLTEvaluationSpeed, "min-speed", cfg.DLTEvaluationSpeed, "Alias for --speed.")
	fs.IntVar(&cfg.Loop, "loop", cfg.Loop, "Retest qualified candidates for N confirmation cycles; refill from the original pool if fewer than --result remain.")
	fs.StringVar(&opts.LoopIntervalRaw, "loop-interval", opts.LoopIntervalRaw, "Time to wait between loop cycles in seconds or duration (e.g. 50s, 60s, 1m).")
	fs.BoolVar(&cfg.Supplement, "supplement", cfg.Supplement, "Enable multi-tier IP source supplementation when target result count is not met (User IPs -> --fast -> Full CF CIDRs).")
	fs.BoolVar(&cfg.SupplementIPv4, "supplement-ipv4", cfg.SupplementIPv4, "Enable IPv4 for supplementation fallback.")
	fs.BoolVar(&cfg.SupplementIPv6, "supplement-ipv6", cfg.SupplementIPv6, "Enable IPv6 for supplementation fallback.")
	fs.IntVarP(&cfg.ResultMin, "result", "r", cfg.ResultMin, "Target number of final qualified results.")
	fs.IntVar(&cfg.ResultMin, "result-count", cfg.ResultMin, "Alias for --result.")

	fs.BoolVar(&cfg.DisableDownload, "disable-download", cfg.DisableDownload, "Deprecated, use --dt-only instead.")
	fs.BoolVar(&cfg.DTOnly, "dt-only", cfg.DTOnly, "Perform Delay Test only.")
	fs.BoolVar(&cfg.DLTOnly, "dlt-only", cfg.DLTOnly, "Perform Download Test only.")
	fs.BoolVarP(&cfg.IPv4Mode, "ipv4", "4", cfg.IPv4Mode, "Test IPv4 only.")
	fs.BoolVarP(&cfg.IPv6Mode, "ipv6", "6", cfg.IPv6Mode, "Test IPv6 only.")
	fs.StringVar(&cfg.FetchIPv6File, "fetch-ipv6", cfg.FetchIPv6File, "Fetch active Cloudflare IPv6 CIDRs dynamically, save to file, and exit.")
	fs.StringVar(&cfg.FetchIPv4File, "fetch-ipv4", cfg.FetchIPv4File, "Fetch active Cloudflare IPv4 CIDRs dynamically, save to file, and exit.")
	fs.StringVar(&cfg.FetchCFDomainsFile, "fetch-cf-domains", cfg.FetchCFDomainsFile, "Fetch, verify, and save top domains using Cloudflare CDN to a file, and exit.")
	fs.StringVar(&cfg.DNSServer, "dns", cfg.DNSServer, "Custom DNS server for dynamic fetching (e.g. 1.1.1.1:53, tls://1.1.1.1, https://1.1.1.1/dns-query)")
	fs.IntVar(&cfg.TrancoLimit, "tranco-limit", cfg.TrancoLimit, "Number of top Tranco domains to fetch for dynamic scanning verification.")
	fs.BoolVarP(&cfg.TestAll, "test-all", "a", cfg.TestAll, "Test all IPs until no more IP left.")
	fs.StringVar(&opts.Mark, "mark", opts.Mark, "Set Linux socket fwmark for outbound packets. Supports decimal and hex.")
	fs.StringVar(&opts.XMark, "xmark", opts.XMark, "Alias for --mark.")
	fs.StringVar(&cfg.OutboundInterface, "interface", cfg.OutboundInterface, "Bind outbound packets to an interface name, interface index, or local source IP.")
	fs.BoolVar(&opts.TLSHelloFirefox, "hello-firefox", opts.TLSHelloFirefox, "Simulate Firefox TLS fingerprint.")
	fs.BoolVar(&opts.TLSHelloChrome, "hello-chrome", opts.TLSHelloChrome, "Simulate Chrome TLS fingerprint.")
	fs.BoolVar(&opts.TLSHelloEdge, "hello-edge", opts.TLSHelloEdge, "Simulate Edge TLS fingerprint.")
	fs.BoolVar(&opts.TLSHelloSafari, "hello-safari", opts.TLSHelloSafari, "Simulate Safari TLS fingerprint.")
	fs.StringVar(&opts.TestTimeoutRaw, "test-timeout", opts.TestTimeoutRaw, "Test timeout in minutes or duration (e.g. 50mins, 50s, 30m).")

	fs.BoolVarP(&cfg.StoreToFile, "to-file", "w", cfg.StoreToFile, "Save results to a CSV file.")
	fs.BoolVar(&cfg.StoreToFile, "to-csv", cfg.StoreToFile, "Alias for --to-file.")
	fs.StringVarP(&cfg.ResultFile, "out-file", "o", cfg.ResultFile, "Path for the output CSV file.")
	fs.StringVar(&cfg.ResultFile, "csv-file", cfg.ResultFile, "Alias for --out-file.")
	fs.BoolVarP(&cfg.StoreToDB, "to-db", "e", cfg.StoreToDB, "Save results to a SQLite3 database.")
	fs.BoolVar(&cfg.StoreToDB, "to-sqlite", cfg.StoreToDB, "Alias for --to-db.")
	fs.BoolVar(&cfg.ResolveLocalASNAndCity, "local-asn", cfg.ResolveLocalASNAndCity, "Retrieve and store local ASN/city info.")
	fs.StringVarP(&cfg.DBFile, "db-file", "f", cfg.DBFile, "Path for the SQLite3 database file.")
	fs.StringVar(&cfg.DBFile, "sqlite-file", cfg.DBFile, "Alias for --db-file.")
	fs.StringVarP(&cfg.SuffixLabel, "label", "g", cfg.SuffixLabel, "Label for output files and database records.")
	fs.StringVar(&cfg.SuffixLabel, "record-label", cfg.SuffixLabel, "Alias for --label.")
	fs.BoolVar(&cfg.ResolveLoc, "resolve-loc", cfg.ResolveLoc, "Attempt to resolve and display Cloudflare location.")
	fs.BoolVar(&cfg.ResolveLoc, "resolve-location", cfg.ResolveLoc, "Alias for --resolve-loc.")
	fs.IntVar(&cfg.ResolveLocPar, "resolve-loc-par", cfg.ResolveLocPar, "Number of concurrent threads to resolve locations in parallel.")
	fs.IntVar(&cfg.ResolveLocPar, "resolve-loc-threads", cfg.ResolveLocPar, "Alias for --resolve-loc-par.")
	fs.BoolVarP(&cfg.NoCache, "no-cache", "C", cfg.NoCache, "Bypass CDN/proxy caching for custom URLs.")

	fs.BoolVarP(&cfg.SilenceMode, "silence", "S", cfg.SilenceMode, "Enable silence mode with minimal output.")
	fs.BoolVar(&cfg.SilenceMode, "quiet", cfg.SilenceMode, "Alias for --silence.")
	fs.BoolVarP(&cfg.Debug, "debug", "V", cfg.Debug, "Print debug message.")
	fs.BoolVarP(&opts.PrintVersion, "version", "v", opts.PrintVersion, "Show version.")
}

func FlagChanged(fs *flag.FlagSet, names ...string) bool {
	for _, name := range names {
		f := fs.Lookup(name)
		if f != nil && f.Changed {
			return true
		}
	}
	return false
}

func ApplyTLSFingerprint(opts *CliOptions) {
	cfg := &opts.Config
	if opts.TLSHelloFirefox {
		cfg.TLSClientID = utls.HelloFirefox_Auto
		cfg.UserAgent = UserAgentFirefox
	}
	if opts.TLSHelloChrome {
		cfg.TLSClientID = utls.HelloChrome_Auto
		cfg.UserAgent = UserAgentChrome
	}
	if opts.TLSHelloEdge {
		cfg.TLSClientID = utls.HelloEdge_Auto
		cfg.UserAgent = UserAgentEdge
	}
	if opts.TLSHelloSafari {
		cfg.TLSClientID = utls.HelloSafari_Auto
		cfg.UserAgent = UserAgentSafari
	}
}

func LoadSourceIPs(tMode int8, ipv4Changed, ipv6Changed bool) error {
	hasUserSources := HasUserSources()
	if !hasUserSources {
		if (tMode & TypeIPv4) == TypeIPv4 {
			tCFIPv4 := CFIPV4FULL
			if Config.FastMode {
				logger.Log.Infoln("Fast mode enabled for IPv4: dynamically fetching optimized active IPv4 CIDRs...")
				cidrs, err := fetcher.FetchDynamicIPv4(Config.DNSServer, Config.TrancoLimit)
				if err != nil {
					logger.Log.Warningf("Dynamic fetch failed, falling back to fast ranges: %v", err)
					tCFIPv4 = CFIPV4
				} else if len(cidrs) > 0 {
					tCFIPv4 = cidrs
				} else {
					tCFIPv4 = CFIPV4
				}
			}
			if err := SrcIPs.AddFromSlice(tCFIPv4, TypeIPv4); err != nil {
				return err
			}
		}
		if (tMode & TypeIPv6) == TypeIPv6 {
			tCFIPv6 := CFIPV6FULL
			if Config.FastMode {
				logger.Log.Infoln("Fast mode enabled for IPv6: dynamically fetching optimized active IPv6 CIDRs...")
				cidrs, err := fetcher.FetchDynamicIPv6(Config.DNSServer, Config.TrancoLimit)
				if err != nil {
					logger.Log.Warningf("Dynamic fetch failed, falling back to fast ranges: %v", err)
					tCFIPv6 = CFIPV6
				} else if len(cidrs) > 0 {
					tCFIPv6 = cidrs
				} else {
					tCFIPv6 = CFIPV6
				}
			}
			if err := SrcIPs.AddFromSlice(tCFIPv6, TypeIPv6); err != nil {
				return err
			}
		}
		return nil
	}

	if !ipv6Changed && !ipv4Changed {
		Config.IPv6Mode = true
		tMode = TypeIPv4 | TypeIPv6
	}
	if len(IPStr) > 0 {
		if err := SrcIPs.AddFromSlice(IPStr, tMode); err != nil {
			return err
		}
	}
	if len(Config.IPFiles) > 0 {
		for _, file := range Config.IPFiles {
			file = strings.TrimSpace(file)
			if file == "" {
				continue
			}
			if err := SrcIPs.AddFromFile(file, tMode); err != nil {
				return err
			}
		}
	} else if len(Config.IPFile) != 0 {
		if err := SrcIPs.AddFromFile(Config.IPFile, tMode); err != nil {
			return err
		}
	}
	if SrcIPs.LenInt() == 0 {
		return fmt.Errorf("no source IPs provided")
	}
	return nil
}


func ResetRuntimeState() {
	VerifyResultsMap = make(map[string]VerifyResults)
	MyRand = utils.NewRand()
	SrcIPs = NewSourceIPsWithRand(MyRand)
}

func NormalizeDTVia() error {
	Config.DTVia = strings.ToLower(Config.DTVia)
	switch Config.DTVia {
	case "https":
		Config.DTHttps = true
	case "ssl", "tls":
		Config.DTHttps = false
	default:
		return fmt.Errorf("invalid value for \"--dt-via\": use one of https, tls, or ssl")
	}
	return nil
}

func PrintVersionInfo() {
	fmt.Println(AppArt)
	fmt.Println(`  CF CDN IP scanner, find best IPs for you.
  https://github.com/zhfreal/cftestor`)
	fmt.Println()
	fmt.Printf("Version:    %v\n", Version)
	fmt.Printf("BuildDate:  %v\n", BuildDate)
	fmt.Printf("BuildTag:   %v\n", BuildTag)
	fmt.Printf("BuildHash:  %v\n", BuildHash)
	fmt.Println()
}

func ConfigureApp(args []string) (CliOptions, bool, int, error) {
	opts, err := ParseCLI(args)
	if err != nil {
		if err == flag.ErrHelp {
			return opts, true, 0, nil
		}
		return opts, true, 2, err
	}

	Config = opts.Config
	IPStr = opts.IPs
	VerifyResultsMap = make(map[string]VerifyResults)
	utils.InitRandSeed(MyRand)
	SrcIPs = NewSourceIPsWithRand(MyRand)

	if len(Version) == 0 {
		Version = "dev"
	}
	if Config.SilenceMode {
		Config.Debug = false
		Config.StoreToDB = false
		Config.StoreToFile = false
	}

	InitLoggerFromConfig()
	if Config.Debug && !opts.PrintVersion {
		PrintVersionInfo()
	}
	if opts.PrintVersion {
		return opts, true, 0, nil
	}
	if err := prepareRuntime(&opts); err != nil {
		return opts, true, 1, err
	}
	return opts, false, 0, nil
}

func InitLoggerFromConfig() {
	if Config.SilenceMode {
		logger.Log.LoggerLevel = logger.LogLevelFatal
	} else {
		logger.Log.LoggerLevel = logger.LogLevelInfo
		if Config.Debug {
			logger.Log.LoggerLevel = logger.LogLevelDebug
		}
	}
}

func ShouldApplyNoCache(sourceURL string) bool {
	return Config.NoCache && !IsDefaultTestURL(sourceURL)
}

func IsDefaultTestURL(sourceURL string) bool {
	return EquivalentURL(sourceURL, DefaultDTUrl) || EquivalentURL(sourceURL, DefaultDLTUrl)
}

func EquivalentURL(a, b string) bool {
	aURL, err := url.Parse(strings.TrimSpace(a))
	if err != nil || aURL == nil {
		return false
	}
	bURL, err := url.Parse(strings.TrimSpace(b))
	if err != nil || bURL == nil {
		return false
	}
	return strings.EqualFold(aURL.Scheme, bURL.Scheme) &&
		strings.EqualFold(strings.TrimSuffix(aURL.Hostname(), "."), strings.TrimSuffix(bURL.Hostname(), ".")) &&
		effectiveURLPort(aURL) == effectiveURLPort(bURL) &&
		normalizedURLPath(aURL) == normalizedURLPath(bURL) &&
		aURL.Query().Encode() == bURL.Query().Encode()
}

func effectiveURLPort(u *url.URL) string {
	if port := u.Port(); len(port) > 0 {
		return port
	}
	switch strings.ToLower(u.Scheme) {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}

func normalizedURLPath(u *url.URL) string {
	if len(u.EscapedPath()) == 0 {
		return "/"
	}
	return u.EscapedPath()
}

func prepareRuntime(opts *CliOptions) error {
	if HasUserSources() && Config.FastMode {
		if !Config.Supplement {
			logger.Log.Warningln("--fast wouldn't work when user provide sources (-s/-i), you can try --supplement and -r <N>.")
		} else {
			logger.Log.Warningln("sources from --fast just tested after no enough -r <N> archived, --fast can be omitted while --supplement provided.")
		}
	}
	if Config.Supplement && !Config.SupplementIPv4 && !Config.SupplementIPv6 {
		return fmt.Errorf("%q and %q cannot both be disabled when %q is enabled", "--supplement-ipv4", "--supplement-ipv6", "--supplement")
	}
	if Config.DisableDownload {
		Config.DTOnly = true
		logger.Log.Warningln("deprecated flag \"--disable-download\"; use \"--dt-only\" instead")
	}
	if Config.DTHttps {
		Config.DTVia = "https"
		logger.Log.Warningln("deprecated flag \"--dt-via-https\"; use \"--dt-via https\" instead")
	}
	if Config.DTOnly && Config.DLTOnly {
		return fmt.Errorf("%q and %q cannot be provided at the same time", "--dt-only", "--dlt-only")
	}
	if Config.DTEvaluationDTPR > 100 {
		Config.DTEvaluationDTPR = 100
	} else if Config.DTEvaluationDTPR < 0 {
		Config.DTEvaluationDTPR = 0
	}
	if err := NormalizeDTVia(); err != nil {
		return err
	}

	tMode, err := selectedIPMode(opts.IPv4Changed, opts.IPv6Changed)
	if err != nil {
		return err
	}
	trimConfigStrings()

	if err := LoadSourceIPs(tMode, opts.IPv4Changed, opts.IPv6Changed); err != nil {
		return err
	}
	if err := validateURLs(); err != nil {
		return err
	}
	if err := prepareSourcePool(); err != nil {
		return err
	}
	if err := prepareTestModes(opts.DTTimeoutChanged); err != nil {
		return err
	}
	if opts.ResolveLocParChanged && !opts.ResolveLocChanged {
		Config.ResolveLoc = true
		opts.Config.ResolveLoc = true
	}
	if Config.ResolveLocPar <= 0 {
		return positiveIntFlagError("--resolve-loc-par", Config.ResolveLocPar)
	}
	syncDurationAndSeconds(&Config.LoopIntervalDuration, &Config.LoopInterval, 60)
	if Config.LoopIntervalDuration <= 0 {
		return positiveIntFlagError("--loop-interval", Config.LoopInterval)
	}

	syncDurationAndMinutes(&Config.TestTimeoutDuration, &Config.TestTimeout, 30)
	if Config.TestTimeoutDuration <= 0 {
		return positiveIntFlagError("--test-timeout", Config.TestTimeout)
	}

	prepareOutputTargets()
	return nil
}

func selectedIPMode(ipv4Changed, ipv6Changed bool) (int8, error) {
	if ipv4Changed && !ipv6Changed {
		Config.IPv6Mode = false
	} else if ipv6Changed && !ipv4Changed {
		Config.IPv4Mode = false
	}
	tMode := int8(0)
	if Config.IPv4Mode {
		tMode |= TypeIPv4
	}
	if Config.IPv6Mode {
		tMode |= TypeIPv6
	}
	if tMode == TypeIPErr {
		return tMode, fmt.Errorf("IPv4 and IPv6 cannot both be disabled")
	}
	return tMode, nil
}

func trimConfigStrings() {
	Config.IPFile = strings.TrimSpace(Config.IPFile)
	cleanedFiles := make([]string, 0, len(Config.IPFiles))
	for _, f := range Config.IPFiles {
		trimmed := strings.TrimSpace(f)
		if trimmed != "" {
			cleanedFiles = append(cleanedFiles, trimmed)
		}
	}
	Config.IPFiles = cleanedFiles
	if len(Config.IPFiles) > 0 {
		Config.IPFile = Config.IPFiles[0]
	}
	Config.ResultFile = strings.TrimSpace(Config.ResultFile)
	Config.SuffixLabel = strings.TrimSpace(Config.SuffixLabel)
	Config.HostName = strings.TrimSpace(Config.HostName)
	Config.DTUrl = strings.TrimSpace(Config.DTUrl)
	Config.DLTUrl = strings.TrimSpace(Config.DLTUrl)
	Config.DBFile = strings.TrimSpace(Config.DBFile)
	Config.OutboundInterface = strings.TrimSpace(Config.OutboundInterface)
}

func validateURLs() error {
	if !Config.DLTOnly && Config.DTHttps {
		tURL, err := validateHTTPSURL(Config.DTUrl, "--dt-url")
		if err != nil {
			return err
		}
		Config.DTUrl = tURL
	}
	if !Config.DTOnly {
		tURL, err := validateHTTPSURL(Config.DLTUrl, "--dlt-url")
		if err != nil {
			return err
		}
		Config.DLTUrl = tURL
	}
	return nil
}

func validateHTTPSURL(urlStr, flagName string) (string, error) {
	if len(urlStr) == 0 {
		return "", fmt.Errorf("%q must not be empty", flagName)
	}
	tURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("%q must be a valid URL (got %q)", flagName, urlStr)
	}
	if tURL.Scheme != "https" {
		return "", fmt.Errorf("%q must use HTTPS (got %q)", flagName, urlStr)
	}
	return tURL.String(), nil
}

func prepareSourcePool() error {
	syncDurationAndMs(&Config.IntervalDuration, &Config.Interval, 500)
	if Config.IntervalDuration <= 0 {
		return positiveIntFlagError("-I|--interval", Config.Interval)
	}
	if Config.ResultMin <= 0 {
		return positiveIntFlagError("-r|--result", Config.ResultMin)
	}
	SrcIPs.Shuffle()
	if err := SrcIPs.AddPorts(Config.PortStrSlice); err != nil {
		return err
	}
	tQty := SrcIPs.Len()
	if Config.TestAll {
		Config.ResultMin = -1
	} else {
		tResultMin := big.NewInt(int64(Config.ResultMin))
		if tQty.Cmp(tResultMin) == -1 && !Config.Supplement {
			Config.ResultMin = int(tQty.Int64())
		}
	}
	if len(Config.SuffixLabel) == 0 {
		Config.SuffixLabel = Config.HostName
	}
	return nil
}

func positiveIntFlagError(flagName string, value int) error {
	return fmt.Errorf("%q must be greater than 0 (got %d)", flagName, value)
}

func prepareTestModes(dtTimeoutChanged bool) error {
	if !Config.DLTOnly {
		if err := prepareDelayTest(dtTimeoutChanged); err != nil {
			return err
		}
	}
	if !Config.DTOnly {
		if err := prepareDownloadTest(); err != nil {
			return err
		}
	}
	return nil
}

func prepareDelayTest(dtTimeoutChanged bool) error {
	if Config.DTWorkerThread <= 0 {
		return positiveIntFlagError("-m|--dt-thread", Config.DTWorkerThread)
	}
	if Config.DTCount <= 0 {
		return positiveIntFlagError("-c|--dt-count", Config.DTCount)
	}
	syncDurationAndMs(&Config.DTTimeoutDuration, &Config.DTTimeout, 2000)
	if Config.DTTimeoutDuration <= 0 {
		return positiveIntFlagError("-t|--dt-timeout", Config.DTTimeout)
	}
	if !Config.DTHttps {
		if !dtTimeoutChanged && Config.DTTimeout == 5000 && Config.DTTimeoutDuration == 5000*time.Millisecond {
			Config.DTTimeout = 2000
			Config.DTTimeoutDuration = 2000 * time.Millisecond
			Config.SyncDurations()
		}
		if len(Config.HostName) == 0 {
			return fmt.Errorf("%q must not be empty", "--hostname")
		}
		Config.DTSource = DtsSSL
	} else {
		if !dtTimeoutChanged && Config.DTTimeout == 2000 && Config.DTTimeoutDuration == 2000*time.Millisecond {
			Config.DTTimeout = 5000
			Config.DTTimeoutDuration = 5000 * time.Millisecond
			Config.SyncDurations()
		}
		label, _, err := utils.ParseUrl(Config.DTUrl, DefaultDLTUrl)
		if err != nil {
			return fmt.Errorf("failed to derive label from \"--dt-url\": %w", err)
		}
		Config.SuffixLabel = label
		Config.DTSource = DtsHTTPS
	}
	syncDurationAndMs(&Config.DTEvaluationDelayDuration, &Config.DTEvaluationDelay, 600)
	if Config.EnableDTEvaluation {
		if Config.DTEvaluationDelayDuration <= 0 {
			return positiveIntFlagError("-k|--ev-dt-delay", Config.DTEvaluationDelay)
		}
		if Config.DTTimeoutDuration < Config.DTEvaluationDelayDuration {
			logger.Log.Warningf("\"-t|--dt-timeout\" (%v) is less than \"-k|--ev-dt-delay\" (%v); this may cause some tests to fail\n", Config.DTTimeoutDuration, Config.DTEvaluationDelayDuration)
			if !utils.Confirm("Continue?", 3) {
				os.Exit(0)
			}
		}
		if Config.DTStdExp > 0 {
			Config.EnableStdEv = true
		}
	}
	return nil
}

func prepareDownloadTest() error {
	if Config.DLTWorkerThread <= 0 {
		return positiveIntFlagError("-n|--dlt-thread", Config.DLTWorkerThread)
	}
	if Config.DLTCount <= 0 {
		return positiveIntFlagError("-b|--dlt-count", Config.DLTCount)
	}
	syncDurationAndSeconds(&Config.DLTDurationInTotal, &Config.DLTDurMax, 10)
	if Config.DLTDurationInTotal <= 0 {
		return positiveIntFlagError("-d|--dlt-period", Config.DLTDurMax)
	}

	syncDurationAndMs(&Config.HttpRspTimeoutDuration, &Config.DLTTimeout, 5000)
	if Config.HttpRspTimeoutDuration <= 0 {
		return positiveIntFlagError("--dlt-timeout", Config.DLTTimeout)
	}

	if Config.DLTEvaluationSpeed <= 0 {
		return fmt.Errorf("%q must be greater than 0 (got %v)", "-l|--speed", Config.DLTEvaluationSpeed)
	}
	if Config.HttpRspTimeoutDuration > Config.DLTDurationInTotal {
		return fmt.Errorf("%q must be less than or equal to %q (%v > %v)", "--dlt-timeout", "-d|--dlt-period", Config.HttpRspTimeoutDuration, Config.DLTDurationInTotal)
	}
	label, _, err := utils.ParseUrl(Config.DLTUrl, DefaultDLTUrl)
	if err != nil {
		return fmt.Errorf("failed to derive label from \"--dlt-url\": %w", err)
	}
	Config.SuffixLabel = label
	return nil
}

func prepareOutputTargets() {
	if len(Config.ResultFile) > 0 {
		Config.StoreToFile = true
		if !strings.HasSuffix(strings.ToLower(Config.ResultFile), ".csv") {
			Config.ResultFile += ".csv"
		}
	} else {
		Config.ResultFile = "Result_" + utils.GetTimeNowStrSuffix() + "-" + Config.SuffixLabel + ".csv"
	}
	if len(Config.DBFile) > 0 {
		Config.StoreToDB = true
	} else if Config.StoreToDB && len(Config.DBFile) == 0 {
		Config.DBFile = DefaultDBFile
	}
}

func SupplementSourceIPs(level int, tMode int8) error {
	SrcIPs = NewSourceIPsWithRand(MyRand)

	if level == SourceLevelFast {
		logger.Log.Infoln("Supplementing source IPs from --fast ranges...")
		if (tMode & TypeIPv4) == TypeIPv4 {
			tCFIPv4 := CFIPV4
			logger.Log.Debugln("Supplementing IPv4: dynamically fetching optimized active CIDRs...")
			cidrs, err := fetcher.FetchDynamicIPv4(Config.DNSServer, Config.TrancoLimit)
			if err != nil {
				logger.Log.Warningf("Dynamic fetch failed, falling back to fast ranges: %v", err)
				tCFIPv4 = CFIPV4
			} else if len(cidrs) > 0 {
				tCFIPv4 = cidrs
			} else {
				tCFIPv4 = CFIPV4
			}
			if err := SrcIPs.AddFromSlice(tCFIPv4, TypeIPv4); err != nil {
				return err
			}
		}
		if (tMode & TypeIPv6) == TypeIPv6 {
			tCFIPv6 := CFIPV6
			logger.Log.Debugln("Supplementing IPv6: dynamically fetching optimized active CIDRs...")
			cidrs, err := fetcher.FetchDynamicIPv6(Config.DNSServer, Config.TrancoLimit)
			if err != nil {
				logger.Log.Warningf("Dynamic fetch failed, falling back to fast ranges: %v", err)
				tCFIPv6 = CFIPV6
			} else if len(cidrs) > 0 {
				tCFIPv6 = cidrs
			} else {
				tCFIPv6 = CFIPV6
			}
			if err := SrcIPs.AddFromSlice(tCFIPv6, TypeIPv6); err != nil {
				return err
			}
		}
	} else if level == SourceLevelFull {
		logger.Log.Infoln("Supplementing source IPs from full ranges...")
		if (tMode & TypeIPv4) == TypeIPv4 {
			if err := SrcIPs.AddFromSlice(CFIPV4FULL, TypeIPv4); err != nil {
				return err
			}
		}
		if (tMode & TypeIPv6) == TypeIPv6 {
			if err := SrcIPs.AddFromSlice(CFIPV6FULL, TypeIPv6); err != nil {
				return err
			}
		}
	}

	SrcIPs.Shuffle()
	if err := SrcIPs.AddPorts(Config.PortStrSlice); err != nil {
		return err
	}
	return nil
}

// HasUserSources returns true if user specified candidate IPs via -s or -i.
func HasUserSources() bool {
	return len(IPStr) > 0 || len(Config.IPFiles) > 0 || len(Config.IPFile) > 0
}

// SupplementIPMode returns the IP type bitmask (TypeIPv4, TypeIPv6, or both) for supplementation fallback.
func SupplementIPMode() int8 {
	var sMode int8
	if Config.SupplementIPv4 {
		sMode |= TypeIPv4
	}
	if Config.SupplementIPv6 {
		sMode |= TypeIPv6
	}
	return sMode
}

