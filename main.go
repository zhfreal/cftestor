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
	flag.BoolVar(&fastMode, "fast", false, "Fast mode")
	flag.StringSliceVarP(&ipStr, "ip", "s", []string{}, "Specific IP or CIDR for test.")
	flag.StringVarP(&ipFile, "in", "i", "", "Specific file of IPs and CIDRs for test.")

	flag.IntVarP(&dtWorkerThread, "dt-thread", "m", 20, "Number of concurrent threads for Delay Test(DT).")
	flag.IntVarP(&dtTimeout, "dt-timeout", "t", 2000, "Timeout for single DT(ms).")
	flag.IntVarP(&dtCount, "dt-count", "c", 4, "Tries of DT for a IP.")
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
	flag.IntVar(&dltTimeout, "dlt-timeout", 5000, "Specify the timeout for http response when do DLT in milliseconds, default 5000 ms.")
	flag.IntVarP(&interval, "interval", "I", 500, "Interval between two tests, unit ms, default 500ms.")

	flag.BoolVar(&enableDTEvaluation, "ev-dt", false, "Evaluate DT test result. Default as disabled")
	flag.IntVarP(&dtEvaluationDelay, "ev-dt-delay", "k", 600, "Delay for DT is beyond this one will be cause failure, unit ms, default 600ms.")
	flag.Float64Var(&dtEvaluationDTPR, "ev-dt-dtpr", 100, "The DT successful rate below this will be cause failure, default 100%.")
	flag.Float64Var(&dtStdExp, "ev-dt-std", 30, "expect standard deviation while do DT evaluation.")
	flag.Float64VarP(&dltEvaluationSpeed, "speed", "l", 6000, "Download speed should not less than this, Unit KB/s, default 6000KB/s.")
	flag.IntVar(&loop, "loop", -1, "Loop N round")
	flag.IntVar(&loopInterval, "loop-interval", 60, "sleep N second between two loop")
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

	flag.BoolVarP(&silenceMode, "silence", "S", false, "silence mode.")
	flag.BoolVarP(&debug, "debug", "V", false, "Print debug message.")
	// flag.BoolVar(&tcellMode, "tcell", false, "Use tcell form to show debug messages.")
	flag.BoolVarP(&printVersion, "version", "v", false, "Show version.")
	flag.Usage = func() {
		fmt.Print(help)
	}
	flag.Parse()
	if !silenceMode {
		print_version()
	} else {
		debug = false
		// tcellMode = false
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
		myLogger.Fatalln("Invalid value for \"--dt-via\". Please use one of: https, tls, or ssl.")
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
	// if tcellMode {
	// 	debug = true
	// }

	// initialize myLogger
	if silenceMode {
		loggerLevel = logLevelFatal
	} else {
		loggerLevel = logLevelInfo
		if debug {
			loggerLevel = logLevelDebug
		}
	}
	// if debug {
	// 	loggerLevel = logLevelDebug
	// } else {
	// 	if !silenceMode {
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
	// set ipv4Mode to false, when just --ipv6 provided
	v4Flag := flag.Lookup("ipv4")
	if (!v4Flag.Changed) && ipv6Mode {
		ipv4Mode = false
	}
	if ipv4Mode {
		tMode |= TypeIPv4
	}
	if ipv6Mode {
		tMode |= TypeIPv6
	}
	if tMode == TypeIPErr {
		myLogger.Fatalln("We can't disable both IPv4 and IPv6 at the same time!")
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

	// var srcIPS []*string
	// no source IPs provided
	if len(ipStr) == 0 && len(ipFile) == 0 {
		// it's invalid when ipv4Mode and ipv6Mode is both true or false at the same time and
		// no specified source IPs or file is provided
		if (tMode&TypeIPv4) == TypeIPv4 && (tMode&TypeIPv6) == TypeIPv6 {
			myLogger.Fatalln("The options \"-4|--ipv4\" and \"-6|--ipv6\" cannot be used together when no specific IPs or file are provided!")
			// not need to exit, because it's a fatal error
			//os.Exit(1)
		}
		if (tMode & TypeIPv4) == TypeIPv4 {
			t_cf_ipv4 := CFIPV4FULL
			if fastMode {
				t_cf_ipv4 = CFIPV4
			}
			err := srcIPs.AddFromSlice(t_cf_ipv4, TypeIPv4)
			if err != nil {
				myLogger.Fatalln(err)
			}
		} else {
			t_cf_ipv6 := CFIPV6FULL
			if fastMode {
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
		if len(ipFile) != 0 {
			err := srcIPs.AddFromFile(ipFile, tMode)
			if err != nil {
				myLogger.Fatalln(err)
			}
		}
		if srcIPs.LenInt() == 0 {
			myLogger.Fatalln("no source IPs provided!")
		}
		// set ipv6Mode to true when specific IPs are provided and neither "-4|--ipv4" nor ""-6|--ipv6" is provided
		v6Flag := flag.Lookup("ipv6")
		if !v6Flag.Changed && !v4Flag.Changed {
			ipv6Mode = true
		}
	}

	// check dtUrl is valid URL and uses HTTPS
	if !dltOnly && dtHttps {
		if len(dtUrl) == 0 {
			myLogger.Fatalf("\"-u|--dt-url %v\" should not be empty!\n", dtUrl)
		}
		t_url, err := url.Parse(dtUrl)
		if err != nil {
			myLogger.Fatalf("\"-u|--dt-url %v\" is not a valid URL!\n", dtUrl)
		}
		if t_url.Scheme != "https" {
			myLogger.Fatalf("\"-u|--dt-url %v\" should use HTTPS!\n", dtUrl)
		}
		if len(dtUrl) > 0 {
			dtUrl = t_url.String()
		}
	}
	// check dltUrl is valid URL and uses HTTPS
	if !dtOnly {
		if len(dltUrl) == 0 {
			myLogger.Fatalf("\"-d|--dlt-url %v\" should not be empty!\n", dltUrl)
		}
		t_url, err := url.Parse(dltUrl)
		if err != nil {
			myLogger.Fatalf("\"-d|--dlt-url %v\" is not a valid URL!\n", dltUrl)
		}
		if t_url.Scheme != "https" {
			myLogger.Fatalf("\"-d|--dlt-url %v\" should use HTTPS!\n", dltUrl)
		}
		if len(dltUrl) > 0 {
			dltUrl = t_url.String()
		}
	}

	// shuffle srcIPR and srcIPRsCache when do not testAll
	// and fix resultMin

	srcIPs.Shuffle()
	srcIPs.AddPorts(portStrSlice)
	t_qty := srcIPs.Len()
	// check resultMin
	if resultMin <= 0 {
		myLogger.Fatalf("\"-r|--result %v\" should not be smaller than 0!\n", resultMin)
	}
	// re-calculate resultMin based on the source IPs
	t_result_min := big.NewInt(int64(resultMin))
	if testAll {
		resultMin = -1
	} else {
		if t_qty.Cmp(t_result_min) == -1 {
			resultMin = int(t_qty.Int64())
		}
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

}

func validDTResult(tVerifyResult *VerifyResults) bool {
	if tVerifyResult.da > 0.0 &&
		tVerifyResult.da <= float64(dtEvaluationDelay) &&
		tVerifyResult.dtpr*100.0 >= float64(dtEvaluationDTPR) &&
		(!enableStdEv || (enableStdEv && tVerifyResult.daStd <= dtStdExp)) {
		return true
	}
	return false
}

func validDLTResult(tVerifyResult *VerifyResults) bool {
	if tVerifyResult.dls >= dltEvaluationSpeed && tVerifyResult.dlds > downloadSizeMin {
		return true
	}
	return false
}

func runWorker() {

	var wg sync.WaitGroup
	var dtTaskChan = make(chan *task, dtWorkerThread)
	var dtResultChan = make(chan singleVerifyResult, cap(dtTaskChan))
	var dltTaskChan = make(chan *task, dltWorkerThread)
	var dltResultChan = make(chan singleVerifyResult, cap(dltTaskChan))

	// if debug && tcellMode {
	// 	go termControl(&wg)
	// 	wg.Add(1)
	// }

	// showQuitWaiting := false

	if !dltOnly {
		for range dtWorkerThread {
			wg.Add(1)
			if dtHttps {
				go downloadWorkerNew(dtTaskChan, dtResultChan, &wg, &dtUrl, dtTimeoutDuration, dtCount, true)
			} else {
				go sslDTWorkerNew(dtTaskChan, dtResultChan, &wg)
			}
		}
	}
	if !dtOnly {
		for range dltWorkerThread {
			wg.Add(1)
			go downloadWorkerNew(dltTaskChan, dltResultChan, &wg, &dltUrl, httpRspTimeoutDuration, dltCount, false)
		}
	}
	tmpResultMap := make(map[string]VerifyResults)
	var tmpTestSlice map[string]bool
	var thisSourceIPs = CopySourceIPs(srcIPs)
	looper := NewSafeLooperWithInterval(loop, loopInterval*1000)
	// verifyResultsMap
LOOP:
	for {
		dtDoneTasks := 0
		// the item in dtTaskCache is a ip string.
		dtTaskCache := make([]*string, 0)
		dltDoneTasks := 0
		// the item in dltTaskCache is a ip string.
		dltTaskCache := make([]*string, 0)
		// the key of cachedMap should be: <ipv4:port> or <[ipv6]:port>, should not be just a ip string.
		cachedMap := make(map[string]VerifyResults)
		haveEnoughResult := false
		// reset ResultIPSlice while do LOOP
		tmpTestSlice = make(map[string]bool)
	SINGLE_ROUND:
		for {
			// DT
			if !dltOnly {
				if len(dtTaskCache) < dtWorkerThread {
					dtTaskCache = append(dtTaskCache, thisSourceIPs.RetrieveSome(dtWorkerThread, !testAll)...)
				}
				// no more sources for testing
				t_dt_sources_len := len(dtTaskCache)
				if t_dt_sources_len == 0 {
					break SINGLE_ROUND
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
				// 		resultCount:  len(tmpResultMap),
				// 	})
				// }
				// put task
				t_dt_task_size := min(dtWorkerThread, t_dt_sources_len)
				go func() {
					max_failure := get_max_failure(true)
					for _, taskIP := range dtTaskCache[:t_dt_task_size] {
						t := NewTask(taskIP, max_failure)
						dtTaskChan <- t
					}
					// dtTaskCache = make([]*string, 0)
				}()
				// retrieve from dtResultChan
				for range t_dt_task_size {
					dtResult := <-dtResultChan
					// if ip not test then put it into dltTaskChan
					dtDoneTasks += 1
					var tVerifyResult = calcResult(dtResult, false)
					if validDTResult(&tVerifyResult) {
						if !dtOnly { // there are download test ongoing
							// put ping test result to cachedMap for later
							cachedMap[*tVerifyResult.ip] = tVerifyResult
							dltTaskCache = append(dltTaskCache, tVerifyResult.ip)
							// debug msg, show only in debug mode
							if debug {
								displayDetails(false, looper.Status() > -1, []VerifyResults{tVerifyResult})
							}
						} else { // Download test disabled
							// update loc when --resolve-loc is enabled and it is in silent mode and we are not in LOOP
							if ResolveLoc && silenceMode && looper.Status() == -1 && (tVerifyResult.loc == nil || len(*tVerifyResult.loc) == 0) {
								loc := getGeoInfoFromCF(tVerifyResult.ip)
								tVerifyResult.loc = &loc
							}
							// combine tmpResultMap with tmpResultMap[*tVerifyResult.ip]
							v, ok := tmpResultMap[*tVerifyResult.ip]
							if !ok {
								tmpResultMap[*tVerifyResult.ip] = tVerifyResult
							} else {
								tVerifyResult.combine(v)
								tmpResultMap[*tVerifyResult.ip] = tVerifyResult
							}
							// non-debug msg
							displayDetails(false, looper.Status() > -1, []VerifyResults{tVerifyResult})
							tmpTestSlice[*tVerifyResult.ip] = true
							// we have expected result, break LOOP
							// TODO: this is a dirty way
							if !testAll && len(tmpTestSlice) >= resultMin {
								haveEnoughResult = true
							}
						}
					} else if debug {
						// update tmpResultMap[*tVerifyResult.ip] with tVerifyResult while in LOOP mode
						if looper.InLooping() {
							v, ok := tmpResultMap[*tVerifyResult.ip]
							if !ok {
								tmpResultMap[*tVerifyResult.ip] = tVerifyResult
							} else {
								tVerifyResult.combine(v)
								tmpResultMap[*tVerifyResult.ip] = tVerifyResult
							}
						}
						// debug msg
						displayDetails(false, looper.Status() > -1, []VerifyResults{tVerifyResult})
					}
				}
				// cut out the used source from dtTaskCache
				if t_dt_task_size < t_dt_sources_len {
					dtTaskCache = dtTaskCache[t_dt_task_size:]
				} else {
					dtTaskCache = make([]*string, 0)
				}
				if debug {
					displayStat(overAllStat{
						dtTasksDone:  dtDoneTasks,
						dtOnGoing:    0,
						dtCached:     len(dtTaskCache),
						dltTasksDone: dltDoneTasks,
						dltOnGoing:   0,
						dltCached:    len(dltTaskCache),
						resultCount:  len(tmpTestSlice),
						remain:       thisSourceIPs.LenInt(),
					})
				}
			}
			//DLT
			if !dtOnly {
				// no source to do DLT
				if len(dltTaskCache) <= dltWorkerThread {
					// DT enabled, just continue to do DT
					if !dltOnly {
						if len(dltTaskCache) == 0 {
							continue
						}
					} else {
						// retrieve source IP
						dltTaskCache = append(dltTaskCache, thisSourceIPs.RetrieveSome(dltWorkerThread, !testAll)...)
						// no source IP, break LOOP
						if len(dltTaskCache) == 0 {
							break SINGLE_ROUND
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
				// 		resultCount:  len(tmpResultMap),
				// 		remain:       thisSourceIPs.LenInt(),
				// 	})
				// }
				// put task
				t_dlt_task_size := dltWorkerThread
				if t_dlt_sources_len < t_dlt_task_size || !dltOnly {
					t_dlt_task_size = t_dlt_sources_len
				}
				go func() {
					max_failure := get_max_failure(false)
					for _, taskIP := range dltTaskCache[:t_dlt_task_size] {
						t := NewTask(taskIP, max_failure)
						dltTaskChan <- t
					}
					// dltTaskCache = make([]*string, 0)
				}()
				// retrieve result
				for range t_dlt_task_size {
					out := <-dltResultChan
					dltDoneTasks += 1
					var tVerifyResult = calcResult(out, true)
					if !dltOnly {
						v := cachedMap[*tVerifyResult.ip]
						// reset TestTime according download test result
						tVerifyResult.combine(v)
						cachedMap[*tVerifyResult.ip] = tVerifyResult
					}
					// tVerifyResult = v
					// check speed and data size downloaded
					if validDLTResult(&tVerifyResult) && (dltOnly || (!dltOnly && validDTResult(&tVerifyResult))) {
						// update loc when --resolve-loc is enabled and it is in silent mode and we are not in LOOP
						if ResolveLoc && silenceMode && looper.Status() == -1 && (tVerifyResult.loc == nil || len(*tVerifyResult.loc) == 0) {
							loc := getGeoInfoFromCF(tVerifyResult.ip)
							tVerifyResult.loc = &loc
						}
						// combine tmpResultMap with tmpResultMap[*tVerifyResult.ip]
						v, ok := tmpResultMap[*tVerifyResult.ip]
						if !ok {
							tmpResultMap[*tVerifyResult.ip] = tVerifyResult
						} else {
							tVerifyResult.combine(v)
							tmpResultMap[*tVerifyResult.ip] = tVerifyResult
						}
						tmpTestSlice[*tVerifyResult.ip] = true
						// we have expected result
						if !testAll && len(tmpTestSlice) >= resultMin {
							haveEnoughResult = true
						}
						displayDetails(true, looper.Status() > -1, []VerifyResults{tVerifyResult})
					} else {
						if debug {
							displayDetails(true, looper.Status() > -1, []VerifyResults{tVerifyResult})
						}
					}

				}
				// ut out the used source from dltTaskCache
				if t_dlt_task_size < t_dlt_sources_len {
					dltTaskCache = dltTaskCache[t_dlt_task_size:]
				} else {
					dltTaskCache = make([]*string, 0)
				}
				if debug {
					displayStat(overAllStat{
						dtTasksDone:  dtDoneTasks,
						dtOnGoing:    0,
						dtCached:     len(dtTaskCache),
						dltTasksDone: dltDoneTasks,
						dltOnGoing:   0,
						dltCached:    len(dltTaskCache),
						resultCount:  len(tmpTestSlice),
						remain:       thisSourceIPs.LenInt(),
					})
				}
			}
			if haveEnoughResult {
				break SINGLE_ROUND
			}
		}
		// if there is no target IP after initial round, break LOOP
		if len(tmpResultMap) == 0 {
			break LOOP
		}
		// enter loop round after initial round
		ok := looper.Loop()
		if !ok {
			// looper is not valid or finished, break LOOP
			break LOOP
		} else { // looper is ready or in progress
			// do thisSourceIPs reset and set new source ips before loop
			tmp_slice := make([]string, 0)
			for k := range tmpResultMap {
				tmp_slice = append(tmp_slice, k)
			}
			thisSourceIPs.Reset()
			thisSourceIPs.AddFromSlice(tmp_slice, TypeIPv4|TypeIPv6)
			thisSourceIPs.AddPorts(portStrSlice)
			// set resultMin to len(tmpTestSlice) while we don't do testAll
			// to perform testing all target in tmpTestSlice
			if !testAll {
				resultMin = len(tmp_slice)
			}
			looper_interval := float64(looper.GetInterval()) / 1000.0
			next_round := looper.GetRound()
			myLogger.Debugf("sleep %v seconds for loop round %d\n", looper_interval, next_round)
			looper.Sleep()
		}
	}
	for tIP := range tmpTestSlice {
		tr := tmpResultMap[tIP]
		isValid := true
		if !dltOnly && !validDTResult(&tr) {
			isValid = false
		}
		if !dtOnly && !validDLTResult(&tr) {
			isValid = false
		}
		if isValid {
			verifyResultsMap[tIP] = tmpResultMap[tIP]
		}
	}
	// update statistic just before quit controller
	// displayStat(overAllStat{
	// 	dtTasksDone:  0,
	// 	dtOnGoing:    0,
	// 	dtCached:     0,
	// 	dltTasksDone: 0,
	// 	dltOnGoing:   0,
	// 	dltCached:    0,
	// 	resultCount:  len(verifyResultsMap),
	// 	remain:       0,
	// })
	// close all chan
	close(dtTaskChan)
	close(dtResultChan)
	close(dltTaskChan)
	close(dltResultChan)
	wg.Wait()
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
		var records []DBRecord
		if storeToFile || storeToDB {
			records = genDBRecords(verifyResultsSlice, resolveLocalASNAndCity)
			// write to csv file
			if storeToFile {
				if !silenceMode {
					myLogger.Print("Write to csv " + resultFile)
				}
				writeCSVResult(records, resultFile)
				if !silenceMode {
					myLogger.Println("  Done!")
				}
			}
			// write to db
			if storeToDB {
				if !silenceMode {
					myLogger.Print("Write to sqlite3 db file " + dbFile)
				}
				saveDBRecords(records, dbFile)
				if !silenceMode {
					myLogger.Println("  Done!")
				}
			}
		}
		// sort by speed
		sort.Sort(sort.Reverse(resultSpeedSorter(verifyResultsSlice)))
		if !silenceMode {
			myLogger.Println()
			myLogger.Println("All Results:")
			printFinalStat(verifyResultsSlice, dtOnly, false)
		} else {
			if loop > 0 {
				printFinalStat(verifyResultsSlice, dtOnly, true)
			}
		}
	}
}
