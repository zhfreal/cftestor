package main

import (
	"bufio"
	"fmt"
	"math"
	"math/big"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	utls "github.com/refraction-networking/utls"
	flag "github.com/spf13/pflag"
)

func print_version() {
	fmt.Println(appArt)
	fmt.Println(`  CF CDN IP scanner, find best IPs for your Cloudflare CDN applications.
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
	var portStrSlice []string

	// version = "dev"
	flag.BoolVar(&fastMode, "fast", false, "Fast mode")
	flag.VarP(&ipStr, "ip", "s", "Specific IP or CIDR for test.")
	flag.StringVarP(&ipFile, "in", "i", "", "Specific file of IPs and CIDRs for test.")

	flag.IntVarP(&dtWorkerThread, "dt-thread", "m", 20, "Number of concurrent threads for Delay Test(DT).")
	flag.IntVarP(&dtTimeout, "dt-timeout", "t", 2000, "Timeout for single DT(ms).")
	flag.IntVarP(&dtCount, "dt-count", "c", 2, "Tries of DT for a IP.")
	// flag.IntVarP(&port, "port", "p", 443, "Port to test")
	flag.StringSliceVarP(&portStrSlice, "port", "p", []string{}, "Port to test, could be specific one or more ports at same time.")
	flag.StringVar(&hostName, "hostname", DefaultTestHost, "Hostname for DT test.")
	flag.StringVar(&dtVia, "dt-via", "https", "DT via https rather than SSL/TLS shaking hands.")
	flag.IntVar(&dtHttpRspReturnCodeExpected, "dt-expect-code", 200, "HTTP status code expected for DT test.")
	flag.BoolVar(&dtHttps, "dt-via-https", false, "DT via https rather than SSL/TLS shaking hands.")
	flag.StringVar(&dtUrl, "dt-url", defaultDTUrl, "Specific the url while DT via https.")

	flag.IntVarP(&dltWorkerThread, "dlt-thread", "n", 1, "Number of concurrent Threads for Download Test(DLT).")
	flag.IntVarP(&dltDurMax, "dlt-period", "d", 10, "The total times escaped for single DLT in seconds, default 10s.")
	flag.IntVarP(&dltCount, "dlt-count", "b", 1, "Tries of DLT for a IP, default 1.")
	flag.StringVarP(&dltUrl, "dlt-url", "u", defaultDLTUrl, "Customize test URL for DLT.")
	flag.IntVar(&dltTimeout, "dlt-timeout", 5000, "Specify the timeout for http reponse when do DLT in milliseconds, default 5000 ms.")
	flag.IntVarP(&interval, "interval", "I", 500, "Interval between two tests, unit ms, default 500ms.")

	flag.BoolVar(&enableDTEvaluation, "ev-dt", false, "Evaluate DT test result. Default as disabled")
	flag.IntVarP(&dtEvaluationDelay, "ev-dt-delay", "k", 600, "Delay for DT is beyond this one will be cause failure, unit ms, default 600ms.")
	flag.Float64VarP(&dtEvaluationDTPR, "ev-dt-dtpr", "S", 100, "The DT successful rate below this will be cause failure, default 100%.")
	flag.Float64Var(&dtStdExp, "ev-dt-std", 0, "expect standard deviation while do DT evaluation.")
	flag.Float64VarP(&dltEvaluationSpeed, "speed", "l", 6000, "Download speed should not less than this, Unit KB/s, default 6000KB/s.")
	flag.IntVarP(&resultMin, "result", "r", 10, "The total IPs qualified limitation, default 10")

	flag.BoolVar(&disableDownload, "disable-download", false, "Deprecated, use --dt-only instead.")
	flag.BoolVar(&dtOnly, "dt-only", false, "Do DT only, we do DT & DLT at the same time by default.")
	flag.BoolVar(&dltOnly, "dlt-only", false, "Do DLT only, we do DT & DLT at the same time by default.")
	flag.BoolVarP(&ipv4Mode, "ipv4", "4", true, "Just test IPv4.")
	flag.BoolVarP(&ipv6Mode, "ipv6", "6", false, "Just test IPv6.")
	flag.BoolVarP(&testAll, "test-all", "a", false, "Test all IPs until no more IP left.")
	flag.BoolVar(&tlsHelloFirefox, "hello-firefox", false, "work as firefox")
	flag.BoolVar(&tlsHelloChrome, "hello-chrome", false, "work as chrome")
	flag.BoolVar(&tlsHelloEdge, "hello-edge", false, "work as edge")
	flag.BoolVar(&tlsHelloSafari, "hello-safari", false, "work as safari")

	flag.BoolVarP(&storeToFile, "to-file", "w", false, "Write result to csv file, disabled by default.")
	flag.StringVarP(&resultFile, "out-file", "o", "", "File name of result. ")
	flag.BoolVarP(&storeToDB, "to-db", "e", false, "Write result to sqlite3 db file.")
	flag.BoolVar(&resolveLocalASNAndCity, "local-asn", false, "get local asn and city info")
	flag.StringVarP(&dbFile, "db-file", "f", "", "Sqlite3 db file name.")
	flag.StringVarP(&suffixLabel, "label", "g", "", "the label for a part of the result file's name and sqlite3 record.")
	flag.BoolVar(&ResolveLoc, "resolve-loc", false, "try to resolve location.")

	flag.BoolVar(&silenceMode, "silence", false, "silence mode.")
	flag.BoolVarP(&debug, "debug", "V", false, "Print debug message.")
	flag.BoolVar(&tcellMode, "tcell", false, "Use tcell form to show debug messages.")
	flag.BoolVarP(&printVersion, "version", "v", false, "Show version.")
	flag.Usage = func() {
		fmt.Print(help)
	}
	flag.Parse()
	if !silenceMode {
		print_version()
	} else {
		debug = false
		tcellMode = false
		storeToDB = false
		storeToFile = false
	}
	if len(version) == 0 {
		version = "dev"
	}
	if printVersion {
		os.Exit(0)
	}
	if disableDownload {
		dtOnly = true
		println("Warning! \"--disable-download\" is deprecated, use \"--dt-only\" instead!")
	}
	if dtHttps {
		dtVia = "https"
		println("Warning! \"--dt-via-https\" is deprecated, use \"--dt-via https|tls|ssl\" instead!")
	}
	if dtOnly && dltOnly {
		println("\"--dt-only\" and \"--dlt-only\" should not be provided at the same time!")
		os.Exit(1)
	}
	if dtEvaluationDTPR > 100 {
		dtEvaluationDTPR = 100
	} else if dtEvaluationDTPR < 0 {
		dtEvaluationDTPR = 0
	}
	dtVia = strings.ToLower(dtVia)
	if dtVia == "https" {
		dtHttps = true
	} else if dtVia == "ssl" || dtVia == "tls" {
		dtHttps = false
	} else {
		println("invalid value found! Please use \"--dt-via <https|tls|ssl>\"!")
		os.Exit(1)
	}

	if tlsHelloFirefox {
		tlsClientID = utls.HelloFirefox_Auto
		userAgent = userAgentFirefox
	}
	if tlsHelloChrome {
		tlsClientID = utls.HelloChrome_Auto
		userAgent = userAgentChrome
	}
	if tlsHelloEdge {
		tlsClientID = utls.HelloEdge_Auto
		userAgent = userAgentEdge
	}
	if tlsHelloSafari {
		tlsClientID = utls.HelloSafari_Auto
		userAgent = userAgentSafari
	}

	// tcellMode will activate debug automatically
	if tcellMode {
		debug = true
	}

	// initialize myLogger
	if debug {
		loggerLevel = logLevelDebug
	} else {
		if !silenceMode {
			loggerLevel = logLevelInfo
		} else {
			loggerLevel = logLevelFatal
		}
	}
	// init myLogger
	myLogger = myLogger.newLogger(loggerLevel)
	// init rand seed
	initRandSeed()

	// set false for ipv4Mode, when just ipv6 flag set to true
	v4Flag := flag.Lookup("ipv4")
	if (!v4Flag.Changed) && ipv6Mode {
		ipv4Mode = false
	}

	// it's invalid when ipv4Mode and ipv6Mode is both true or false
	if ipv4Mode == ipv6Mode {
		myLogger.Fatalln("\"-4|--ipv4\" and \"-6|--ipv6\" should not be provided at the same time!")
		os.Exit(1)
	}

	// check -I|--interval
	if interval <= 0 {
		myLogger.Fatalf("\"-I|--interval %v\" should not be smaller than 0!\n", interval)
	}

	// trim whitespace
	ipFile = strings.TrimSpace(ipFile)
	resultFile = strings.TrimSpace(resultFile)
	suffixLabel = strings.TrimSpace(suffixLabel)
	hostName = strings.TrimSpace(hostName)
	dtUrl = strings.TrimSpace(dtUrl)
	dltUrl = strings.TrimSpace(dltUrl)
	dbFile = strings.TrimSpace(dbFile)

	var srcIPS []*string
	if len(ipStr) != 0 {
		for i := 0; i < len(ipStr); i++ {
			srcIPS = append(srcIPS, &ipStr[i])
		}
	}
	if len(ipFile) != 0 {
		file, err := os.Open(ipFile)
		if err != nil {
			myLogger.Fatalf("file \"%s\" is not accessible! \n", ipFile)
		}
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			tIp := strings.TrimSpace(scanner.Text())
			if len(tIp) == 0 {
				continue
			}
			srcIPS = append(srcIPS, &tIp)
		}
	}

	// no source IPs provided
	if len(ipStr) == 0 && len(ipFile) == 0 || len(srcIPS) == 0 {
		if !ipv6Mode {
			t_cf_ipv4 := CFIPV4FULL
			if fastMode {
				t_cf_ipv4 = CFIPV4
			}
			for i := 0; i < len(t_cf_ipv4); i++ {
				srcIPS = append(srcIPS, &t_cf_ipv4[i])
			}
		} else {
			t_cf_ipv6 := CFIPV6FULL
			if fastMode {
				t_cf_ipv6 = CFIPV6
			}
			for i := 0; i < len(t_cf_ipv6); i++ {
				srcIPS = append(srcIPS, &t_cf_ipv6[i])
			}
		}
	}

	// shuffle srcIPR and srcIPRsCache when do not testAll
	// and fix resultMin
	t_qty := big.NewInt(0)
	for i := 0; i < len(srcIPS); i++ {
		ips := strings.TrimSpace(*srcIPS[i])
		ips = strings.Split(ips, "#")[0]
		if isValidIPs(ips) {
			ipr := NewIPRangeFromCIDR(&ips)
			if ipr == nil {
				myLogger.Fatalf("\"%v\" is invalid!\n", ips)
			}
			// when it do not testAll and ipr is not bigger than maxHostLenBig, extract to to cache
			t_qty = t_qty.Add(t_qty, ipr.Len)
			if ipr.Len.Cmp(maxHostLenBig) < 1 {
				srcIPRsExtracted = append(srcIPRsExtracted, ipr.ExtractAll()...)
			} else {
				// when it do not perform tealAll or not bigger than maxHostLenBig, just put it to srcIPRs
				srcIPRsRaw = append(srcIPRsRaw, ipr)
			}
		} else if isValidHost(ips) {
			srcHosts = append(srcHosts, &ips)
			t_qty = t_qty.Add(t_qty, big.NewInt(1))
		} else {
			myLogger.Fatalf("\"%v\" is neither valid IP/CIDR nor host!\n", ips)
		}
	}
	// shuffle srcIPRsRaw, srcIPRsExtracted, and srcHosts
	myRand.Shuffle(len(srcIPRsRaw), func(m, n int) {
		srcIPRsRaw[m], srcIPRsRaw[n] = srcIPRsRaw[n], srcIPRsRaw[m]
	})
	myRand.Shuffle(len(srcIPRsExtracted), func(m, n int) {
		srcIPRsExtracted[m], srcIPRsExtracted[n] = srcIPRsExtracted[n], srcIPRsExtracted[m]
	})
	myRand.Shuffle(len(srcHosts), func(m, n int) {
		srcHosts[m], srcHosts[n] = srcHosts[n], srcHosts[m]
	})
	// check resultMin
	if resultMin <= 0 {
		myLogger.Fatalf("\"-r|--result %v\" should not be smaller than 0!\n", resultMin)
	}
	// re-caculate resultMin based on the source IPs
	t_result_min := big.NewInt(int64(resultMin))
	if testAll {
		resultMin = -1
	} else {
		if t_qty.Cmp(t_result_min) == -1 {
			resultMin = int(t_qty.Int64())
		}
	}
	port_regex := regexp.MustCompile(`[,;|]+`)
	port_range_regex := regexp.MustCompile(`\d+[-:]\d+`)
	port_range_split_regex := regexp.MustCompile(`[-:]`)
	// set ports
	if len(portStrSlice) > 0 {
		for _, portStr := range portStrSlice {
			portStr_slice := port_regex.Split(portStr, -1)
			for _, t_port_str := range portStr_slice {
				t_port_str = strings.TrimSpace(t_port_str)
				if len(t_port_str) == 0 {
					continue
				}
				// it's a range
				if port_range_regex.MatchString(t_port_str) {
					t_port_list := port_range_split_regex.Split(t_port_str, -1)
					if len(t_port_list) != 2 {
						myLogger.Fatalf("\"-p|--port %v\" is invalid!\n", t_port_str)
					}
					t_start_port := t_port_list[0]
					t_end_port := t_port_list[1]
					start_port, err := strconv.Atoi(t_start_port)
					if err != nil {
						myLogger.Fatalf("\"-p|--port %v\" is invalid!\n", t_start_port)
					}
					end_port, err := strconv.Atoi(t_end_port)
					if err != nil {
						myLogger.Fatalf("\"-p|--port %v\" is invalid!\n", t_end_port)
					}
					if start_port > end_port || start_port < 1 || end_port > 65535 {
						myLogger.Fatalf("\"-p|--port %v\" is invalid!\n", t_port_str)
					}
					for i := start_port; i <= end_port; i++ {
						ports = append(ports, i)
					}
				} else { // it's a single port
					port, err := strconv.Atoi(t_port_str)
					if err != nil || port < 1 || port > 65535 {
						myLogger.Fatalf("\"-p|--port %v\" is invalid!\n", t_port_str)
					}
					ports = append(ports, port)
				}
			}
		}
		// clean ports, make them unique
		ports = uniqueIntSlice(ports)
	}
	if len(ports) == 0 {
		ports = append(ports, 443)
	}
	// set suffixLabel
	if len(suffixLabel) == 0 {
		suffixLabel = hostName
	}
	// set DT parameters when we perform DT
	if !dltOnly {
		// check parameters
		if dtWorkerThread <= 0 {
			myLogger.Fatalf("\"-m|--dt-thread %v\" should not be smaller than 0!\n", dtWorkerThread)
		}
		if dtCount <= 0 {
			myLogger.Fatalf("\"-c|--dt-count %v\" should not be smaller than 0!\n", dtCount)
		}
		if dtTimeout <= 0 {
			myLogger.Fatalf("\"-t|--dt-timeout %v\" should not be smaller than 0!\n", dtTimeout)
		}
		// if we ping via ssl negotiation and don't perform download test, we need check hostname and port
		if !dtHttps {
			//ping via ssl negotiation
			if len(hostName) == 0 {
				myLogger.Fatal("\"--hostname\" should not be empty. \n")
			}
			dtSource = dtsSSL
		} else {
			// set default value for dtTimeout in dtHttps
			timeoutFlag := flag.Lookup("dt-timeout")
			if !timeoutFlag.Changed {
				dtTimeout = 5000
			}
			// check dtUrl is valid or not by ParseUrl() and set suffixLabel
			suffixLabel, _ = parseUrl(dtUrl)
			dtSource = dtsHTTPS
		}
		if enableDTEvaluation {
			if dtEvaluationDelay <= 0 {
				myLogger.Fatalf("\"-k|--evaluate-dt-delay %v\" should not be smaller than 0!\n", dtEvaluationDelay)
			}
			if dtTimeout < dtEvaluationDelay {
				myLogger.Warning(fmt.Sprintf("\"-t|--dt-timeout\" - %v is less than \"-k|--evaluate-dt-delay\" - %v. This will led to failure for some test!", dtTimeout, dtEvaluationDelay))
				if !confirm("Continue?", 3) {
					os.Exit(0)
				}
			}
			// when --ev-dt is enabled and dtStdExp is greater than 0, we do standard deviation evaluation for delay
			if dtStdExp > 0 {
				enableStdEv = true
			}
		}
		dtTimeoutDuration = time.Duration(dtTimeout) * time.Millisecond

		// dtThreadsNumLen = len(strconv.Itoa(dtWorkerThread))
	}
	// set downloadTimeMaxDuration only when we need do DLT
	if !dtOnly {
		// dltThreadsAmount = len(strconv.Itoa(dltWorkerThread))
		if dltWorkerThread <= 0 {
			myLogger.Fatalf("\"-n|--dlt-thread %v\" should not be smaller than 0!\n", dltWorkerThread)
		}
		if dltCount <= 0 {
			myLogger.Fatalf("\"-b|--dlt-count %v\" should not be smaller than 0!\n", dltCount)
		}

		if dltDurMax <= 0 {
			myLogger.Fatalf("\"-d|--dlt-period %v\" should not be smaller than 0!\n", dltDurMax)
		}
		if dltEvaluationSpeed <= 0 {
			myLogger.Fatalf("\"-l|--speed %v\" should not be smaller than 0!\n", dltEvaluationSpeed)
		}
		if dltTimeout > dltDurMax*1000 {
			myLogger.Fatalf("\"<--dlt-timeout> %v\" should not be bigger than <-d|--dlt-period> %v!\n", dltTimeout, dltDurMax)
		}
		// check dltUrl is valid or not by ParseUrl() and set suffixLabel
		suffixLabel, _ = parseUrl(dltUrl)
		httpRspTimeoutDuration = time.Duration(dltTimeout) * time.Millisecond
		dltDurationInTotal = time.Duration(dltDurMax) * time.Second
		// dltThreadsNumLen = len(strconv.Itoa(dltWorkerThread))
	}

	// if we write result file
	if len(resultFile) > 0 {
		storeToFile = true
		// fix file type
		re := regexp.MustCompile(`.[c|C][s|S][V|v]$`)
		// file don't end with .csv
		if !re.Match([]byte(resultFile)) {
			resultFile = resultFile + ".csv"
		}
	} else {
		resultFile = "Result_" + getTimeNowStrSuffix() + "-" + suffixLabel + ".csv"
	}
	if len(dbFile) > 0 {
		storeToDB = true
	} else if storeToDB {
		if len(dbFile) == 0 {
			dbFile = defaultDBFile
		}
	}

	if debug && tcellMode { // It's running on tcell mode
		// reset the position of debugHint and debugTitle according maxResultsDisplay and resultMin
		if !testAll && resultMin < maxResultsDisplay {
			maxResultsDisplay = resultMin
			titleDebugHintRow = titleResultRow + maxResultsDisplay + 2
			titleDebugRow = titleDebugHintRow + 1
		}
		// init
		resultStrSlice = make([][]*string, 0)
		debugStrSlice = make([][]*string, 0)
		detailTitleSlice = make([]string, 0)
		// fix interval
		// statInterval = statisticIntervalT
		// fix rows when --dlt-only mode
		if dltOnly {
			titleCancelRow -= 1
			titleTasksStatRow -= 1
			titleResultHintRow -= 1
			titleResultRow -= 1
			titleDebugHintRow -= 1
			titleDebugRow -= 1
		}
		initTitleStr()
		// init tcell screen instance
		ts, te := tcell.NewScreen()
		if te != nil {
			fmt.Fprintf(os.Stderr, "%v\n", te)
			os.Exit(1)
		}
		if te := ts.Init(); te != nil {
			fmt.Fprintf(os.Stderr, "%v\n", te)
			os.Exit(1)
		}
		termAll = &ts
		(*termAll).SetStyle(normalStyle)
		// (*termAll).Sync()
	}
}

func runWorker() {

	var wg sync.WaitGroup
	var dtTaskChan = make(chan *string, dtWorkerThread)
	var dtResultChan = make(chan singleVerifyResult, cap(dtTaskChan))
	var dltTaskChan = make(chan *string, dltWorkerThread)
	var dltResultChan = make(chan singleVerifyResult, cap(dltTaskChan))

	if debug && tcellMode {
		go termControl(&wg)
		wg.Add(1)
	}
	dtDoneTasks := 0
	// the item in dtTaskCache is a ip string.
	dtTaskCache := make([]*string, 0)
	dltDoneTasks := 0
	// the item in dltTaskCache is a ip string.
	dltTaskCache := make([]*string, 0)
	// the key of cacheResultMap should be: <ipv4:port> or <[ipv6]:port>, should not be just a ip string.
	cacheResultMap := make(map[string]VerifyResults)
	haveEnoughResult := false
	showQuitWaiting := false

	if !dltOnly {
		for i := 0; i < dtWorkerThread; i++ {
			wg.Add(1)
			if dtHttps {
				go downloadWorkerNew(dtTaskChan, dtResultChan, &wg, &dtUrl, dtTimeoutDuration, dtCount, true)
			} else {
				go sslDTWorkerNew(dtTaskChan, dtResultChan, &wg)
			}
		}
	}
	if !dtOnly {
		for i := 0; i < dltWorkerThread; i++ {
			wg.Add(1)
			go downloadWorkerNew(dltTaskChan, dltResultChan, &wg, &dltUrl, httpRspTimeoutDuration, dltCount, false)
		}
	}

LOOP:
	for {
		// DT
		if !dltOnly {
			if len(dtTaskCache) < dtWorkerThread {
				dtTaskCache = append(dtTaskCache, retrieveSome(dtWorkerThread)...)
			}
			// no more sources for testing
			t_dt_sources_len := len(dtTaskCache)
			if t_dt_sources_len == 0 {
				break LOOP
			}
			// // print stat
			// if debug {
			// 	(overAllStat{
			// 		dtTasksDone:  dtDoneTasks,
			// 		dtOnGoing:    0,
			// 		dtCached:     len(dtTaskCache),
			// 		dltTasksDone: dltDoneTasks,
			// 		dltOnGoing:   0,
			// 		dltCached:    len(dltTaskCache),
			// 		resultCount:  len(verifyResultsMap),
			// 	})
			// }
			// put task
			go func() {
				for i := 0; i < t_dt_sources_len; i++ {
					dtTaskChan <- dtTaskCache[i]
				}
				dtTaskCache = make([]*string, 0)
			}()
			// retrieve from dtResultChan
			for i := 0; i < t_dt_sources_len; i++ {
				dtResult := <-dtResultChan
				// if ip not test then put it into dltTaskChan
				dtDoneTasks += 1
				var tVerifyResult = singleResultStatistic(dtResult, false)
				if tVerifyResult.da > 0.0 &&
					tVerifyResult.da <= float64(dtEvaluationDelay) &&
					tVerifyResult.dtpr*100.0 >= float64(dtEvaluationDTPR) &&
					(!enableStdEv || (enableStdEv && tVerifyResult.daStd <= dtStdExp)) {
					if !dtOnly { // there are download test ongoing
						// put ping test result to cacheResultMap for later
						cacheResultMap[*tVerifyResult.ip] = tVerifyResult
						dltTaskCache = append(dltTaskCache, tVerifyResult.ip)
						// debug msg, show only in debug mode
						if debug {
							displayDetails(false, false, []VerifyResults{tVerifyResult})
						}
					} else { // Download test disabled
						// non-debug msg
						displayDetails(true, false, []VerifyResults{tVerifyResult})
						verifyResultsMap[tVerifyResult.ip] = tVerifyResult
						// we have expected result, break LOOP
						if !testAll && len(verifyResultsMap) >= resultMin {
							haveEnoughResult = true
						}
					}
				} else if debug {
					// debug msg
					displayDetails(false, false, []VerifyResults{tVerifyResult})
				}
			}
			if debug {
				displayStat(overAllStat{
					dtTasksDone:  dtDoneTasks,
					dtOnGoing:    0,
					dtCached:     len(dtTaskCache),
					dltTasksDone: dltDoneTasks,
					dltOnGoing:   0,
					dltCached:    len(dltTaskCache),
					resultCount:  len(verifyResultsMap),
				})
			}
		}
		//DLT
		if !dtOnly {
			// no source to do DLT
			if len(dltTaskCache) <= 0 {
				// DT enabled, just continue to do DT
				if !dltOnly {
					continue
				} else {
					// retrieve source IP
					dltTaskCache = append(dltTaskCache, retrieveSome(dltWorkerThread)...)
					// no source IP, break LOOP
					if len(dltTaskCache) == 0 {
						break LOOP
					}
				}
			}
			t_dlt_sources_len := len(dltTaskCache)
			// print stat
			// if debug {
			// 	displayStat(overAllStat{
			// 		dtTasksDone:  dtDoneTasks,
			// 		dtOnGoing:    0,
			// 		dtCached:     len(dtTaskCache),
			// 		dltTasksDone: dltDoneTasks,
			// 		dltOnGoing:   0,
			// 		dltCached:    t_dlt_sources_len,
			// 		resultCount:  len(verifyResultsMap),
			// 	})
			// }
			// put task
			go func() {
				for i := 0; i < t_dlt_sources_len; i++ {
					dltTaskChan <- dltTaskCache[i]
				}
				dltTaskCache = make([]*string, 0)
			}()
			// retrieve result
			for i := 0; i < t_dlt_sources_len; i++ {
				out := <-dltResultChan
				dltDoneTasks += 1
				var tVerifyResult = singleResultStatistic(out, true)
				var v = VerifyResults{}
				if dltOnly {
					v = tVerifyResult
				} else {
					v = cacheResultMap[*tVerifyResult.ip]
					// reset TestTime according download test result
					v.testTime = tVerifyResult.testTime
					v.dltc = tVerifyResult.dltc
					v.dls = tVerifyResult.dls
					v.dltpc = tVerifyResult.dltpc
					v.dltpr = tVerifyResult.dltpr
					v.dlds = tVerifyResult.dlds
					v.dltd = tVerifyResult.dltd
					// update ping static
					tDelayTotal := float64(v.dtpc) * v.da
					v.dtc += tVerifyResult.dtc
					v.dtpc += tVerifyResult.dtpc
					if v.dtc > 0 {
						v.dtpr = float64(v.dtpc) / float64(v.dtc)
					}
					if tVerifyResult.dtpc > 0 {
						v.dmx = math.Max(v.dmx, tVerifyResult.dmx)
						v.dmi = math.Min(v.dmi, tVerifyResult.dmi)
						v.da = (tDelayTotal + float64(tVerifyResult.dtpc)*tVerifyResult.da) / float64(v.dtpc)
					}
				}
				tVerifyResult = v
				// check speed and data size downloaded
				if v.dls >= dltEvaluationSpeed && v.dlds > downloadSizeMin {
					// put v into verifyResultsMap
					verifyResultsMap[tVerifyResult.ip] = tVerifyResult
					// we have expected result
					if !testAll && len(verifyResultsMap) >= resultMin {
						haveEnoughResult = true
					}
					// non-debug msg
					displayDetails(true, true, []VerifyResults{tVerifyResult})
				} else if debug {
					// debug msg
					displayDetails(false, true, []VerifyResults{tVerifyResult})
				}
			}
			if debug {
				displayStat(overAllStat{
					dtTasksDone:  dtDoneTasks,
					dtOnGoing:    0,
					dtCached:     len(dtTaskCache),
					dltTasksDone: dltDoneTasks,
					dltOnGoing:   0,
					dltCached:    len(dltTaskCache),
					resultCount:  len(verifyResultsMap),
				})
			}
		}
		// cancel from terminal, or have enough results
		// flush ping and download task chan
		// MARK as REMOVE
		if cancelSigFromTerm {
			// show waiting msg, only when debug
			if debug && !showQuitWaiting {
				if tcellMode {
					printQuitWaiting()
				} else {
					myLogger.Debugln(titleWaitQuit)
				}
				showQuitWaiting = true
			}
			break LOOP
		}
		if haveEnoughResult {
			break LOOP
		}
	}
	// for tcell only, send terminate signal to termControl
	terminateConfirm = true
	// update statistic just before quit controller
	displayStat(overAllStat{
		dtTasksDone:  dtDoneTasks,
		dtOnGoing:    0,
		dtCached:     len(dtTaskCache),
		dltTasksDone: dltDoneTasks,
		dltOnGoing:   0,
		dltCached:    len(dltTaskCache),
		resultCount:  len(verifyResultsMap),
	})
	// close all chan
	close(dtTaskChan)
	close(dtResultChan)
	close(dltTaskChan)
	close(dltResultChan)
	wg.Wait()
}

func termControl(wg *sync.WaitGroup) {
	defer (*wg).Done()
	defer (*termAll).Fini()
LOOP:
	for !terminateConfirm {
		if !(*termAll).HasPendingEvent() {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		ev := (*termAll).PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape:
				if !terminateConfirm && !cancelSigFromTerm && confirmQuit() {
					cancelSigFromTerm = true
				}
				if terminateConfirm {
					break LOOP
				}
			default:
				if terminateConfirm {
					break LOOP
				}
			}
		case *tcell.EventResize:
			initScreen()
		}
	}
	printQuittingCountDown(quitWaitingTime)
}

func confirmQuit() bool {
	printCancelConfirm()
	for {
		ev := (*termAll).PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEnter:
				printQuitWaiting()
				return true
			default:
				printCancel()
				return false
			}
		case *tcell.EventResize:
			initScreen()
			printCancelConfirm()
		}
	}
}

func main() {

	// start controller worker
	runWorker()
	if len(verifyResultsMap) > 0 {
		verifyResultsSlice := make([]VerifyResults, 0)
		for _, v := range verifyResultsMap {
			if ResolveLoc && len(*v.loc) == 0 {
				t_loc := getGeoInfoFromCF(v.ip)
				v.loc = &t_loc
			}
			verifyResultsSlice = append(verifyResultsSlice, v)
		}
		if !silenceMode {
			records := genDBRecords(verifyResultsSlice, resolveLocalASNAndCity)
			// write to csv file
			if storeToFile {
				myLogger.Print("Write to csv " + resultFile)
				writeCSVResult(records, resultFile)
				myLogger.Println("  Done!")
			}
			// write to db
			if storeToDB {
				myLogger.Print("Write to sqlite3 db file " + dbFile)
				saveDBRecords(records, dbFile)
				myLogger.Println("  Done!")
			}
			// sort by speed
			sort.Sort(sort.Reverse(resultSpeedSorter(verifyResultsSlice)))
			myLogger.Println()
			myLogger.Println("All Results:")
			printFinalStat(verifyResultsSlice, dtOnly)
			// } else { // we display results in controler when in silence mode
			// 	for _, v := range verifyResultsSlice {
			// 		myLogger.Println(*(v.ip))
			// 	}
		}
	}
}
