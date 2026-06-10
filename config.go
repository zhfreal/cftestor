package main

import (
	"fmt"
	"math/big"
	"net/url"
	"regexp"
	"strings"
	"time"

	utls "github.com/refraction-networking/utls"
	flag "github.com/spf13/pflag"
)

type cliOptions struct {
	Config           AppConfig
	IPs              []string
	SourceIPs        []string
	Mark             string
	XMark            string
	PrintVersion     bool
	TLSHelloFirefox  bool
	TLSHelloChrome   bool
	TLSHelloEdge     bool
	TLSHelloSafari   bool
	IPv4Changed      bool
	IPv6Changed      bool
	DTTimeoutChanged bool
	MarkChanged      bool
	XMarkChanged     bool
}

func DefaultConfig() AppConfig {
	return AppConfig{
		DTCount:                     4,
		DTWorkerThread:              20,
		DLTDurMax:                   10,
		DLTWorkerThread:             1,
		DLTCount:                    1,
		ResultMin:                   10,
		Interval:                    500,
		DTEvaluationDelay:           600,
		DTTimeout:                   2000,
		DTStdExp:                    30,
		HostName:                    DefaultTestHost,
		DLTUrl:                      defaultDLTUrl,
		DTUrl:                       defaultDTUrl,
		DLTTimeout:                  5000,
		Loop:                        -1,
		TestTimeout:                 30,
		LoopInterval:                60,
		DTEvaluationDTPR:            100,
		DLTEvaluationSpeed:          6000,
		DTVia:                       "https",
		DTHttpRspReturnCodeExpected: 200,
		IPv4Mode:                    true,
		TLSClientID:                 utls.HelloChrome_Auto,
		UserAgent:                   userAgentChrome,
		PortStrSlice:                []string{},
		DNSServer:                   "1.1.1.1:53",
	}
}

func parseCLI(args []string) (cliOptions, error) {
	opts := cliOptions{
		Config:    DefaultConfig(),
		IPs:       []string{},
		SourceIPs: []string{},
	}
	fs := flag.NewFlagSet(runTime, flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Print(help)
	}
	registerCLIFlags(fs, &opts)
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	opts.IPs = append(opts.IPs, opts.SourceIPs...)
	opts.IPv4Changed = flagChanged(fs, "ipv4")
	opts.IPv6Changed = flagChanged(fs, "ipv6")
	opts.DTTimeoutChanged = flagChanged(fs, "dt-timeout", "dt-timeout-ms")
	opts.MarkChanged = flagChanged(fs, "mark")
	opts.XMarkChanged = flagChanged(fs, "xmark")
	applyTLSFingerprint(&opts)
	return opts, nil
}

func registerCLIFlags(fs *flag.FlagSet, opts *cliOptions) {
	cfg := &opts.Config

	fs.BoolVar(&cfg.FastMode, "fast", cfg.FastMode, "Use a limited set of internal Cloudflare IPs for quick scanning.")
	fs.StringSliceVarP(&opts.IPs, "ip", "s", opts.IPs, "IP, CIDR, or host:port candidate to test. Can be provided multiple times.")
	fs.StringSliceVar(&opts.SourceIPs, "source", opts.SourceIPs, "Alias for --ip.")
	fs.StringVarP(&cfg.IPFile, "in", "i", cfg.IPFile, "Path to a file containing IPs, CIDRs, or host:port entries.")
	fs.StringVar(&cfg.IPFile, "source-file", cfg.IPFile, "Alias for --in.")

	fs.IntVarP(&cfg.DTWorkerThread, "dt-thread", "m", cfg.DTWorkerThread, "Number of concurrent Delay Test (DT) workers.")
	fs.IntVar(&cfg.DTWorkerThread, "dt-workers", cfg.DTWorkerThread, "Alias for --dt-thread.")
	fs.IntVarP(&cfg.DTTimeout, "dt-timeout", "t", cfg.DTTimeout, "Timeout for a single DT attempt in milliseconds.")
	fs.IntVar(&cfg.DTTimeout, "dt-timeout-ms", cfg.DTTimeout, "Alias for --dt-timeout.")
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
	fs.IntVarP(&cfg.DLTDurMax, "dlt-period", "d", cfg.DLTDurMax, "Maximum duration for one DLT attempt in seconds.")
	fs.IntVar(&cfg.DLTDurMax, "dlt-duration", cfg.DLTDurMax, "Alias for --dlt-period.")
	fs.IntVarP(&cfg.DLTCount, "dlt-count", "b", cfg.DLTCount, "Number of DLT attempts per candidate.")
	fs.IntVar(&cfg.DLTCount, "dlt-attempts", cfg.DLTCount, "Alias for --dlt-count.")
	fs.StringVarP(&cfg.DLTUrl, "dlt-url", "u", cfg.DLTUrl, "URL to use for DLT.")
	fs.IntVar(&cfg.DLTTimeout, "dlt-timeout", cfg.DLTTimeout, "HTTP response timeout for DLT in milliseconds.")
	fs.IntVar(&cfg.DLTTimeout, "dlt-timeout-ms", cfg.DLTTimeout, "Alias for --dlt-timeout.")
	fs.IntVarP(&cfg.Interval, "interval", "I", cfg.Interval, "Interval between test attempts in milliseconds.")
	fs.IntVar(&cfg.Interval, "test-interval-ms", cfg.Interval, "Alias for --interval.")

	fs.BoolVar(&cfg.EnableDTEvaluation, "ev-dt", cfg.EnableDTEvaluation, "Enable DT evaluation using all attempts.")
	fs.BoolVar(&cfg.EnableDTEvaluation, "dt-evaluate", cfg.EnableDTEvaluation, "Alias for --ev-dt.")
	fs.IntVarP(&cfg.DTEvaluationDelay, "ev-dt-delay", "k", cfg.DTEvaluationDelay, "Maximum allowed average DT delay in milliseconds.")
	fs.IntVar(&cfg.DTEvaluationDelay, "dt-max-delay", cfg.DTEvaluationDelay, "Alias for --ev-dt-delay.")
	fs.Float64Var(&cfg.DTEvaluationDTPR, "ev-dt-dtpr", cfg.DTEvaluationDTPR, "Minimum required DT pass rate percentage.")
	fs.Float64Var(&cfg.DTEvaluationDTPR, "dt-min-pass-rate", cfg.DTEvaluationDTPR, "Alias for --ev-dt-dtpr.")
	fs.Float64Var(&cfg.DTStdExp, "ev-dt-std", cfg.DTStdExp, "Maximum allowed DT standard deviation when enabled.")
	fs.Float64Var(&cfg.DTStdExp, "dt-max-stddev", cfg.DTStdExp, "Alias for --ev-dt-std.")
	fs.Float64VarP(&cfg.DLTEvaluationSpeed, "speed", "l", cfg.DLTEvaluationSpeed, "Minimum required download speed in KB/s.")
	fs.Float64Var(&cfg.DLTEvaluationSpeed, "min-speed", cfg.DLTEvaluationSpeed, "Alias for --speed.")
	fs.IntVar(&cfg.Loop, "loop", cfg.Loop, "Retest qualified candidates for N confirmation cycles; refill from the original pool if fewer than --result remain.")
	fs.IntVar(&cfg.LoopInterval, "loop-interval", cfg.LoopInterval, "Seconds to wait between loop cycles.")
	fs.IntVarP(&cfg.ResultMin, "result", "r", cfg.ResultMin, "Target number of final qualified results.")
	fs.IntVar(&cfg.ResultMin, "result-count", cfg.ResultMin, "Alias for --result.")

	fs.BoolVar(&cfg.DisableDownload, "disable-download", cfg.DisableDownload, "Deprecated, use --dt-only instead.")
	fs.BoolVar(&cfg.DTOnly, "dt-only", cfg.DTOnly, "Perform Delay Test only.")
	fs.BoolVar(&cfg.DLTOnly, "dlt-only", cfg.DLTOnly, "Perform Download Test only.")
	fs.BoolVarP(&cfg.IPv4Mode, "ipv4", "4", cfg.IPv4Mode, "Test IPv4 only.")
	fs.BoolVarP(&cfg.IPv6Mode, "ipv6", "6", cfg.IPv6Mode, "Test IPv6 only.")
	fs.StringVar(&cfg.FetchIPv6File, "fetch-ipv6", cfg.FetchIPv6File, "Fetch active Cloudflare IPv6 CIDRs dynamically, save to file, and exit.")
	fs.StringVar(&cfg.DNSServer, "dns", cfg.DNSServer, "Custom DNS server for dynamic fetching (e.g. 1.1.1.1:53, tls://1.1.1.1, https://1.1.1.1/dns-query)")
	fs.BoolVarP(&cfg.TestAll, "test-all", "a", cfg.TestAll, "Test all IPs until no more IP left.")
	fs.StringVar(&opts.Mark, "mark", opts.Mark, "Set Linux socket fwmark for outbound packets. Supports decimal and hex.")
	fs.StringVar(&opts.XMark, "xmark", opts.XMark, "Alias for --mark.")
	fs.StringVar(&cfg.OutboundInterface, "interface", cfg.OutboundInterface, "Bind outbound packets to an interface name, interface index, or local source IP.")
	fs.BoolVar(&opts.TLSHelloFirefox, "hello-firefox", opts.TLSHelloFirefox, "Simulate Firefox TLS fingerprint.")
	fs.BoolVar(&opts.TLSHelloChrome, "hello-chrome", opts.TLSHelloChrome, "Simulate Chrome TLS fingerprint.")
	fs.BoolVar(&opts.TLSHelloEdge, "hello-edge", opts.TLSHelloEdge, "Simulate Edge TLS fingerprint.")
	fs.BoolVar(&opts.TLSHelloSafari, "hello-safari", opts.TLSHelloSafari, "Simulate Safari TLS fingerprint.")
	fs.IntVar(&cfg.TestTimeout, "test-timeout", cfg.TestTimeout, "Test timeout in minutes.")

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
	fs.BoolVarP(&cfg.NoCache, "no-cache", "C", cfg.NoCache, "Bypass CDN/proxy caching for custom URLs.")

	fs.BoolVarP(&cfg.SilenceMode, "silence", "S", cfg.SilenceMode, "Enable silence mode with minimal output.")
	fs.BoolVar(&cfg.SilenceMode, "quiet", cfg.SilenceMode, "Alias for --silence.")
	fs.BoolVarP(&cfg.Debug, "debug", "V", cfg.Debug, "Print debug message.")
	fs.BoolVarP(&opts.PrintVersion, "version", "v", opts.PrintVersion, "Show version.")
}

func flagChanged(fs *flag.FlagSet, names ...string) bool {
	for _, name := range names {
		f := fs.Lookup(name)
		if f != nil && f.Changed {
			return true
		}
	}
	return false
}

func applyTLSFingerprint(opts *cliOptions) {
	cfg := &opts.Config
	if opts.TLSHelloFirefox {
		cfg.TLSClientID = utls.HelloFirefox_Auto
		cfg.UserAgent = userAgentFirefox
	}
	if opts.TLSHelloChrome {
		cfg.TLSClientID = utls.HelloChrome_Auto
		cfg.UserAgent = userAgentChrome
	}
	if opts.TLSHelloEdge {
		cfg.TLSClientID = utls.HelloEdge_Auto
		cfg.UserAgent = userAgentEdge
	}
	if opts.TLSHelloSafari {
		cfg.TLSClientID = utls.HelloSafari_Auto
		cfg.UserAgent = userAgentSafari
	}
}

func configureApp(args []string) (bool, int, error) {
	opts, err := parseCLI(args)
	if err != nil {
		if err == flag.ErrHelp {
			return true, 0, nil
		}
		return true, 2, err
	}

	Config = opts.Config
	ipStr = opts.IPs
	resetRuntimeState()

	if len(version) == 0 {
		version = "dev"
	}
	if !Config.SilenceMode {
		print_version()
	} else {
		Config.Debug = false
		Config.StoreToDB = false
		Config.StoreToFile = false
	}
	if opts.PrintVersion {
		return true, 0, nil
	}

	initLoggerFromConfig()
	initRandSeed()
	return prepareRuntime(&opts)
}

func resetRuntimeState() {
	verifyResultsMap = make(map[string]VerifyResults)
	myRand = newRand()
	srcIPs = NewSourceIPsWithRand(myRand)
}

func initLoggerFromConfig() {
	if Config.SilenceMode {
		loggerLevel = logLevelFatal
	} else {
		loggerLevel = logLevelInfo
		if Config.Debug {
			loggerLevel = logLevelDebug
		}
	}
	myLogger = myLogger.newLogger(loggerLevel)
}

func prepareRuntime(opts *cliOptions) (bool, int, error) {
	if Config.DisableDownload {
		Config.DTOnly = true
		myLogger.Warningln("deprecated flag \"--disable-download\"; use \"--dt-only\" instead")
	}
	if Config.DTHttps {
		Config.DTVia = "https"
		myLogger.Warningln("deprecated flag \"--dt-via-https\"; use \"--dt-via https\" instead")
	}
	if Config.DTOnly && Config.DLTOnly {
		return true, 1, fmt.Errorf("%q and %q cannot be provided at the same time", "--dt-only", "--dlt-only")
	}
	if Config.DTEvaluationDTPR > 100 {
		Config.DTEvaluationDTPR = 100
	} else if Config.DTEvaluationDTPR < 0 {
		Config.DTEvaluationDTPR = 0
	}
	if err := normalizeDTVia(); err != nil {
		return true, 1, err
	}

	tMode, err := selectedIPMode(opts.IPv4Changed)
	if err != nil {
		return true, 1, err
	}
	trimConfigStrings()
	if err := prepareOutboundOptions(opts); err != nil {
		return true, 1, err
	}

	if len(Config.FetchIPv6File) > 0 {
		myLogger.Infof("Starting dynamic IPv6 fetch...")
		cidrs, err := FetchDynamicIPv6(Config.DNSServer)
		if err != nil {
			myLogger.Errorf("Fetch failed: %v", err)
			return true, 1, err
		}
		if err := writeStringsToFile(Config.FetchIPv6File, cidrs); err != nil {
			myLogger.Errorf("Failed to save fetched IPs: %v", err)
			return true, 1, err
		}
		myLogger.Infof("Successfully saved %d IPv6 CIDRs to %s", len(cidrs), Config.FetchIPv6File)
		return true, 0, nil
	}

	if err := loadSourceIPs(tMode, opts.IPv4Changed, opts.IPv6Changed); err != nil {
		return true, 1, err
	}
	if err := validateURLs(); err != nil {
		return true, 1, err
	}
	if err := prepareSourcePool(); err != nil {
		return true, 1, err
	}
	shouldExit, code, err := prepareTestModes(opts.DTTimeoutChanged)
	if shouldExit || err != nil {
		return shouldExit, code, err
	}
	prepareOutputTargets()
	return false, 0, nil
}

func normalizeDTVia() error {
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

func selectedIPMode(ipv4Changed bool) (int8, error) {
	if !ipv4Changed && Config.IPv6Mode {
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
	Config.ResultFile = strings.TrimSpace(Config.ResultFile)
	Config.SuffixLabel = strings.TrimSpace(Config.SuffixLabel)
	Config.HostName = strings.TrimSpace(Config.HostName)
	Config.DTUrl = strings.TrimSpace(Config.DTUrl)
	Config.DLTUrl = strings.TrimSpace(Config.DLTUrl)
	Config.DBFile = strings.TrimSpace(Config.DBFile)
	Config.OutboundInterface = strings.TrimSpace(Config.OutboundInterface)
}

func loadSourceIPs(tMode int8, ipv4Changed, ipv6Changed bool) error {
	hasUserSources := len(ipStr) > 0 || len(Config.IPFile) > 0
	if !hasUserSources {
		if (tMode&TypeIPv4) == TypeIPv4 && (tMode&TypeIPv6) == TypeIPv6 {
			return fmt.Errorf("the options \"-4|--ipv4\" and \"-6|--ipv6\" cannot be used together when no specific IPs or file are provided")
		}
		if (tMode & TypeIPv4) == TypeIPv4 {
			tCFIPv4 := CFIPV4FULL
			if Config.FastMode {
				tCFIPv4 = CFIPV4
			}
			return srcIPs.AddFromSlice(tCFIPv4, TypeIPv4)
		}
		tCFIPv6 := CFIPV6FULL
		if Config.FastMode {
			myLogger.Infoln("Fast mode enabled for IPv6: dynamically fetching optimized active IPv6 CIDRs...")
			cidrs, err := FetchDynamicIPv6(Config.DNSServer)
			if err != nil {
				myLogger.Warningf("Dynamic fetch failed, falling back to full ranges: %v", err)
				tCFIPv6 = CFIPV6FULL
			} else if len(cidrs) > 0 {
				tCFIPv6 = cidrs
			}
		}
		return srcIPs.AddFromSlice(tCFIPv6, TypeIPv6)
	}

	if !ipv6Changed && !ipv4Changed {
		Config.IPv6Mode = true
		tMode = TypeIPv4 | TypeIPv6
	}
	if len(ipStr) > 0 {
		if err := srcIPs.AddFromSlice(ipStr, tMode); err != nil {
			return err
		}
	}
	if len(Config.IPFile) != 0 {
		if err := srcIPs.AddFromFile(Config.IPFile, tMode); err != nil {
			return err
		}
	}
	if srcIPs.LenInt() == 0 {
		return fmt.Errorf("no source IPs provided")
	}
	return nil
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
	if Config.Interval <= 0 {
		return positiveIntFlagError("-I|--interval", Config.Interval)
	}
	if Config.ResultMin <= 0 {
		return positiveIntFlagError("-r|--result", Config.ResultMin)
	}
	srcIPs.Shuffle()
	if err := srcIPs.AddPorts(Config.PortStrSlice); err != nil {
		return err
	}
	tQty := srcIPs.Len()
	if Config.TestAll {
		Config.ResultMin = -1
	} else {
		tResultMin := big.NewInt(int64(Config.ResultMin))
		if tQty.Cmp(tResultMin) == -1 {
			Config.ResultMin = int(tQty.Int64())
		}
	}
	if len(Config.SuffixLabel) == 0 {
		Config.SuffixLabel = Config.HostName
	}
	return nil
}

func prepareTestModes(dtTimeoutChanged bool) (bool, int, error) {
	if !Config.DLTOnly {
		shouldExit, code, err := prepareDelayTest(dtTimeoutChanged)
		if shouldExit || err != nil {
			return shouldExit, code, err
		}
	}
	if !Config.DTOnly {
		if err := prepareDownloadTest(); err != nil {
			return true, 1, err
		}
	}
	return false, 0, nil
}

func prepareDelayTest(dtTimeoutChanged bool) (bool, int, error) {
	if Config.DTWorkerThread <= 0 {
		return true, 1, positiveIntFlagError("-m|--dt-thread", Config.DTWorkerThread)
	}
	if Config.DTCount <= 0 {
		return true, 1, positiveIntFlagError("-c|--dt-count", Config.DTCount)
	}
	if Config.DTTimeout <= 0 {
		return true, 1, positiveIntFlagError("-t|--dt-timeout", Config.DTTimeout)
	}
	if !Config.DTHttps {
		if len(Config.HostName) == 0 {
			return true, 1, fmt.Errorf("%q must not be empty", "--hostname")
		}
		Config.DTSource = dtsSSL
	} else {
		if !dtTimeoutChanged {
			Config.DTTimeout = 5000
		}
		label, _, err := parseUrl(Config.DTUrl)
		if err != nil {
			return true, 1, fmt.Errorf("failed to derive label from \"--dt-url\": %w", err)
		}
		Config.SuffixLabel = label
		Config.DTSource = dtsHTTPS
	}
	if Config.EnableDTEvaluation {
		if Config.DTEvaluationDelay <= 0 {
			return true, 1, positiveIntFlagError("-k|--ev-dt-delay", Config.DTEvaluationDelay)
		}
		if Config.DTTimeout < Config.DTEvaluationDelay {
			myLogger.Warningf("\"-t|--dt-timeout\" (%d ms) is less than \"-k|--ev-dt-delay\" (%d ms); this may cause some tests to fail\n", Config.DTTimeout, Config.DTEvaluationDelay)
			if !confirm("Continue?", 3) {
				return true, 0, nil
			}
		}
		if Config.DTStdExp > 0 {
			Config.EnableStdEv = true
		}
	}
	Config.DTTimeoutDuration = time.Duration(Config.DTTimeout) * time.Millisecond
	return false, 0, nil
}

func prepareDownloadTest() error {
	if Config.DLTWorkerThread <= 0 {
		return positiveIntFlagError("-n|--dlt-thread", Config.DLTWorkerThread)
	}
	if Config.DLTCount <= 0 {
		return positiveIntFlagError("-b|--dlt-count", Config.DLTCount)
	}
	if Config.DLTDurMax <= 0 {
		return positiveIntFlagError("-d|--dlt-period", Config.DLTDurMax)
	}
	if Config.DLTEvaluationSpeed <= 0 {
		return positiveFloatFlagError("-l|--speed", Config.DLTEvaluationSpeed)
	}
	if Config.DLTTimeout > Config.DLTDurMax*1000 {
		return fmt.Errorf("%q must be less than or equal to %q (%d ms > %d ms)", "--dlt-timeout", "-d|--dlt-period", Config.DLTTimeout, Config.DLTDurMax*1000)
	}
	label, _, err := parseUrl(Config.DLTUrl)
	if err != nil {
		return fmt.Errorf("failed to derive label from \"--dlt-url\": %w", err)
	}
	Config.SuffixLabel = label
	Config.HttpRspTimeoutDuration = time.Duration(Config.DLTTimeout) * time.Millisecond
	Config.DLTDurationInTotal = time.Duration(Config.DLTDurMax) * time.Second
	return nil
}

func prepareOutputTargets() {
	if len(Config.ResultFile) > 0 {
		Config.StoreToFile = true
		re := regexp.MustCompile(`.[c|C][s|S][V|v]$`)
		if !re.Match([]byte(Config.ResultFile)) {
			Config.ResultFile = Config.ResultFile + ".csv"
		}
	} else {
		Config.ResultFile = "Result_" + getTimeNowStrSuffix() + "-" + Config.SuffixLabel + ".csv"
	}
	if len(Config.DBFile) > 0 {
		Config.StoreToDB = true
	} else if Config.StoreToDB && len(Config.DBFile) == 0 {
		Config.DBFile = defaultDBFile
	}
}

func positiveIntFlagError(flagName string, value int) error {
	return fmt.Errorf("%q must be greater than 0 (got %d)", flagName, value)
}

func positiveFloatFlagError(flagName string, value float64) error {
	return fmt.Errorf("%q must be greater than 0 (got %v)", flagName, value)
}
