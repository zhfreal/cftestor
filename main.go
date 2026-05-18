package main

import (
	"fmt"
	"math/big"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	utls "github.com/refraction-networking/utls"
	flag "github.com/spf13/pflag"
)

func print_version() {
	fmt.Println(appArt)
	fmt.Println(`  CF CDN IP scanner, find best IPs for you.
  https://github.com/zhfreal/cftestor`)
	fmt.Println()
	fmt.Printf("Version:    %v\n", version)
	fmt.Printf("BuildDate:  %v\n", buildDate)
	fmt.Printf("BuildTag:   %v\n", buildTag)
	fmt.Printf("BuildHash:  %v\n", buildHash)
	fmt.Println()
}

func init() {
	var printVersion bool
	var tlsHelloFirefox, tlsHelloChrome, tlsHelloEdge, tlsHelloSafari bool = false, false, false, false

	// version = "dev"
	flag.BoolVar(&Config.FastMode, "fast", false, "Fast mode")
	flag.StringSliceVarP(&ipStr, "ip", "s", []string{}, "Specific IP or CIDR for test.")
	flag.StringVarP(&Config.IPFile, "in", "i", "", "Specific file of IPs and CIDRs for test.")

	flag.IntVarP(&Config.DTWorkerThread, "dt-thread", "m", 20, "Number of concurrent threads for Delay Test(DT).")
	flag.IntVarP(&Config.DTTimeout, "dt-timeout", "t", 2000, "Timeout for single DT(ms).")
	flag.IntVarP(&Config.DTCount, "dt-count", "c", 4, "Tries of DT for a IP.")
	// flag.IntVarP(&port, "port", "p", 443, "Port to test")
	flag.StringSliceVarP(&Config.PortStrSlice, "port", "p", []string{}, "Port to test, could be specific one or more ports at same time.")
	flag.StringVar(&Config.HostName, "hostname", DefaultTestHost, "Hostname for DT test.")
	flag.StringVar(&Config.DTVia, "dt-via", "https", "DT via https rather than SSL/TLS shaking hands.")
	flag.IntVar(&Config.DTHttpRspReturnCodeExpected, "dt-expect-code", 200, "HTTP status code expected for DT test.")
	flag.BoolVar(&Config.DTHttps, "dt-via-https", false, "DT via https rather than SSL/TLS shaking hands.")
	flag.StringVar(&Config.DTUrl, "dt-url", defaultDTUrl, "Specific the url while DT via https.")

	flag.IntVarP(&Config.DLTWorkerThread, "dlt-thread", "n", 1, "Number of concurrent Threads for Download Test(DLT).")
	flag.IntVarP(&Config.DLTDurMax, "dlt-period", "d", 10, "The total times escaped for single DLT in seconds, default 10s.")
	flag.IntVarP(&Config.DLTCount, "dlt-count", "b", 1, "Tries of DLT for a IP, default 1.")
	flag.StringVarP(&Config.DLTUrl, "dlt-url", "u", defaultDLTUrl, "Customize test URL for DLT.")
	flag.IntVar(&Config.DLTTimeout, "dlt-timeout", 5000, "Specify the timeout for http response when do DLT in milliseconds, default 5000 ms.")
	flag.IntVarP(&Config.Interval, "interval", "I", 500, "Interval between two tests, unit ms, default 500ms.")

	flag.BoolVar(&Config.EnableDTEvaluation, "ev-dt", false, "Evaluate DT test result. Default as disabled")
	flag.IntVarP(&Config.DTEvaluationDelay, "ev-dt-delay", "k", 600, "Delay for DT is beyond this one will be cause failure, unit ms, default 600ms.")
	flag.Float64Var(&Config.DTEvaluationDTPR, "ev-dt-dtpr", 100, "The DT successful rate below this will be cause failure, default 100%.")
	flag.Float64Var(&Config.DTStdExp, "ev-dt-std", 30, "expect standard deviation while do DT evaluation.")
	flag.Float64VarP(&Config.DLTEvaluationSpeed, "speed", "l", 6000, "Download speed should not less than this, Unit KB/s, default 6000KB/s.")
	flag.IntVar(&Config.Loop, "loop", -1, "Loop N round")
	flag.IntVar(&Config.LoopInterval, "loop-interval", 60, "sleep N second between two loop")
	flag.IntVarP(&Config.ResultMin, "result", "r", 10, "The total IPs qualified limitation, default 10")

	flag.BoolVar(&Config.DisableDownload, "disable-download", false, "Deprecated, use --dt-only instead.")
	flag.BoolVar(&Config.DTOnly, "dt-only", false, "Do DT only, we do DT & DLT at the same time by default.")
	flag.BoolVar(&Config.DLTOnly, "dlt-only", false, "Do DLT only, we do DT & DLT at the same time by default.")
	flag.BoolVarP(&Config.IPv4Mode, "ipv4", "4", true, "Just test IPv4.")
	flag.BoolVarP(&Config.IPv6Mode, "ipv6", "6", false, "Just test IPv6.")
	flag.BoolVarP(&Config.TestAll, "test-all", "a", false, "Test all IPs until no more IP left.")
	flag.BoolVar(&tlsHelloFirefox, "hello-firefox", false, "work as firefox")
	flag.BoolVar(&tlsHelloChrome, "hello-chrome", false, "work as chrome")
	flag.BoolVar(&tlsHelloEdge, "hello-edge", false, "work as edge")
	flag.BoolVar(&tlsHelloSafari, "hello-safari", false, "work as safari")
	flag.IntVar(&Config.TestTimeout, "test-timeout", 30, "Test timeout in minutes.")

	flag.BoolVarP(&Config.StoreToFile, "to-file", "w", false, "Write result to csv file, disabled by default.")
	flag.StringVarP(&Config.ResultFile, "out-file", "o", "", "File name of result. ")
	flag.BoolVarP(&Config.StoreToDB, "to-db", "e", false, "Write result to sqlite3 db file.")
	flag.BoolVar(&Config.ResolveLocalASNAndCity, "local-asn", false, "get local asn and city info")
	flag.StringVarP(&Config.DBFile, "db-file", "f", "", "Sqlite3 db file name.")
	flag.StringVarP(&Config.SuffixLabel, "label", "g", "", "the label for a part of the result file's name and sqlite3 record.")
	flag.BoolVar(&Config.ResolveLoc, "resolve-loc", false, "try to resolve location.")
	flag.BoolVarP(&Config.NoCache, "no-cache", "C", false, "disable cdn/proxy caching")

	flag.BoolVarP(&Config.SilenceMode, "silence", "S", false, "silence mode.")
	flag.BoolVarP(&Config.Debug, "debug", "V", false, "Print debug message.")
	// flag.BoolVar(&tcellMode, "tcell", false, "Use tcell form to show debug messages.")
	flag.BoolVarP(&printVersion, "version", "v", false, "Show version.")
	flag.Usage = func() {
		fmt.Print(help)
	}
	flag.Parse()
	if !Config.SilenceMode {
		print_version()
	} else {
		Config.Debug = false
		// tcellMode = false
		Config.StoreToDB = false
		Config.StoreToFile = false
	}
	if len(version) == 0 {
		version = "dev"
	}
	if printVersion {
		os.Exit(0)
	}
	if Config.DisableDownload {
		Config.DTOnly = true
		println("Warning! \"--disable-download\" is deprecated, use \"--dt-only\" instead!")
	}
	if Config.DTHttps {
		Config.DTVia = "https"
		println("Warning! \"--dt-via-https\" is deprecated, use \"--dt-via https|tls|ssl\" instead!")
	}
	if Config.DTOnly && Config.DLTOnly {
		println("\"--dt-only\" and \"--dlt-only\" should not be provided at the same time!")
		os.Exit(1)
	}
	if Config.DTEvaluationDTPR > 100 {
		Config.DTEvaluationDTPR = 100
	} else if Config.DTEvaluationDTPR < 0 {
		Config.DTEvaluationDTPR = 0
	}
	switch strings.ToLower(Config.DTVia) {
	case "https":
		Config.DTHttps = true
	case "ssl", "tls":
		Config.DTHttps = false
	default:
		myLogger.Fatalln("Invalid value for \"--dt-via\". Please use one of: https, tls, or ssl.")
		os.Exit(1)
	}

	if tlsHelloFirefox {
		Config.TLSClientID = utls.HelloFirefox_Auto
		Config.UserAgent = userAgentFirefox
	}
	if tlsHelloChrome {
		Config.TLSClientID = utls.HelloChrome_Auto
		Config.UserAgent = userAgentChrome
	}
	if tlsHelloEdge {
		Config.TLSClientID = utls.HelloEdge_Auto
		Config.UserAgent = userAgentEdge
	}
	if tlsHelloSafari {
		Config.TLSClientID = utls.HelloSafari_Auto
		Config.UserAgent = userAgentSafari
	}

	// tcellMode will activate Config.Debug automatically
	// if tcellMode {
	// 	Config.Debug = true
	// }

	// initialize myLogger
	if Config.SilenceMode {
		loggerLevel = logLevelFatal
	} else {
		loggerLevel = logLevelInfo
		if Config.Debug {
			loggerLevel = logLevelDebug
		}
	}
	// if Config.Debug {
	// 	loggerLevel = logLevelDebug
	// } else {
	// 	if !Config.SilenceMode {
	// 		loggerLevel = logLevelInfo
	// 	} else {
	// 		loggerLevel = logLevelFatal
	// 	}
	// }
	// init myLogger
	myLogger = myLogger.newLogger(loggerLevel)
	// init rand seed
	initRandSeed()

	tMode := int8(0)
	// set Config.IPv4Mode to false, when just --ipv6 provided
	v4Flag := flag.Lookup("ipv4")
	if (!v4Flag.Changed) && Config.IPv6Mode {
		Config.IPv4Mode = false
	}
	if Config.IPv4Mode {
		tMode |= TypeIPv4
	}
	if Config.IPv6Mode {
		tMode |= TypeIPv6
	}
	if tMode == TypeIPErr {
		myLogger.Fatalln("We can't disable both IPv4 and IPv6 at the same time!")
	}

	// check -I|--Config.Interval
	if Config.Interval <= 0 {
		myLogger.Fatalf("\"-I|--interval %v\" should not be smaller than 0!\n", Config.Interval)
	}

	// trim whitespace
	Config.IPFile = strings.TrimSpace(Config.IPFile)
	Config.ResultFile = strings.TrimSpace(Config.ResultFile)
	Config.SuffixLabel = strings.TrimSpace(Config.SuffixLabel)
	Config.HostName = strings.TrimSpace(Config.HostName)
	Config.DTUrl = strings.TrimSpace(Config.DTUrl)
	Config.DLTUrl = strings.TrimSpace(Config.DLTUrl)
	Config.DBFile = strings.TrimSpace(Config.DBFile)

	// var srcIPS []*string
	// no source IPs provided
	if len(ipStr) == 0 && len(Config.IPFile) == 0 {
		// it's invalid when Config.IPv4Mode and Config.IPv6Mode is both true or false at the same time and
		// no specified source IPs or file is provided
		if (tMode&TypeIPv4) == TypeIPv4 && (tMode&TypeIPv6) == TypeIPv6 {
			myLogger.Fatalln("The options \"-4|--ipv4\" and \"-6|--ipv6\" cannot be used together when no specific IPs or file are provided!")
			// not need to exit, because it's a fatal error
			//os.Exit(1)
		}
		if (tMode & TypeIPv4) == TypeIPv4 {
			t_cf_ipv4 := CFIPV4FULL
			if Config.FastMode {
				t_cf_ipv4 = CFIPV4
			}
			err := srcIPs.AddFromSlice(t_cf_ipv4, TypeIPv4)
			if err != nil {
				myLogger.Fatalln(err)
			}
		} else {
			t_cf_ipv6 := CFIPV6FULL
			if Config.FastMode {
				t_cf_ipv6 = CFIPV6
			}
			err := srcIPs.AddFromSlice(t_cf_ipv6, TypeIPv6)
			if err != nil {
				myLogger.Fatalln(err)
			}
		}
	} else {
		if len(ipStr) > 0 {
			err := srcIPs.AddFromSlice(ipStr, tMode)
			if err != nil {
				myLogger.Fatalln(err)
			}
		}
		if len(Config.IPFile) != 0 {
			err := srcIPs.AddFromFile(Config.IPFile, tMode)
			if err != nil {
				myLogger.Fatalln(err)
			}
		}
		if srcIPs.LenInt() == 0 {
			myLogger.Fatalln("no source IPs provided!")
		}
		// set IPv6Mode to true when specific IPs are provided and neither "-4|--ipv4" nor ""-6|--ipv6" is provided
		v6Flag := flag.Lookup("ipv6")
		if !v6Flag.Changed && !v4Flag.Changed {
			Config.IPv6Mode = true
		}
	}

	// check Config.DTUrl is valid URL and uses HTTPS
	if !Config.DLTOnly && Config.DTHttps {
		if len(Config.DTUrl) == 0 {
			myLogger.Fatalf("\"-u|--dt-url %v\" should not be empty!\n", Config.DTUrl)
		}
		t_url, err := url.Parse(Config.DTUrl)
		if err != nil {
			myLogger.Fatalf("\"-u|--dt-url %v\" is not a valid URL!\n", Config.DTUrl)
		}
		if t_url.Scheme != "https" {
			myLogger.Fatalf("\"-u|--dt-url %v\" should use HTTPS!\n", Config.DTUrl)
		}
		if len(Config.DTUrl) > 0 {
			Config.DTUrl = t_url.String()
		}
	}
	// check Config.DLTUrl is valid URL and uses HTTPS
	if !Config.DTOnly {
		if len(Config.DLTUrl) == 0 {
			myLogger.Fatalf("\"-d|--dlt-url %v\" should not be empty!\n", Config.DLTUrl)
		}
		t_url, err := url.Parse(Config.DLTUrl)
		if err != nil {
			myLogger.Fatalf("\"-d|--dlt-url %v\" is not a valid URL!\n", Config.DLTUrl)
		}
		if t_url.Scheme != "https" {
			myLogger.Fatalf("\"-d|--dlt-url %v\" should use HTTPS!\n", Config.DLTUrl)
		}
		if len(Config.DLTUrl) > 0 {
			Config.DLTUrl = t_url.String()
		}
	}

	// shuffle srcIPR and srcIPRsCache when do not Config.TestAll
	// and fix Config.ResultMin

	srcIPs.Shuffle()
	srcIPs.AddPorts(Config.PortStrSlice)
	t_qty := srcIPs.Len()
	// check Config.ResultMin
	if Config.ResultMin <= 0 {
		myLogger.Fatalf("\"-r|--result %v\" should not be smaller than 0!\n", Config.ResultMin)
	}
	// re-calculate Config.ResultMin based on the source IPs
	t_result_min := big.NewInt(int64(Config.ResultMin))
	if Config.TestAll {
		Config.ResultMin = -1
	} else {
		if t_qty.Cmp(t_result_min) == -1 {
			Config.ResultMin = int(t_qty.Int64())
		}
	}
	// set Config.SuffixLabel
	if len(Config.SuffixLabel) == 0 {
		Config.SuffixLabel = Config.HostName
	}
	// set DT parameters when we perform DT
	if !Config.DLTOnly {
		// check parameters
		if Config.DTWorkerThread <= 0 {
			myLogger.Fatalf("\"-m|--dt-thread %v\" should not be smaller than 0!\n", Config.DTWorkerThread)
		}
		if Config.DTCount <= 0 {
			myLogger.Fatalf("\"-c|--dt-count %v\" should not be smaller than 0!\n", Config.DTCount)
		}
		if Config.DTTimeout <= 0 {
			myLogger.Fatalf("\"-t|--dt-timeout %v\" should not be smaller than 0!\n", Config.DTTimeout)
		}
		// if we ping via ssl negotiation and don't perform download test, we need check hostname and port
		if !Config.DTHttps {
			//ping via ssl negotiation
			if len(Config.HostName) == 0 {
				myLogger.Fatal("\"--hostname\" should not be empty. \n")
			}
			Config.DTSource = dtsSSL
		} else {
			// set default value for Config.DTTimeout in Config.DTHttps
			timeoutFlag := flag.Lookup("dt-timeout")
			if !timeoutFlag.Changed {
				Config.DTTimeout = 5000
			}
			// check Config.DTUrl is valid or not by ParseUrl() and set Config.SuffixLabel
			Config.SuffixLabel, _ = parseUrl(Config.DTUrl)
			Config.DTSource = dtsHTTPS
		}
		if Config.EnableDTEvaluation {
			if Config.DTEvaluationDelay <= 0 {
				myLogger.Fatalf("\"-k|--evaluate-dt-delay %v\" should not be smaller than 0!\n", Config.DTEvaluationDelay)
			}
			if Config.DTTimeout < Config.DTEvaluationDelay {
				myLogger.Warning(fmt.Sprintf("\"-t|--dt-timeout\" - %v is less than \"-k|--evaluate-dt-delay\" - %v. This will led to failure for some test!", Config.DTTimeout, Config.DTEvaluationDelay))
				if !confirm("Continue?", 3) {
					os.Exit(0)
				}
			}
			// when --ev-dt is enabled and Config.DTStdExp is greater than 0, we do standard deviation evaluation for delay
			if Config.DTStdExp > 0 {
				Config.EnableStdEv = true
			}
		}
		Config.DTTimeoutDuration = time.Duration(Config.DTTimeout) * time.Millisecond

		// dtThreadsNumLen = len(strconv.Itoa(Config.DTWorkerThread))
	}
	// set downloadTimeMaxDuration only when we need do DLT
	if !Config.DTOnly {
		// dltThreadsAmount = len(strconv.Itoa(Config.DLTWorkerThread))
		if Config.DLTWorkerThread <= 0 {
			myLogger.Fatalf("\"-n|--dlt-thread %v\" should not be smaller than 0!\n", Config.DLTWorkerThread)
		}
		if Config.DLTCount <= 0 {
			myLogger.Fatalf("\"-b|--dlt-count %v\" should not be smaller than 0!\n", Config.DLTCount)
		}

		if Config.DLTDurMax <= 0 {
			myLogger.Fatalf("\"-d|--dlt-period %v\" should not be smaller than 0!\n", Config.DLTDurMax)
		}
		if Config.DLTEvaluationSpeed <= 0 {
			myLogger.Fatalf("\"-l|--speed %v\" should not be smaller than 0!\n", Config.DLTEvaluationSpeed)
		}
		if Config.DLTTimeout > Config.DLTDurMax*1000 {
			myLogger.Fatalf("\"<--dlt-timeout> %v\" should not be bigger than <-d|--dlt-period> %v!\n", Config.DLTTimeout, Config.DLTDurMax)
		}
		// check Config.DLTUrl is valid or not by ParseUrl() and set Config.SuffixLabel
		Config.SuffixLabel, _ = parseUrl(Config.DLTUrl)
		Config.HttpRspTimeoutDuration = time.Duration(Config.DLTTimeout) * time.Millisecond
		Config.DLTDurationInTotal = time.Duration(Config.DLTDurMax) * time.Second
		// dltThreadsNumLen = len(strconv.Itoa(Config.DLTWorkerThread))
	}

	// if we write result file
	if len(Config.ResultFile) > 0 {
		Config.StoreToFile = true
		// fix file type
		re := regexp.MustCompile(`.[c|C][s|S][V|v]$`)
		// file don't end with .csv
		if !re.Match([]byte(Config.ResultFile)) {
			Config.ResultFile = Config.ResultFile + ".csv"
		}
	} else {
		Config.ResultFile = "Result_" + getTimeNowStrSuffix() + "-" + Config.SuffixLabel + ".csv"
	}
	if len(Config.DBFile) > 0 {
		Config.StoreToDB = true
	} else if Config.StoreToDB {
		if len(Config.DBFile) == 0 {
			Config.DBFile = defaultDBFile
		}
	}

}

func validDTResult(tVerifyResult *VerifyResults) bool {
	if tVerifyResult.da > 0.0 &&
		tVerifyResult.da <= float64(Config.DTEvaluationDelay) &&
		tVerifyResult.dtpr*100.0 >= float64(Config.DTEvaluationDTPR) &&
		(!Config.EnableStdEv || (Config.EnableStdEv && tVerifyResult.daStd <= Config.DTStdExp)) {
		return true
	}
	return false
}

func validDLTResult(tVerifyResult *VerifyResults) bool {
	if tVerifyResult.dls >= Config.DLTEvaluationSpeed && tVerifyResult.dlds > downloadSizeMin {
		return true
	}
	return false
}

var (
	dtTaskChan    chan *task
	dtResultChan  chan singleVerifyResult
	dltTaskChan   chan *task
	dltResultChan chan singleVerifyResult
	workerWG      sync.WaitGroup
)

func initWorkers() {
	if !Config.DLTOnly {
		dtTaskChan = make(chan *task, Config.DTWorkerThread)
		dtResultChan = make(chan singleVerifyResult, Config.DTWorkerThread)
		for range Config.DTWorkerThread {
			workerWG.Add(1)
			if Config.DTHttps {
				go downloadWorkerNew(dtTaskChan, dtResultChan, &workerWG, &Config.DTUrl, Config.DTTimeoutDuration, Config.DTCount, true)
			} else {
				go sslDTWorkerNew(dtTaskChan, dtResultChan, &workerWG)
			}
		}
	}
	if !Config.DTOnly {
		dltTaskChan = make(chan *task, Config.DLTWorkerThread)
		dltResultChan = make(chan singleVerifyResult, Config.DLTWorkerThread)
		for range Config.DLTWorkerThread {
			workerWG.Add(1)
			go downloadWorkerNew(dltTaskChan, dltResultChan, &workerWG, &Config.DLTUrl, Config.HttpRspTimeoutDuration, Config.DLTCount, false)
		}
	}
}

func runDTSingleRound(ips []*string, handler func(singleVerifyResult)) {
	size := len(ips)
	if size == 0 {
		return
	}
	max_failure := get_max_failure(true)
	go func() {
		for _, ip := range ips {
			dtTaskChan <- NewTask(ip, max_failure)
		}
	}()

	for range size {
		handler(<-dtResultChan)
	}
}

func runDLTSingleRound(ips []*string, handler func(singleVerifyResult)) {
	size := len(ips)
	if size == 0 {
		return
	}
	max_failure := get_max_failure(false)
	go func() {
		for _, ip := range ips {
			dltTaskChan <- NewTask(ip, max_failure)
		}
	}()

	for range size {
		handler(<-dltResultChan)
	}
}

func runWorker() {
	initWorkers()

	tmpResultMap := make(map[string]VerifyResults)
	var tmpTestSlice map[string]bool
	var thisSourceIPs = srcIPs
	var t_result_min = Config.ResultMin
	var start_time = time.Now()

RETRY_LOOP:
	for {
		looper := NewSafeLooperWithInterval(Config.Loop, Config.LoopInterval*1000)
	LOOP:
		for {
			dtDoneTasks := 0
			dltDoneTasks := 0
			tmpTestSlice = make(map[string]bool)

		SINGLE_ROUND:
			for {
				if time.Since(start_time) >= time.Duration(Config.TestTimeout)*time.Minute {
					break SINGLE_ROUND
				}

				// DT Stage
				if !Config.DLTOnly {
					dtBatch := thisSourceIPs.RetrieveSome(Config.DTWorkerThread, !Config.TestAll)
					if len(dtBatch) == 0 {
						break SINGLE_ROUND
					}

					dltBatch := make([]*string, 0)
					cachedMap := make(map[string]VerifyResults)

					runDTSingleRound(dtBatch, func(dtRes singleVerifyResult) {
						dtDoneTasks++
						tVerifyResult := calcResult(dtRes, false)
						t_ip := *tVerifyResult.ip

						if validDTResult(&tVerifyResult) {
							if !Config.DTOnly {
								cachedMap[t_ip] = tVerifyResult
								dltBatch = append(dltBatch, &t_ip)
								if Config.Debug {
									displayDetails(false, looper.Status() > -1, []VerifyResults{tVerifyResult})
								}
							} else {
								if Config.ResolveLoc && Config.SilenceMode && looper.Status() == -1 && (tVerifyResult.loc == nil || len(*tVerifyResult.loc) == 0) {
									loc := getGeoInfoFromCF(tVerifyResult.ip)
									tVerifyResult.loc = &loc
								}
								v, ok := tmpResultMap[t_ip]
								if ok {
									tVerifyResult.combine(v)
								}
								tmpResultMap[t_ip] = tVerifyResult
								displayDetails(false, looper.Status() > -1, []VerifyResults{tVerifyResult})
								tmpTestSlice[t_ip] = true
							}
						} else {
							if looper.InLooping() {
								v, ok := tmpResultMap[t_ip]
								if ok {
									tVerifyResult.combine(v)
								}
								tmpResultMap[t_ip] = tVerifyResult
							}
							if Config.Debug {
								displayDetails(false, looper.Status() > -1, []VerifyResults{tVerifyResult})
							}
						}
					})

					// DLT Stage for DT candidates
					if !Config.DTOnly && len(dltBatch) > 0 {
						runDLTSingleRound(dltBatch, func(dltRes singleVerifyResult) {
							dltDoneTasks++
							tVerifyResult := calcResult(dltRes, true)
							t_ip := *tVerifyResult.ip
							v := cachedMap[t_ip]
							tVerifyResult.combine(v)

							if validDLTResult(&tVerifyResult) && validDTResult(&tVerifyResult) {
								if Config.ResolveLoc && Config.SilenceMode && looper.Status() == -1 && (tVerifyResult.loc == nil || len(*tVerifyResult.loc) == 0) {
									loc := getGeoInfoFromCF(&t_ip)
									tVerifyResult.loc = &loc
								}
								mv, ok := tmpResultMap[t_ip]
								if ok {
									tVerifyResult.combine(mv)
								}
								tmpResultMap[t_ip] = tVerifyResult
								tmpTestSlice[t_ip] = true
								displayDetails(true, looper.Status() > -1, []VerifyResults{tVerifyResult})
							} else {
								if looper.InLooping() {
									mv, ok := tmpResultMap[t_ip]
									if ok {
										tVerifyResult.combine(mv)
									}
									tmpResultMap[t_ip] = tVerifyResult
								}
								if Config.Debug {
									displayDetails(true, looper.Status() > -1, []VerifyResults{tVerifyResult})
								}
							}
						})
					}
				} else {
					// DLT Only Stage
					dltBatch := thisSourceIPs.RetrieveSome(Config.DLTWorkerThread, !Config.TestAll)
					if len(dltBatch) == 0 {
						break SINGLE_ROUND
					}
					runDLTSingleRound(dltBatch, func(dltRes singleVerifyResult) {
						dltDoneTasks++
						tVerifyResult := calcResult(dltRes, true)
						t_ip := *tVerifyResult.ip
						if validDLTResult(&tVerifyResult) {
							if Config.ResolveLoc && Config.SilenceMode && looper.Status() == -1 && (tVerifyResult.loc == nil || len(*tVerifyResult.loc) == 0) {
								loc := getGeoInfoFromCF(&t_ip)
								tVerifyResult.loc = &loc
							}
							v, ok := tmpResultMap[t_ip]
							if ok {
								tVerifyResult.combine(v)
							}
							tmpResultMap[t_ip] = tVerifyResult
							tmpTestSlice[t_ip] = true
							displayDetails(true, looper.Status() > -1, []VerifyResults{tVerifyResult})
						} else {
							if looper.InLooping() {
								v, ok := tmpResultMap[t_ip]
								if ok {
									tVerifyResult.combine(v)
								}
								tmpResultMap[t_ip] = tVerifyResult
							}
							if Config.Debug {
								displayDetails(true, looper.Status() > -1, []VerifyResults{tVerifyResult})
							}
						}
					})
				}

				if Config.Debug {
					displayStat(overAllStat{
						dtTasksDone:  dtDoneTasks,
						dltTasksDone: dltDoneTasks,
						resultCount:  len(tmpTestSlice),
						remain:       thisSourceIPs.LenInt(),
					})
				}

				if !Config.TestAll && len(tmpTestSlice) >= t_result_min {
					break SINGLE_ROUND
				}
			}

			if len(tmpResultMap) == 0 {
				break LOOP
			}
			if !looper.Loop() {
				break LOOP
			} else {
				tmp_slice := make([]string, 0, len(tmpResultMap))
				for k := range tmpResultMap {
					tmp_slice = append(tmp_slice, k)
				}
				newSourceIPs := NewSourceIPs()
				newSourceIPs.AddFromSlice(tmp_slice, TypeIPv4|TypeIPv6)
				newSourceIPs.AddPorts(Config.PortStrSlice)
				thisSourceIPs = newSourceIPs
				if !Config.TestAll {
					t_result_min = len(tmp_slice)
				}
				looper.Sleep()
			}
		}

		for tIP := range tmpTestSlice {
			tr := tmpResultMap[tIP]
			isValid := true
			if !Config.DLTOnly && !validDTResult(&tr) {
				isValid = false
			}
			if !Config.DTOnly && !validDLTResult(&tr) {
				isValid = false
			}
			if isValid {
				verifyResultsMap[tIP] = tr
			}
		}

		thisSourceIPs = srcIPs
		if (!Config.TestAll && len(verifyResultsMap) >= Config.ResultMin) || thisSourceIPs.IsEmpty() || time.Since(start_time) >= time.Duration(Config.TestTimeout)*time.Minute {
			break RETRY_LOOP
		} else {
			t_result_min = Config.ResultMin - len(verifyResultsMap)
			verifyResultsMap = make(map[string]VerifyResults)
		}
	}

	if dtTaskChan != nil {
		close(dtTaskChan)
	}
	if dltTaskChan != nil {
		close(dltTaskChan)
	}
	workerWG.Wait()
}

func main() {

	// start controller worker
	runWorker()
	if len(verifyResultsMap) > 0 {
		verifyResultsSlice := make([]VerifyResults, 0)
		for _, v := range verifyResultsMap {
			if Config.ResolveLoc && len(*v.loc) == 0 {
				t_loc := getGeoInfoFromCF(v.ip)
				v.loc = &t_loc
			}
			verifyResultsSlice = append(verifyResultsSlice, v)
		}
		var records []DBRecord
		if Config.StoreToFile || Config.StoreToDB {
			records = genDBRecords(verifyResultsSlice, Config.ResolveLocalASNAndCity)
			// write to csv file
			if Config.StoreToFile {
				if !Config.SilenceMode {
					myLogger.Print("Write to csv " + Config.ResultFile)
				}
				writeCSVResult(records, Config.ResultFile)
				if !Config.SilenceMode {
					myLogger.Println("  Done!")
				}
			}
			// write to db
			if Config.StoreToDB {
				if !Config.SilenceMode {
					myLogger.Print("Write to sqlite3 db file " + Config.DBFile)
				}
				saveDBRecords(records, Config.DBFile)
				if !Config.SilenceMode {
					myLogger.Println("  Done!")
				}
			}
		}
		// sort by speed
		sort.Sort(sort.Reverse(resultSpeedSorter(verifyResultsSlice)))
		if !Config.SilenceMode {
			myLogger.Println()
			myLogger.Println("All Results:")
			printFinalStat(verifyResultsSlice, Config.DTOnly, false)
		} else {
			if Config.Loop > 0 {
				printFinalStat(verifyResultsSlice, Config.DTOnly, true)
			}
		}
	}
}
