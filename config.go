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
	PrintVersion     bool
	TLSHelloFirefox  bool
	TLSHelloChrome   bool
	TLSHelloEdge     bool
	TLSHelloSafari   bool
	IPv4Changed      bool
	IPv6Changed      bool
	DTTimeoutChanged bool
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
	}
}

func parseCLI(args []string) (cliOptions, error) {
	opts := cliOptions{
		Config: DefaultConfig(),
		IPs:    []string{},
	}
	fs := flag.NewFlagSet(runTime, flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Print(help)
	}
	registerCLIFlags(fs, &opts)
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	opts.IPv4Changed = fs.Lookup("ipv4").Changed
	opts.IPv6Changed = fs.Lookup("ipv6").Changed
	opts.DTTimeoutChanged = fs.Lookup("dt-timeout").Changed
	applyTLSFingerprint(&opts)
	return opts, nil
}

func registerCLIFlags(fs *flag.FlagSet, opts *cliOptions) {
	cfg := &opts.Config

	fs.BoolVar(&cfg.FastMode, "fast", cfg.FastMode, "Fast mode")
	fs.StringSliceVarP(&opts.IPs, "ip", "s", opts.IPs, "Specific IP, CIDR, or host:port for test.")
	fs.StringVarP(&cfg.IPFile, "in", "i", cfg.IPFile, "Specific file of IPs, CIDRs, or host:port entries for test.")

	fs.IntVarP(&cfg.DTWorkerThread, "dt-thread", "m", cfg.DTWorkerThread, "Number of concurrent threads for Delay Test(DT).")
	fs.IntVarP(&cfg.DTTimeout, "dt-timeout", "t", cfg.DTTimeout, "Timeout for single DT(ms).")
	fs.IntVarP(&cfg.DTCount, "dt-count", "c", cfg.DTCount, "Tries of DT for a IP.")
	fs.StringSliceVarP(&cfg.PortStrSlice, "port", "p", cfg.PortStrSlice, "Port to test, could be specific one or more ports at same time.")
	fs.StringVar(&cfg.HostName, "hostname", cfg.HostName, "Hostname for DT test.")
	fs.StringVar(&cfg.DTVia, "dt-via", cfg.DTVia, "DT via https rather than SSL/TLS shaking hands.")
	fs.IntVar(&cfg.DTHttpRspReturnCodeExpected, "dt-expect-code", cfg.DTHttpRspReturnCodeExpected, "HTTP status code expected for DT test.")
	fs.BoolVar(&cfg.DTHttps, "dt-via-https", cfg.DTHttps, "DT via https rather than SSL/TLS shaking hands.")
	fs.StringVar(&cfg.DTUrl, "dt-url", cfg.DTUrl, "Specific the url while DT via https.")

	fs.IntVarP(&cfg.DLTWorkerThread, "dlt-thread", "n", cfg.DLTWorkerThread, "Number of concurrent Threads for Download Test(DLT).")
	fs.IntVarP(&cfg.DLTDurMax, "dlt-period", "d", cfg.DLTDurMax, "The total times escaped for single DLT in seconds, default 10s.")
	fs.IntVarP(&cfg.DLTCount, "dlt-count", "b", cfg.DLTCount, "Tries of DLT for a IP, default 1.")
	fs.StringVarP(&cfg.DLTUrl, "dlt-url", "u", cfg.DLTUrl, "Customize test URL for DLT.")
	fs.IntVar(&cfg.DLTTimeout, "dlt-timeout", cfg.DLTTimeout, "Specify the timeout for http response when do DLT in milliseconds, default 5000 ms.")
	fs.IntVarP(&cfg.Interval, "interval", "I", cfg.Interval, "Interval between two tests, unit ms, default 500ms.")

	fs.BoolVar(&cfg.EnableDTEvaluation, "ev-dt", cfg.EnableDTEvaluation, "Evaluate DT test result. Default as disabled")
	fs.IntVarP(&cfg.DTEvaluationDelay, "ev-dt-delay", "k", cfg.DTEvaluationDelay, "Delay for DT is beyond this one will be cause failure, unit ms, default 600ms.")
	fs.Float64Var(&cfg.DTEvaluationDTPR, "ev-dt-dtpr", cfg.DTEvaluationDTPR, "The DT successful rate below this will be cause failure, default 100%.")
	fs.Float64Var(&cfg.DTStdExp, "ev-dt-std", cfg.DTStdExp, "expect standard deviation while do DT evaluation.")
	fs.Float64VarP(&cfg.DLTEvaluationSpeed, "speed", "l", cfg.DLTEvaluationSpeed, "Download speed should not less than this, Unit KB/s, default 6000KB/s.")
	fs.IntVar(&cfg.Loop, "loop", cfg.Loop, "Retest qualified candidates for this many confirmation cycles.")
	fs.IntVar(&cfg.LoopInterval, "loop-interval", cfg.LoopInterval, "sleep N second between two loop")
	fs.IntVarP(&cfg.ResultMin, "result", "r", cfg.ResultMin, "The total IPs qualified limitation, default 10")

	fs.BoolVar(&cfg.DisableDownload, "disable-download", cfg.DisableDownload, "Deprecated, use --dt-only instead.")
	fs.BoolVar(&cfg.DTOnly, "dt-only", cfg.DTOnly, "Do DT only, we do DT & DLT at the same time by default.")
	fs.BoolVar(&cfg.DLTOnly, "dlt-only", cfg.DLTOnly, "Do DLT only, we do DT & DLT at the same time by default.")
	fs.BoolVarP(&cfg.IPv4Mode, "ipv4", "4", cfg.IPv4Mode, "Just test IPv4.")
	fs.BoolVarP(&cfg.IPv6Mode, "ipv6", "6", cfg.IPv6Mode, "Just test IPv6.")
	fs.BoolVarP(&cfg.TestAll, "test-all", "a", cfg.TestAll, "Test all IPs until no more IP left.")
	fs.BoolVar(&opts.TLSHelloFirefox, "hello-firefox", opts.TLSHelloFirefox, "work as firefox")
	fs.BoolVar(&opts.TLSHelloChrome, "hello-chrome", opts.TLSHelloChrome, "work as chrome")
	fs.BoolVar(&opts.TLSHelloEdge, "hello-edge", opts.TLSHelloEdge, "work as edge")
	fs.BoolVar(&opts.TLSHelloSafari, "hello-safari", opts.TLSHelloSafari, "work as safari")
	fs.IntVar(&cfg.TestTimeout, "test-timeout", cfg.TestTimeout, "Test timeout in minutes.")

	fs.BoolVarP(&cfg.StoreToFile, "to-file", "w", cfg.StoreToFile, "Write result to csv file, disabled by default.")
	fs.StringVarP(&cfg.ResultFile, "out-file", "o", cfg.ResultFile, "File name of result. ")
	fs.BoolVarP(&cfg.StoreToDB, "to-db", "e", cfg.StoreToDB, "Write result to sqlite3 db file.")
	fs.BoolVar(&cfg.ResolveLocalASNAndCity, "local-asn", cfg.ResolveLocalASNAndCity, "get local asn and city info")
	fs.StringVarP(&cfg.DBFile, "db-file", "f", cfg.DBFile, "Sqlite3 db file name.")
	fs.StringVarP(&cfg.SuffixLabel, "label", "g", cfg.SuffixLabel, "the label for a part of the result file's name and sqlite3 record.")
	fs.BoolVar(&cfg.ResolveLoc, "resolve-loc", cfg.ResolveLoc, "try to resolve location.")
	fs.BoolVarP(&cfg.NoCache, "no-cache", "C", cfg.NoCache, "disable cdn/proxy caching")

	fs.BoolVarP(&cfg.SilenceMode, "silence", "S", cfg.SilenceMode, "silence mode.")
	fs.BoolVarP(&cfg.Debug, "debug", "V", cfg.Debug, "Print debug message.")
	fs.BoolVarP(&opts.PrintVersion, "version", "v", opts.PrintVersion, "Show version.")
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
		fmt.Println("Warning! \"--disable-download\" is deprecated, use \"--dt-only\" instead!")
	}
	if Config.DTHttps {
		Config.DTVia = "https"
		fmt.Println("Warning! \"--dt-via-https\" is deprecated, use \"--dt-via https|tls|ssl\" instead!")
	}
	if Config.DTOnly && Config.DLTOnly {
		return true, 1, fmt.Errorf("\"--dt-only\" and \"--dlt-only\" should not be provided at the same time")
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
	switch strings.ToLower(Config.DTVia) {
	case "https":
		Config.DTHttps = true
	case "ssl", "tls":
		Config.DTHttps = false
	default:
		return fmt.Errorf("invalid value for \"--dt-via\". Please use one of: https, tls, or ssl")
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
		return tMode, fmt.Errorf("we can't disable both IPv4 and IPv6 at the same time")
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
			tCFIPv6 = CFIPV6
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
		return "", fmt.Errorf("%q should not be empty", flagName)
	}
	tURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("%q %v is not a valid URL", flagName, urlStr)
	}
	if tURL.Scheme != "https" {
		return "", fmt.Errorf("%q %v should use HTTPS", flagName, urlStr)
	}
	return tURL.String(), nil
}

func prepareSourcePool() error {
	if Config.Interval <= 0 {
		return fmt.Errorf("\"-I|--interval %v\" should not be smaller than 0", Config.Interval)
	}
	if Config.ResultMin <= 0 {
		return fmt.Errorf("\"-r|--result %v\" should not be smaller than 0", Config.ResultMin)
	}
	srcIPs.Shuffle()
	srcIPs.AddPorts(Config.PortStrSlice)
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
		return true, 1, fmt.Errorf("\"-m|--dt-thread %v\" should not be smaller than 0", Config.DTWorkerThread)
	}
	if Config.DTCount <= 0 {
		return true, 1, fmt.Errorf("\"-c|--dt-count %v\" should not be smaller than 0", Config.DTCount)
	}
	if Config.DTTimeout <= 0 {
		return true, 1, fmt.Errorf("\"-t|--dt-timeout %v\" should not be smaller than 0", Config.DTTimeout)
	}
	if !Config.DTHttps {
		if len(Config.HostName) == 0 {
			return true, 1, fmt.Errorf("\"--hostname\" should not be empty")
		}
		Config.DTSource = dtsSSL
	} else {
		if !dtTimeoutChanged {
			Config.DTTimeout = 5000
		}
		Config.SuffixLabel, _ = parseUrl(Config.DTUrl)
		Config.DTSource = dtsHTTPS
	}
	if Config.EnableDTEvaluation {
		if Config.DTEvaluationDelay <= 0 {
			return true, 1, fmt.Errorf("\"-k|--evaluate-dt-delay %v\" should not be smaller than 0", Config.DTEvaluationDelay)
		}
		if Config.DTTimeout < Config.DTEvaluationDelay {
			myLogger.Warning(fmt.Sprintf("\"-t|--dt-timeout\" - %v is less than \"-k|--evaluate-dt-delay\" - %v. This will led to failure for some test!", Config.DTTimeout, Config.DTEvaluationDelay))
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
		return fmt.Errorf("\"-n|--dlt-thread %v\" should not be smaller than 0", Config.DLTWorkerThread)
	}
	if Config.DLTCount <= 0 {
		return fmt.Errorf("\"-b|--dlt-count %v\" should not be smaller than 0", Config.DLTCount)
	}
	if Config.DLTDurMax <= 0 {
		return fmt.Errorf("\"-d|--dlt-period %v\" should not be smaller than 0", Config.DLTDurMax)
	}
	if Config.DLTEvaluationSpeed <= 0 {
		return fmt.Errorf("\"-l|--speed %v\" should not be smaller than 0", Config.DLTEvaluationSpeed)
	}
	if Config.DLTTimeout > Config.DLTDurMax*1000 {
		return fmt.Errorf("\"<--dlt-timeout> %v\" should not be bigger than <-d|--dlt-period> %v", Config.DLTTimeout, Config.DLTDurMax)
	}
	Config.SuffixLabel, _ = parseUrl(Config.DLTUrl)
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
