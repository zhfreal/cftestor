package main

import (
	"bufio"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"net"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	flag "github.com/spf13/pflag"
)

const (
	workerStopSignal        = "0"
	workOnGoing             = 1
	controllerInterval      = 100               // in millisecond
	statisticIntervalT      = 1000              // in millisecond, valid in tcell mode
	statisticIntervalNT     = 10000             // in millisecond, valid in non-tcell mode
	quitWaitingTime         = 3                 // in second
	downloadBufferSize      = 1024 * 16         // in byte
	fileDefaultSize         = 1024 * 1024 * 300 // in byte
	downloadSizeMin         = 1024 * 1024       // in byte
	defaultDLTUrl           = "https://cf.9999876.xyz/500mb.dat"
	defaultDTUrl            = "https://cf.9999876.xyz/test.dat"
	userAgent               = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
	defaultDBFile           = "ip.db"
	DefaultTestHost         = "cf.9999876.xyz"
	maxHostLen              = 1 << 16
	dtsSSL                  = "SSL"
	dtsHTTPS                = "HTTPS"
	runTime                 = "cftestor"
	retrieveCount       int = 100
)

var (
	maxHostLenBig                           = big.NewInt(maxHostLen)
	ipFile                                  string
	version, buildTag, buildDate, buildHash string     = "dev", "dev", "dev", "dev"
	srcIPRsRaw                              []*ipRange // CIDR slice
	srcIPRsExtracted                        []net.IP   // net.IP slice
	srcHosts                                []*string  // slice stored host: <ip>:<port>
	ipStr                                   arrayFlags
	dtCount, dtWorkerThread                 int
	ports                                   []int
	dltDurMax, dltWorkerThread              int
	dltCount, resultMin                     int
	interval, dtEvaluationDelay, dtTimeout  int
	hostName, dltUrl, dtSource, dtUrl       string
	dltTimeout                              int
	dtEvaluationDTPR, dltEvaluationSpeed    float64
	dtHttps, disableDownload                bool
	dtVia                                   string
	enableDTEvaluation                      bool
	ipv4Mode, ipv6Mode, dtOnly, dltOnly     bool
	storeToFile, storeToDB, testAll, debug  bool
	resultFile, suffixLabel, dbFile         string
	myLogger                                MyLogger
	loggerLevel                             LogLevel
	httpRspTimeoutDuration                  time.Duration
	dtTimeoutDuration                       time.Duration
	dltTimeDurationMax                      time.Duration
	verifyResultsMap                        = make(map[*string]VerifyResults)
	defaultASN                              = 0
	defaultCity                             = ""
	myRand                                  = rand.New(rand.NewSource(0))
	titleRuntime                            *string
	titlePre                                [2][4]string
	titleTasksStat                          [2]*string
	detailTitleSlice                        []string
	resultStrSlice, debugStrSlice           [][]*string
	termAll                                 *tcell.Screen
	titleStyle                              = tcell.StyleDefault.Foreground(tcell.ColorBlack.TrueColor()).Background(tcell.ColorWhite)
	normalStyle                             = tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	titleStyleCancel                        = tcell.StyleDefault.Foreground(tcell.ColorBlack.TrueColor()).Background(tcell.ColorGray)
	contentStyle                            = tcell.StyleDefault
	maxResultsDisplay                       = 10
	maxDebugDisplay                         = 10
	titleRuntimeRow                         = 0
	titlePreRow                             = titleRuntimeRow + 2
	titleCancelRow                          = titlePreRow + 3
	titleTasksStatRow                       = titleCancelRow + 2
	titleResultHintRow                      = titleTasksStatRow + 2
	titleResultRow                          = titleResultHintRow + 1
	titleDebugHintRow                       = titleResultRow + maxResultsDisplay + 2
	titleDebugRow                           = titleDebugHintRow + 1
	titleCancel                             = "Press ESC to cancel!"
	titleCancelConfirm                      = "Press ENTER to confirm; Any other key to back!"
	titleWaitQuit                           = "Waiting for exit..."
	titleResultHint                         = "Result:"
	titleDebugHint                          = "Debug Msg:"
	cancelSigFromTerm                       = false
	terminateConfirm                        = false
	resultStatIndent                        = 9
	dtThreadsNumLen, dltThreadsNumLen       = 0, 0
	tcellMode                               = false
	fastMode                                = false
	silenceMode                             = false
	statInterval                            = statisticIntervalNT
	// titleExitHint                          = "Press any key to exit!"
	appArt string = `
  ░█▀▀░█▀▀░▀█▀░█▀▀░█▀▀░▀█▀░█▀█░█▀▄
  ░█░░░█▀▀░░█░░█▀▀░▀▀█░░█░░█░█░█▀▄
  ░▀▀▀░▀░░░░▀░░▀▀▀░▀▀▀░░▀░░▀▀▀░▀░▀
`
)

var help = `Usage: ` + runTime + ` [options]
options:
    -s, --ip           string  Specific IP, CIDR or host for test. E.g.: "-s 1.0.0.1", "-s 1.0.0.1/32",
                               "-s 1.0.0.1/24", "-s 1.1.1.1:443".
    -i, --in           string  Specify file for test, which contains multiple lines. Each line
                               represent one IP or CIDR.
    -p, --port         int     Port to test, could be specific one or more ports at same time,
                               The port should be working via SSL/TLS/HTTPS protocol,  default 443.
    -m, --dt-thread    int     Number of concurrent threads for Delay Test(DT). How many IPs can 
                               be perform DT at the same time. Default 20 threads.
    -t, --dt-timeout   int     Timeout for single DT, unit ms, default 1000ms. A single SSL/TLS 
                               or HTTPS request and response should be finished before timeout. 
                               It should not be less than "-k|--evaluate-dt-delay", It should be 
                               longer when we perform https connections test by "-dt-via-https" 
                               than when we perform SSL/TLS test by default.
    -c, --dt-count     int     Tries of DT for a IP, default 4.
        --hostname     string  Hostname for DT test. It's valid when "--dt-only" is no and "--dt-via-https" 
                               is not provided.
        --dt-via https|tls|ssl DT via https or SSL/TLS shaking hands, "--dt-via <https|tls|ssl>"
                               default https.
        --dt-url       string  Specify test URL for DT.
        --ev-dt                Evaluate DT, we'll try "-c|--dt-count value" to evaluate delay;
                               if we don't turn this on, we'll stop DT after we got the first
                               successfull DT; if we turn this on, we'll evaluate the test result 
                               through average delay of singe DT and statistic of all successfull
                               DT by these two thresholds  "-k|--evaluate-dt-delay value" and 
                               "-S|--evaluate-dt-dtpr value". default turn off.
    -k, --ev-dt-delay  int     single DT's delay should not bigger than this, unit ms, default 600ms.
    -S, --ev-dt-dtpr   float   The DT pass rate should not lower than this, default 100, means 100%, all
                               DT must be below "-k|--evaluate-dt-delay value".
    -n, --dlt-thread   int     Number of concurrent Threads for Download Test(DLT), default 1. 
                               How many IPs can be perform DLT at the same time.
    -d, --dlt-period   int     The total times escaped for single DLT, default 10s.
    -b, --dlt-count    int     Tries of DLT for a IP, default 1.
    -u, --dlt-url      string  Specify test URL for DLT.
        --dlt-timeout  int     Specify the timeout for http response when do DLT. In ms, default as 5000 ms.
    -I  --interval     int     Interval between two tests, unit ms, default 500ms.

    -l, --speed        float   Download speed filter, Unit KB/s, default 6000KB/s. After DLT, it's 
                               qualified when its speed is not lower than this value.
    -r, --result       int     The total IPs qualified limitation, default 10. The Process will stop 
                               after it got equal or more than this indicated. It would be invalid if
                               "--test-all" was set.
        --dt-only              Do DT only, we do DT & DLT at the same time by default.
        --dlt-only             Do DLT only, we do DT & DLT at the same time by default.
        --fast                 Fast mode, use inner IPs for fast detection. Just when neither"-s|--ip value"
                               nor "-i|--in value" is provided, and this flag is provided. It will be working
                               Disabled by default.
    -4, --ipv4                 Just test IPv4. When we don't specify IPs by "-s|--ip value" and "-i|--in value",
                               then it will do IPv4 test from build-in IPs from CloudFlare by default.
    -6, --ipv6                 Just test IPv6. When we don't specify IPs by "-s|--ip value" and "-i|--in value",
                               then it will do IPv6 test from build-in IPs from CloudFlare by using
                               this flag.
    -a  --test-all             Test all IPs until no more IP left. It's disabled by default. 
    -w, --to-file              Write result to csv file, disabled by default. If it is provided and 
                               "-o|--result-file" is not provided, the result file will be named
                               as "Result_<YYYYMMDDHHMISS>-<HOSTNAME>.csv" and be stored in current DIR.
    -o, --outfile      string  File name of result. If it don't provided and "-w|--store-to-file"
                               is provided, the result file will be named as 
                               "Result_<YYYYMMDDHHMISS>-<HOSTNAME>.csv" and be stored in current DIR.
    -e, --to-db                Write result to sqlite3 db file, disabled by default. If it's provided
                               and "-f|--db-file" is not provided, it will be named "ip.db" and
                               store in current directory.
    -f, --dbfile       string  Sqlite3 db file name. If it's not provided and "-e|--store-to-db" is
                               provided, it will be named "ip.db" and store in current directory.
    -g, --label        string  the label for a part of the result file's name and sqlite3 record. It's 
                               hostname from "--hostname" or "-u|--url" by default.
        --silence              Silence mode.
    -V, --debug                Print debug message.
        --tcell                Use tcell to display the running procedure when in debug mode.
                               Turn this on will activate "--debug".
    -v, --version              Show version.
`

func print_version() {
	fmt.Println(appArt)
	fmt.Println(`  CF CDN IP scanner, find best IPs for your Cloudflare CDN applications.
  https://github.com/zhfreal/cftestor`)
	fmt.Println()
	fmt.Printf("Version: %v\n", version)
	fmt.Printf("BuildOn: %v\n", buildDate)
	fmt.Printf("BuildTag: %v\n", buildTag)
	fmt.Printf("BuildFrom: %v\n", buildHash)
	fmt.Println()
}

func init() {
	var printVersion bool

	// version = "dev"
	flag.BoolVar(&fastMode, "fast", false, "Fast mode")
	flag.VarP(&ipStr, "ip", "s", "Specific IP or CIDR for test.")
	flag.StringVarP(&ipFile, "in", "i", "", "Specific file of IPs and CIDRs for test.")

	flag.IntVarP(&dtWorkerThread, "dt-thread", "m", 20, "Number of concurrent threads for Delay Test(DT).")
	flag.IntVarP(&dtTimeout, "dt-timeout", "t", 2000, "Timeout for single DT(ms).")
	flag.IntVarP(&dtCount, "dt-count", "c", 4, "Tries of DT for a IP.")
	// flag.IntVarP(&port, "port", "p", 443, "Port to test")
	flag.IntSliceVarP(&ports, "port", "p", []int{443}, "Port to test, could be specific one or more ports at same time.")
	flag.StringVar(&hostName, "hostname", DefaultTestHost, "Hostname for DT test.")
	flag.StringVar(&dtVia, "dt-via", "https", "DT via https rather than SSL/TLS shaking hands.")
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
	flag.Float64VarP(&dltEvaluationSpeed, "speed", "l", 6000, "Download speed should not less than this, Unit KB/s, default 6000KB/s.")
	flag.IntVarP(&resultMin, "result", "r", 10, "The total IPs qualified limitation, default 10")

	flag.BoolVar(&disableDownload, "disable-download", false, "Deprecated, use --dt-only instead.")
	flag.BoolVar(&dtOnly, "dt-only", false, "Do DT only, we do DT & DLT at the same time by default.")
	flag.BoolVar(&dltOnly, "dlt-only", false, "Do DLT only, we do DT & DLT at the same time by default.")
	flag.BoolVarP(&ipv4Mode, "ipv4", "4", true, "Just test IPv4.")
	flag.BoolVarP(&ipv6Mode, "ipv6", "6", false, "Just test IPv6.")
	flag.BoolVarP(&testAll, "test-all", "a", false, "Test all IPs until no more IP left.")

	flag.BoolVarP(&storeToFile, "to-file", "w", false, "Write result to csv file, disabled by default.")
	flag.StringVarP(&resultFile, "outfile", "o", "", "File name of result. ")
	flag.BoolVarP(&storeToDB, "to-db", "e", false, "Write result to sqlite3 db file.")
	flag.StringVarP(&dbFile, "dbfile", "f", "", "Sqlite3 db file name.")
	flag.StringVarP(&suffixLabel, "label", "g", "", "the label for a part of the result file's name and sqlite3 record.")

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
	dtVia = strings.ToLower(dtVia)
	if dtVia == "https" {
		dtHttps = true
	} else if dtVia == "ssl" || dtVia == "tls" {
		dtHttps = false
	} else {
		println("invalid value found! Please use \"--dt-via <https|tls|ssl>\"!")
		os.Exit(1)
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
				srcIPS = append(srcIPS, &CFIPV4[i])
			}
		} else {
			t_cf_ipv6 := CFIPV6FULL
			if fastMode {
				t_cf_ipv6 = CFIPV6
			}
			for i := 0; i < len(t_cf_ipv6); i++ {
				srcIPS = append(srcIPS, &CFIPV6[i])
			}
		}
	}

	// shuffle srcIPR and srcIPRsCache when do not testAll
	// and fix resultMin
	t_qty := big.NewInt(0)
	for i := 0; i < len(srcIPS); i++ {
		ips := strings.TrimSpace(*srcIPS[i])
		if isValidIPs(ips) {
			ipr := NewIPRangeFromCIDR(&ips)
			if ipr == nil {
				myLogger.Fatalf("\"%v\" is invalid!\n", ips)
			}
			// when it do not testAll and ipr is not bigger than maxHostLenBig, extract to to cache
			t_qty = t_qty.Add(t_qty, ipr.Len)
			if !testAll && ipr.Len.Cmp(maxHostLenBig) < 1 {
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
	// validate ports
	for _, port := range ports {
		if port < 1 || port > 65535 {
			myLogger.Fatalf("\"-p|--port %v\" is invalid!\n", port)
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
		if enableDTEvaluation {
			if dtEvaluationDelay <= 0 {
				myLogger.Fatalf("\"-k|--evaluate-dt-delay %v\" should not be smaller than 0!\n", dtEvaluationDelay)
			}
			if dtTimeout < dtEvaluationDelay {
				timeoutFlag := flag.Lookup("dt-timeout")
				// reset dtTimeout, when dtTimeout less than delayMax and did not set value of dtTimeout from cmdline
				if !timeoutFlag.Changed {
					dtTimeout = dtEvaluationDelay + int(dtEvaluationDelay/2)
				} else {
					myLogger.Warning(fmt.Sprintf("\"-t|--dt-timeout\" - %v is less than \"-k|--evaluate-dt-delay\" - %v. This will led to failure for some test!", dtTimeout, dtEvaluationDelay))
					if !confirm("Continue?", 3) {
						os.Exit(0)
					}
				}
			}
		}
		dtTimeoutDuration = time.Duration(dtTimeout) * time.Millisecond
		// if we ping via ssl negotiation and don't perform download test, we need check hostname and port
		if !dtHttps {
			//ping via ssl negotiation
			if len(hostName) == 0 {
				myLogger.Fatal("\"--hostname\" should not be empty. \n")
			}
			dtSource = dtsSSL
		} else {
			// check dtUrl is valid or not by ParseUrl() and set suffixLabel
			suffixLabel, _ = ParseUrl(dtUrl)
			dtSource = dtsHTTPS
		}
		dtThreadsNumLen = len(strconv.Itoa(dtWorkerThread))
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
		suffixLabel, _ = ParseUrl(dltUrl)
		httpRspTimeoutDuration = time.Duration(dltTimeout) * time.Millisecond
		dltTimeDurationMax = time.Duration(dltDurMax) * time.Second
		dltThreadsNumLen = len(strconv.Itoa(dltWorkerThread))
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
		statInterval = statisticIntervalT
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

// :param: dtTaskChan, dltTaskChan, every item in them should be: <ipv4:port> or <[ipv6]:port>, should not be just a ip string.
// :param: dtResultChan, dltResultChan, should have port in every single item
func controllerWorker(dtTaskChan chan *string, dtResultChan chan singleVerifyResult, dltTaskChan chan *string,
	dltResultChan chan singleVerifyResult, wg *sync.WaitGroup, dtOnGoingChan chan int, dltOnGoingChan chan int) {
	defer func() {
		// send terminate signal to
		terminateConfirm = true
		(*wg).Done()
	}()
	dtTasks := 0
	dltTasks := 0
	dtDoneTasks := 0
	// the item in dtTaskCache is a ip string.
	dtTaskCache := make([]*string, 0)
	dltDoneTasks := 0
	// the item in dltTaskCache is a ip string.
	dltTaskCache := make([]*string, 0)
	// the key of cacheResultMap should be: <ipv4:port> or <[ipv6]:port>, should not be just a ip string.
	cacheResultMap := make(map[string]VerifyResults)
	haveEnoughResult := false
	noMoreSourcesDT := false
	noMoreSourcesDLT := false
	OverAllStatTimer := time.Now()
	showQuitWaiting := false

LOOP:
	for {
		// cancel from terminal, or have enough results
		// flush ping and download task chan
		if cancelSigFromTerm || haveEnoughResult {
			if !dltOnly {
				for len(dtTaskChan) > 0 {
					<-(dtTaskChan)
					dtTasks--
				}
				dtTaskCache = []*string{}
			}
			if !dtOnly {
				for len(dltTaskChan) > 0 {
					<-(dltTaskChan)
					dltTasks--
				}
				dltTaskCache = []*string{}
			}
			// show waiting msg, only when debug
			if debug && !showQuitWaiting {
				if tcellMode {
					printQuitWaiting()
				} else {
					myLogger.Debugln(titleWaitQuit)
				}
				showQuitWaiting = true
			}
		}
		// DT
		if !dltOnly {
			// check ping test result
			for len(dtResultChan) > 0 {
				select {
				case dtResult := <-dtResultChan:
					// if ip not test then put it into dltTaskChan
					dtDoneTasks += 1
					var tVerifyResult = singleResultStatistic(dtResult, false)
					if tVerifyResult.da > 0.0 && tVerifyResult.da <= float64(dtEvaluationDelay) && tVerifyResult.dtpr*100.0 >= float64(dtEvaluationDTPR) {
						if !dtOnly { // there are download test ongoing
							// put ping test result to cacheResultMap for later
							cacheResultMap[*tVerifyResult.ip] = tVerifyResult
							dltTaskCache = append(dltTaskCache, tVerifyResult.ip)
							// debug msg, show only in debug mode
							if debug {
								displayDetails(false, []VerifyResults{tVerifyResult})
							}
						} else { // Download test disabled
							// non-debug msg
							displayDetails(true, []VerifyResults{tVerifyResult})
							verifyResultsMap[tVerifyResult.ip] = tVerifyResult
							// we have expected result, break LOOP
							if !testAll && len(verifyResultsMap) >= resultMin {
								haveEnoughResult = true
							}
						}
					} else if debug {
						// debug msg
						displayDetails(false, []VerifyResults{tVerifyResult})
					}
				default:
				}
				// Print overall stat during waiting time and reset OverAllStatTimer
				if time.Since(OverAllStatTimer) > time.Duration(statInterval)*time.Millisecond {
					displayStat(overAllStat{
						dtTasksDone:  dtDoneTasks,
						dtOnGoing:    len(dtOnGoingChan),
						dtCached:     len(dtTaskCache) + len(dtTaskChan),
						dltTasksDone: dltDoneTasks,
						dltOnGoing:   len(dltOnGoingChan),
						dltCached:    len(dltTaskCache) + len(dltTaskChan),
						resultCount:  len(verifyResultsMap),
					})
					OverAllStatTimer = time.Now()
				}
			}
			// DT task control, when it have enough source ip, don't get cancel signal from term,
			// don't result as expected, and the task chan is not full
			if !noMoreSourcesDT && !cancelSigFromTerm && !haveEnoughResult {
				if len(dtTaskChan) < cap(dtTaskChan) { // this condition is not apply for #line 587
					// get more Hosts while we don't have enough hosts in dtTaskCache
					if len(dtTaskCache) == 0 {
						t_dtTaskCache := retrieveIPsFromIPR(2 * dtWorkerThread)
						for _, ipStr := range t_dtTaskCache {
							for _, port := range ports {
								host := genHostFromIPStrPort(*ipStr, port)
								dtTaskCache = append(dtTaskCache, &host)
							}
						}
						t_dtTaskCache_2 := retrieveHosts(2 * dtWorkerThread)
						dtTaskCache = append(dtTaskCache, t_dtTaskCache_2...)
						// if no more hosts, but just in dt-only mode, we set noMoSources to true
						if len(t_dtTaskCache) == 0 && len(t_dtTaskCache_2) == 0 {
							noMoreSourcesDT = true
						}
						t_dtTaskCache = nil
						t_dtTaskCache_2 = nil
					}
					// when it's dt-only mode or, download task pool has less ip than 2*cap(dltTaskChan)
					// we put ping task into dtTaskCache
					// simplify algorithm
					if dtOnly || len(dltTaskCache) < 2*cap(dltTaskChan) {
						for len(dtTaskCache) > 0 &&
							len(dtTaskChan) < cap(dtTaskChan) && // dtTaskChan has enough room
							len(dtTaskChan)+len(dtOnGoingChan)+len(dtResultChan) < cap(dtResultChan) {
							// to prevent overflow of dtResultChan
							// the total IP and task in dtTaskChan, dtOnGoingChan and dtResultChan is less than the capacity of dtResultChan
							dtTaskChan <- dtTaskCache[0]
							dtTasks += 1
							if len(dtTaskCache) > 1 {
								dtTaskCache = dtTaskCache[1:]
							} else {
								dtTaskCache = []*string{}
							}
						}
					}
				}
			} else if dtOnly && // mission control
				len(dtOnGoingChan) == 0 &&
				len(dtTaskCache) == 0 &&
				len(dtTaskChan) == 0 &&
				dtDoneTasks >= dtTasks { // we did all ping works in dt-only mode, "dtDoneTasks >= dtTasks", make sure all DT tasks did done.
				break LOOP
			}
		}
		// DLT
		if !dtOnly {
			for len(dltResultChan) > 0 {
				select {
				// check download result
				case out := <-dltResultChan:
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
						displayDetails(true, []VerifyResults{tVerifyResult})
					} else if debug {
						// debug msg
						displayDetails(false, []VerifyResults{tVerifyResult})
					}
				default:
				}
				// Print overall stat during waiting time and reset OverAllStatTimer
				if time.Since(OverAllStatTimer) > time.Duration(statInterval)*time.Millisecond {
					displayStat(overAllStat{
						dtTasksDone:  dtDoneTasks,
						dtOnGoing:    len(dtOnGoingChan),
						dtCached:     len(dtTaskCache) + len(dtTaskChan),
						dltTasksDone: dltDoneTasks,
						dltOnGoing:   len(dltOnGoingChan),
						dltCached:    len(dltTaskCache) + len(dltTaskChan),
						resultCount:  len(verifyResultsMap),
					})
					OverAllStatTimer = time.Now()
				}
			}
			// DLT task control, when it don't get cancel signal from term, don't result as expected
			if !cancelSigFromTerm && !haveEnoughResult && ((!dltOnly && len(dltTaskCache) > 0) || (dltOnly && !noMoreSourcesDLT)) {
				// get more hosts while it's on download-only mode
				if dltOnly && len(dltTaskCache) == 0 {
					t_dltTaskCache := retrieveIPsFromIPR(2 * dltWorkerThread)
					for _, ips := range t_dltTaskCache {
						for _, port := range ports {
							host := genHostFromIPStrPort(*ips, port)
							dltTaskCache = append(dltTaskCache, &host)
						}
					}
					t_dltTaskCache_2 := retrieveHosts(2 * dltWorkerThread)
					dltTaskCache = append(dltTaskCache, t_dltTaskCache_2...)
					// if no more hosts, but just in dlt-only mode, we set noMoSources to true
					if len(t_dltTaskCache) == 0 && len(t_dltTaskCache_2) == 0 {
						noMoreSourcesDT = true
					}
					t_dltTaskCache = nil
					t_dltTaskCache_2 = nil
				}
				// put task to download chan when we have IPs from delay test and the task chan have empty slot
				for len(dltTaskCache) > 0 && // it has IP in dltTaskCache
					len(dltTaskChan) < cap(dltTaskChan) && // dltTaskChan is not full
					len(dltTaskChan)+len(dltOnGoingChan)+len(dltResultChan) < cap(dltResultChan) {
					// to prevent overflow of dltResultChan
					// the total IP and task in dltTaskChan, dltOnGoingChan and dltResultChan is less than the capacity of dltResultChan
					dltTaskChan <- dltTaskCache[0]
					dltTasks += 1
					if len(dltTaskCache) > 1 {
						dltTaskCache = dltTaskCache[1:]
					} else {
						dltTaskCache = []*string{}
					}
				}
			} else if len(dltOnGoingChan) == 0 && // mission control
				len(dltTaskChan) == 0 &&
				len(dltTaskCache) == 0 &&
				dltDoneTasks >= dltTasks && // "dltDoneTasks >= dltTasks", make sure all DLT tasks did done.
				(dltOnly ||
					(len(dtOnGoingChan) == 0 &&
						len(dtTaskChan) == 0 &&
						len(dtTaskCache) == 0) &&
						dtDoneTasks >= dtTasks) { // "dtDoneTasks >= dtTasks", make sure all DT tasks did done.
				break LOOP
			}
		}
		// Print overall stat during waiting time and reset OverAllStatTimer
		if time.Since(OverAllStatTimer) > time.Duration(statInterval)*time.Millisecond {
			displayStat(overAllStat{
				dtTasksDone:  dtDoneTasks,
				dtOnGoing:    len(dtOnGoingChan),
				dtCached:     len(dtTaskCache) + len(dtTaskChan),
				dltTasksDone: dltDoneTasks,
				dltOnGoing:   len(dltOnGoingChan),
				dltCached:    len(dltTaskCache) + len(dltTaskChan),
				resultCount:  len(verifyResultsMap),
			})
			OverAllStatTimer = time.Now()
		}
		time.Sleep(time.Duration(controllerInterval) * time.Millisecond)
	}
	// update statistic just before quit controller
	displayStat(overAllStat{
		dtTasksDone:  dtDoneTasks,
		dtOnGoing:    len(dtOnGoingChan),
		dtCached:     len(dtTaskCache) + len(dtTaskChan),
		dltTasksDone: dltDoneTasks,
		dltOnGoing:   len(dltOnGoingChan),
		dltCached:    len(dltTaskCache) + len(dltTaskChan),
		resultCount:  len(verifyResultsMap),
	})
	// put stop signal to all delay test workers and download worker
	if !dltOnly {
		for i := 0; i < dtWorkerThread; i++ {
			tStop := workerStopSignal
			dtTaskChan <- &tStop
		}
	}
	if !dtOnly {
		for i := 0; i < dltWorkerThread; i++ {
			tStop := workerStopSignal
			dltTaskChan <- &tStop
		}
	}
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
	var wg sync.WaitGroup
	var dtTaskChan = make(chan *string, dtWorkerThread*len(ports))
	var dtResultChan = make(chan singleVerifyResult, cap(dtTaskChan)*2)
	var dtOnGoingChan = make(chan int, dtWorkerThread)
	var dltTaskChan = make(chan *string, dltWorkerThread*len(ports))
	var dltResultChan = make(chan singleVerifyResult, cap(dltTaskChan)*2)
	var dltOnGoingChan = make(chan int, dltWorkerThread)

	if debug && tcellMode {
		go termControl(&wg)
		wg.Add(1)
	}

	// start controller worker
	go controllerWorker(dtTaskChan, dtResultChan, dltTaskChan, dltResultChan, &wg, dtOnGoingChan, dltOnGoingChan)

	wg.Add(1)
	// start ping worker
	if !dltOnly {
		for i := 0; i < dtWorkerThread; i++ {
			if dtHttps {
				go downloadWorker(dtTaskChan, dtResultChan, dtOnGoingChan, &wg, &dtUrl,
					dtTimeoutDuration, -1, dtCount, interval, true, enableDTEvaluation)
			} else {
				go sslDTWorker(dtTaskChan, dtResultChan, dtOnGoingChan, &wg, &hostName,
					dtTimeoutDuration, dtCount, interval, enableDTEvaluation)
			}
			wg.Add(1)
		}
	}

	// start download worker if don't do ping only
	if !dtOnly {
		for i := 0; i < dltWorkerThread; i++ {
			go downloadWorker(dltTaskChan, dltResultChan, dltOnGoingChan, &wg, &dltUrl,
				httpRspTimeoutDuration, dltTimeDurationMax, dltCount, interval, false, false)
			wg.Add(1)
		}
	}
	wg.Wait()
	// close all chan
	close(dtTaskChan)
	close(dtResultChan)
	close(dtOnGoingChan)
	close(dltTaskChan)
	close(dltResultChan)
	close(dltOnGoingChan)
	if len(verifyResultsMap) > 0 {
		verifyResultsSlice := make([]VerifyResults, 0)
		for _, v := range verifyResultsMap {
			verifyResultsSlice = append(verifyResultsSlice, v)
		}
		if !silenceMode {
			// write to csv file
			if storeToFile {
				myLogger.Print("Write to csv " + resultFile)
				WriteResult(verifyResultsSlice, resultFile)
				myLogger.Println("  Done!")
			}
			// write to db
			if storeToDB {
				myLogger.Print("Write to sqlite3 db file " + dbFile)
				InsertIntoDb(verifyResultsSlice, dbFile)
				myLogger.Println("  Done!")
			}
			// sort by speed
			sort.Sort(sort.Reverse(resultSpeedSorter(verifyResultsSlice)))
			myLogger.Println()
			myLogger.Println("All Results:")
			PrintFinalStat(verifyResultsSlice, dtOnly)
		} else {
			for _, v := range verifyResultsSlice {
				myLogger.Println(*(v.ip))
			}
		}
	}
}
