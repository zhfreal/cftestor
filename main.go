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
	workerStopSignal    = "0"
	workOnGoing         = 1
	controlerInterval   = 100               // in millisecond
	statisticIntervalT  = 1000              // in millisecond, valid in tcell mode
	statisticIntervalNT = 10000             // in millisecond, valid in non-tcell mode
	quitWaitingTime     = 3                 // in second
	downloadBufferSize  = 1024 * 16         // in byte
	fileDefaultSize     = 1024 * 1024 * 300 // in byte
	downloadSizeMin     = 1024 * 1024       // in byte
	defaultTestUrl      = "https://cf.9999876.xyz/500mb.dat"
	userAgent           = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.54 Safari/537.36"
	defaultDBFile       = "ip.db"
	DefaultTestHost     = "cf.9999876.xyz"
	maxHostLen          = 1 << 16
	dtsSSL              = "SSL"
	dtsHTTPS            = "HTTPS"
	runTime             = "cftestor"
)

var (
	maxHostLenBig                          = big.NewInt(maxHostLen)
	version, ipFile                        string
	srcIPS                                 []*string
	srcIPRs                                []*ipRange
	srcIPRsCache                           []net.IP
	ipStr                                  arrayFlags
	dtCount, dtWorkerThread, port          int
	dltDurMax, dltWorkerThread             int
	dltCount, resultMin                    int
	interval, delayMax, dtTimeout          int
	hostName, urlStr, dtSource             string
	dtPassedRateMin, speedMinimal          float64
	dtHttps, disableDownload               bool
	ipv4Mode, ipv6Mode, dtOnly, dltOnly    bool
	storeToFile, storeToDB, testAll, debug bool
	resultFile, suffixLabel, dbFile        string
	myLogger                               MyLogger
	loggerLevel                            LogLevel
	HttpRspTimeoutDuration                 time.Duration
	dtTimeoutDuration                      time.Duration
	downloadTimeMaxDuration                time.Duration
	verifyResultsMap                       = make(map[*string]VerifyResults)
	defaultASN                             = 0
	defaultCity                            = ""
	myRand                                 = rand.New(rand.NewSource(0))
	titleRuntime                           *string
	titlePre                               [2][4]string
	titleTasksStat                         [2]*string
	detailTitleSlice                       []string
	resultStrSlice, debugStrSlice          [][]*string
	termAll                                *tcell.Screen
	titleStyle                             = tcell.StyleDefault.Foreground(tcell.ColorBlack.TrueColor()).Background(tcell.ColorWhite)
	normalStyle                            = tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	titleStyleCancel                       = tcell.StyleDefault.Foreground(tcell.ColorBlack.TrueColor()).Background(tcell.ColorGray)
	contentStyle                           = tcell.StyleDefault
	maxResultsDisplay                      = 10
	maxDebugDisplay                        = 10
	titleRuntimeRow                        = 0
	titlePreRow                            = titleRuntimeRow + 2
	titleCancelRow                         = titlePreRow + 3
	titleTasksStatRow                      = titleCancelRow + 2
	titleResultHintRow                     = titleTasksStatRow + 2
	titleResultRow                         = titleResultHintRow + 1
	titleDebugHintRow                      = titleResultRow + maxResultsDisplay + 2
	titleDebugRow                          = titleDebugHintRow + 1
	titleCancel                            = "Press ESC to cancel!"
	titleCancelComfirm                     = "Press ENTER to confirm; Any other key to back!"
	titleWaitQuit                          = "Waiting for exit..."
	titleResultHint                        = "Result:"
	titleDebugHint                         = "Debug Msg:"
	cancelSigFromTerm                      = false
	terminateComfirm                       = false
	resultStatIndent                       = 9
	dtThreadNumLen, dltThreadNumLen        = 0, 0
	noTcell                                = false
	statInterval                           = statisticIntervalNT
	// titleExitHint                          = "Press any key to exit!"
)

func print_version() {
	fmt.Printf("%v %v\n", runTime, version)
}

func init() {
	var printVersion bool

	// version = "dev"
	var help = runTime + " " + version + `
    根据延迟、丢包率和速度优选CF IP
    https://github.com/zhfreal/cftestor

    参数:
        -s, --ip            string  待测试IP(段)。例如: "-s 1.0.0.1", "-s 1.0.0.1/32", 
                                    "-s 1.0.0.1/24"。可重复使用, 传递多个IP或者IP段。
        -i, --in            string  IP(段) 数据文件, 文本文件。 其每一行为一个IP或者IP段。
        -m, --dt-thread     int     延时测试线程数量, 默认20。
        -t, --dt-timeout    int     延时超时时间(ms), 默认1000ms。此值不能小于
                                    "-k|--delay-limit"。当使用"--dt-via-https"时, 应适
                                    当加大此值。
        -c, --dt-count      int     延时测试次数, 默认4。
        -p, --port          int     测速端口, 默认443。当使用SSL握手方式测试延时且不进行下载测
                                    试时, 需要根据此参数测试；其余情况则是使用"--url"提供的参
                                    数进行测试。
            --hostname      string  SSL握手时使用的hostname, 默认"cf.9999876.xyz"。仅当
                                    "--dt-only"且不携带"-dt-via-https"时有效。
            --dt-via-https          使用HTTPS请求相应方式进行延时测试开关。
                                    默认关闭, 即使用SSL握手方式测试延时。
        -n, --dlt-thread    int     下测试线程数, 默认1。
        -d, --dlt-period    int     单次下载测速最长时间(s), 默认10s。
        -b, --dlt-count     int     尝试下载次数, 默认1。
        -u, --url           string  下载测速地址, 默认 "https://cf.9999876.xyz/500mb.dat"。
        -I  --interval      int     测试间隔时间(ms), 默认500ms。
        -k, --delay-limit   int     平均延时上限(ms), 默认600ms。 平均延时超过此值不计入结果集,
                                    不进行下载测试。
        -S, --dtpr-limit    float   延迟测试成功率下限(%), 默认100%。
                                    当低于此值时不计入结果集, 不进行下载测试。默认100, 即不低于
                                    100%。此值低于100%的IP会发生断流或者偶尔无法连接的情况。
        -l, --speed         float   下载平均速度下限(KB/s), 默认2000KB/s。下载平均速度低于此值
                                    时不计入结果集。
        -r, --result        int     测速结果集数量, 默认10。
                                    当符合条件的IP数量超过此值时, 结束测试。但是如果开启
                                    "--test-all", 此值不生效。
            --dt-only               只进行延迟测试, 不进行下载测速开关, 默认关闭。
            --dlt-only              不单独使用延迟测试, 直接使用下载测试, 默认关闭。
        -4, --ipv4                  测试IPv4开关, 表示测试IPv4地址。仅当不携带"-s"和"-i"时有效。
                                    默认打开。与"-6|--ipv6"不能同时使用。
        -6, --ipv6                  测试IPv6开关, 表示测试IPv6地址。仅当不携带"-s"和"-i"时有效。
                                    默认关闭。与"-4|--ipv4"不能同时使用。
        -a  --test-all              测试全部IP开关。默认关闭。
        -w, --store-to-file         是否将测试结果写入文件开关, 默认关闭。
        -o, --result-file   string  输出结果文件。携带此参数将结果输出至本参数对应的文件。
        -e, --store-to-db           是否将结果存入sqlite3数据库开关。默认关闭。 
        -f, --db-file       string  sqlite3数据库文件名称。携带此参数将结果输出至本参数对应的数
                                    据库文件。
        -g, --label         string  输出结果文件后缀或者数据库中数据记录的标签, 用于区分测试目标
                                    服务器。默认为"--url"地址的hostname或者"--hostname"。
            --no-tcell      bool    不使用TCell显示。
        -V, --debug                 调试模式。
        -v, --version               打印版本。
    `

	flag.VarP(&ipStr, "ip", "s", "待测试IP或者地址段, 例如1.0.0.1或者1.0.0.0/24")
	flag.StringVarP(&ipFile, "in", "i", "", "IP 数据文件")

	flag.IntVarP(&dtWorkerThread, "dt-thread", "m", 20, "Delay测试线程数")
	flag.IntVarP(&dtTimeout, "dt-timeout", "t", 1000, "Delay超时时间(ms)")
	flag.IntVarP(&dtCount, "dt-count", "c", 4, "Delay测速次数")
	flag.IntVarP(&port, "port", "p", 443, "延迟测速端口")
	flag.StringVar(&hostName, "hostname", DefaultTestHost, "SSL握手对应的hostname")
	flag.BoolVar(&dtHttps, "dt-via-https", false, "使用https连接方式进行Delay测试, 默认是使用SSL握手方式")

	flag.IntVarP(&dltWorkerThread, "dlt-thread", "n", 1, "下测试线程数")
	flag.IntVarP(&dltDurMax, "dlt-period", "d", 10, "单次下载测速最长时间(s)")
	flag.IntVarP(&dltCount, "dlt-count", "b", 1, "尝试下载次数")
	flag.StringVarP(&urlStr, "url", "u", defaultTestUrl, "下载测速地址")
	flag.IntVarP(&interval, "interval", "I", 500, "间隔时间(ms)")

	flag.IntVarP(&delayMax, "delay-limit", "k", 600, "平均延迟上限(ms)")
	flag.Float64VarP(&dtPassedRateMin, "dtpr-limit", "S", 100, "延迟测试成功率下限(%)")
	flag.Float64VarP(&speedMinimal, "speed", "l", 6000, "下载速度下限(KB/s)")
	flag.IntVarP(&resultMin, "result", "r", 10, "测速结果集数量")

	flag.BoolVar(&disableDownload, "disable-download", false, "禁用下载测速。已废弃, 请使用--dt-only。")
	flag.BoolVar(&dtOnly, "dt-only", false, "仅延迟测试, 禁用速率测速")
	flag.BoolVar(&dltOnly, "dlt-only", false, "直接使用速率测试, 不预先使用单独的延迟测速")
	flag.BoolVarP(&ipv4Mode, "ipv4", "4", true, "测试IPv4地址")
	flag.BoolVarP(&ipv6Mode, "ipv6", "6", false, "测试IPv6地址")
	flag.BoolVarP(&testAll, "test-all", "a", false, "测速全部IP")

	flag.BoolVarP(&storeToFile, "store-to-file", "w", false, "是否将测试结果写入文件")
	flag.StringVarP(&resultFile, "result-file", "o", "", "输出结果文件")
	flag.BoolVarP(&storeToDB, "store-to-db", "e", false, "结果写入sqlite数据库")
	flag.StringVarP(&dbFile, "db-file", "f", "", "sqlite数据库文件")
	flag.StringVarP(&suffixLabel, "label", "g", "", "输出结果文件后缀或者数据库中数据记录的标签")

	flag.BoolVar(&noTcell, "no-tcell", false, "不使用Tcell输出")
	flag.BoolVarP(&debug, "debug", "V", false, "调试模式")
	flag.BoolVarP(&printVersion, "version", "v", false, "打印程序版本")
	flag.Usage = func() { fmt.Print(help) }
	flag.Parse()

	if len(version) == 0 {
		version = "dev"
	}
	if printVersion {
		print_version()
		os.Exit(0)
	}
	if disableDownload {
		dtOnly = true
		println("Warning! \"--disable-download\" 已经弃用, 请使用 \"--dt-only\"!")
	}
	if dtOnly && dltOnly {
		print_version()
		println("\"--dt-only\"和\"--dlt-only\"不能同时使用!")
		os.Exit(1)
	}

	// initialize myLogger
	if debug {
		loggerLevel = logLevelDebug
	} else {
		loggerLevel = logLevelInfo
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
	if dtTimeout < delayMax {
		timeoutFlag := flag.Lookup("dt-timeout")
		// reset dtTimeout, when dtTimeout less than delayMax and did not set value of dtTimeout from cmdline
		if !timeoutFlag.Changed {
			dtTimeout = delayMax + int(delayMax/2)
		} else {
			myLogger.Warning(fmt.Sprintf("\"-t|--dt-timeout\" - %v 小于 \"-k|--delay-limit\" - %v, 会导致部分连接总失败！", dtTimeout, delayMax))
			if !confirm("继续？", 3) {
				os.Exit(0)
			}
		}
	}

	// it's invalid when ipv4Mode and ipv6Mode is both true or false
	if ipv4Mode == ipv6Mode {
		print_version()
		println("\"-4|--ipv4\"和\"-6|--ipv6\"只能为其一, 默认\"-4|--ipv4\"!")
		os.Exit(1)
	}

	// trim whitespace
	ipFile = strings.TrimSpace(ipFile)
	resultFile = strings.TrimSpace(resultFile)
	suffixLabel = strings.TrimSpace(suffixLabel)
	hostName = strings.TrimSpace(hostName)
	urlStr = strings.TrimSpace(urlStr)
	dbFile = strings.TrimSpace(dbFile)

	if len(ipStr) != 0 {
		for i := 0; i < len(ipStr); i++ {
			srcIPS = append(srcIPS, &ipStr[i])
		}
	}
	if len(ipFile) != 0 {
		file, err := os.Open(ipFile)
		if err != nil {
			myLogger.Fatal(fmt.Sprintf("文件不存在或者不可读取: %s", ipFile))
		}
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			tIp := strings.TrimSpace(scanner.Text())
			if len(tIp) == 0 {
				continue
			}
			if isValidIPs(tIp) {
				srcIPS = append(srcIPS, &tIp)
			} else {
				myLogger.Fatal(tIp + " 合法的IP或者CIDR.")
			}
		}
	}
	if len(ipStr) == 0 && len(ipFile) == 0 {
		if !ipv6Mode {
			for i := 0; i < len(CFIPV4); i++ {
				srcIPS = append(srcIPS, &CFIPV4[i])
			}
		} else {
			for i := 0; i < len(CFIPV6); i++ {
				srcIPS = append(srcIPS, &CFIPV6[i])
			}
		}
	}
	// init srcIPR and srcIPRsCache
	for i := 0; i < len(srcIPS); i++ {
		ipr := NewIPRangeFromCIDR(srcIPS[i])
		// when it do not testAll and ipr is not bigger than maxHostLenBig, extracto to cache
		if !testAll && ipr.Len.Cmp(maxHostLenBig) < 1 {
			srcIPRsCache = append(srcIPRsCache, ipr.ExtractAll()...)
		} else {
			// when it do not perform tealAll or not bigger than maxHostLenBig, just put it to srcIPRs
			srcIPRs = append(srcIPRs, ipr)
		}
	}
	// random srcIPR and srcIPRsCache when do not testAll
	if !testAll {
		myRand.Shuffle(len(srcIPRs), func(m, n int) {
			srcIPRs[m], srcIPRs[n] = srcIPRs[n], srcIPRs[m]
		})
		myRand.Shuffle(len(srcIPRsCache), func(m, n int) {
			srcIPRsCache[m], srcIPRsCache[n] = srcIPRsCache[n], srcIPRsCache[m]
		})
	}
	if dtWorkerThread <= 0 {
		myLogger.Fatal("\"-m|--dt-thread\" 不应小于0!\n")
	}
	if resultMin <= 0 {
		myLogger.Fatal("\"-r|--result\" 不应小于0!\n")
	}
	dtThreadNumLen = len(strconv.Itoa(dtWorkerThread))
	if dtCount <= 0 {
		myLogger.Fatal("\"-c|--dt-count\" 不应小于0!\n")
	}
	if dltWorkerThread <= 0 {
		myLogger.Fatal("\"-n|--dlt-thread\" 不应小于0!\n")
	}
	dltThreadNumLen = len(strconv.Itoa(dltWorkerThread))
	if dltCount <= 0 {
		myLogger.Fatal("\"-b|--dlt-count\" 不应小于0!\n")
	}

	if dltDurMax <= 0 {
		myLogger.Fatal("\"-d|--dl-period\" 不应小于0!\n")
	}

	if delayMax <= 0 {
		myLogger.Fatal("\"-k|--delay-limit\" 不应小于0!\n")
	}
	if interval <= 0 {
		myLogger.Fatal("\"-I|--interval\" 不应小于0!\n")
	}
	if speedMinimal <= 0 {
		myLogger.Fatal("\"-l|--speed\" 不应小于0!\n")
	}

	dtTimeoutDuration = time.Duration(dtTimeout) * time.Millisecond
	// if we ping via ssl negotiation and don't perform download test, we need check hostname and port
	if !dtHttps && dtOnly {
		//ping via ssl negotiation
		if len(hostName) == 0 {
			myLogger.Fatal("\"--hostname\" 不能为空. \n" + help)
		}
		if port < 1 || port > 65535 {
			port = 443
		}
	} else {
		// we perform download test or just ping via https request
		hostName, port = ParseUrl(urlStr)
	}
	// we set HttpRspTimeoutDuration to 2 times of dtTimeoutDuration if we don't perform ping via https
	if !dtHttps {
		dtSource = dtsSSL
		HttpRspTimeoutDuration = dtTimeoutDuration * 2
	} else {
		dtSource = dtsHTTPS
		HttpRspTimeoutDuration = dtTimeoutDuration
	}
	downloadTimeMaxDuration = time.Duration(dltDurMax) * time.Second
	//
	if len(suffixLabel) == 0 {
		suffixLabel = hostName
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

	if !noTcell { // It's running on tcell mode
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

func controllerWorker(dtTaskChan chan *string, dtResultChan chan singleVerifyResult, dltTaskChan chan *string,
	dltResultChan chan singleVerifyResult, wg *sync.WaitGroup, dtOnGoingChan chan int, dltOnGoingChan chan int) {
	defer func() {
		// send terminate sigal to
		terminateComfirm = true
		(*wg).Done()
	}()
	dtTasks := 0
	dltTasks := 0
	dtDoneTasks := 0
	dtTaskCacher := make([]*string, 0)
	dltDoneTasks := 0
	dltTaskCacher := make([]*string, 0)
	cacheResultMap := make(map[string]VerifyResults)
	haveEnoughResult := false
	noMoreSources := false
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
				dtTaskCacher = []*string{}
			}
			if !dtOnly {
				for len(dltTaskChan) > 0 {
					<-(dltTaskChan)
					dltTasks--
				}
				dltTaskCacher = []*string{}
			}
			// show waiting msg
			if !showQuitWaiting {
				if !noTcell {
					printQuitWaiting()
				} else {
					myLogger.Info(titleWaitQuit + "\n")
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
					if tVerifyResult.da > 0.0 && tVerifyResult.da <= float64(delayMax) && tVerifyResult.dtpr*100.0 >= float64(dtPassedRateMin) {
						if !dtOnly { // there are download test ongoing
							// put ping test result to cacheResultMap for later
							cacheResultMap[*tVerifyResult.ip] = tVerifyResult
							dltTaskCacher = append(dltTaskCacher, tVerifyResult.ip)
							// debug msg
							displayDetails(true, []VerifyResults{tVerifyResult})
						} else { // Download test disabled
							// non-debug msg
							displayDetails(false, []VerifyResults{tVerifyResult})
							verifyResultsMap[tVerifyResult.ip] = tVerifyResult
							// we have expected result, break LOOP
							if !testAll && len(verifyResultsMap) >= resultMin {
								haveEnoughResult = true
							}
						}
					} else if debug {
						// debug msg
						displayDetails(true, []VerifyResults{tVerifyResult})
					}
				default:
				}
				// Print overall stat during waiting time and reset OverAllStatTimer
				if time.Since(OverAllStatTimer) > time.Duration(statInterval)*time.Millisecond {
					displayStat(debug, overAllStat{
						dtTasksDone:  dtDoneTasks,
						dtOnGoing:    len(dtOnGoingChan),
						dtCached:     len(dtTaskCacher) + len(dtTaskChan),
						dltTasksDone: dltDoneTasks,
						dltOnGoing:   len(dltOnGoingChan),
						dltCached:    len(dltTaskCacher) + len(dltTaskChan),
						resultCount:  len(verifyResultsMap),
					})
					OverAllStatTimer = time.Now()
				}
			}
			// DT task control, when it have enougth source ip, don't get cancel signal from term,
			// don't result as expected, and the task chan is not full
			if !noMoreSources && !cancelSigFromTerm && !haveEnoughResult && len(dtTaskChan) < cap(dtTaskChan) {
				// get more Hosts while we don't have enough hosts in dtTaskCacher
				if len(dtTaskCacher) == 0 {
					dtTaskCacher = extractCIDRHosts(2 * dtWorkerThread)
					// if no more hosts, but just in dt-only mode, we set noMoSources to true
					if len(dtTaskCacher) == 0 {
						noMoreSources = true
					}
				}
				// when it's dt-only mode or, download task pool has less ip than 2*cap(dltTaskChan)
				// we put ping task into dtTaskCacher
				// simplify algorithm
				if dtOnly || len(dltTaskCacher) < 2*cap(dltTaskChan) {
					for len(dtTaskCacher) > 0 &&
						len(dtTaskChan) < cap(dtTaskChan) &&
						len(dtTaskChan)+len(dtOnGoingChan)+len(dtResultChan) < cap(dtResultChan) {
						// to prevent overflow of dtResultChan
						// the total IP and task in dtTaskChan, dtOnGoingChan and dtResultChan is less than the capacity of dtResultChan
						dtTasks += 1
						dtTaskChan <- dtTaskCacher[0]
						if len(dtTaskCacher) > 1 {
							dtTaskCacher = dtTaskCacher[1:]
						} else {
							dtTaskCacher = []*string{}
						}
					}
				}
			} else if dtOnly &&
				len(dtOnGoingChan) == 0 &&
				len(dtTaskCacher) == 0 &&
				len(dtTaskChan) == 0 { // we did all ping works in dt-only mode
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
					if v.dls >= speedMinimal && v.dlds > downloadSizeMin {
						// put v into verifyResultsMap
						verifyResultsMap[tVerifyResult.ip] = tVerifyResult
						// we have expected result
						if !testAll && len(verifyResultsMap) >= resultMin {
							haveEnoughResult = true
						}
						// non-debug msg
						displayDetails(false, []VerifyResults{tVerifyResult})
					} else if debug {
						// debug msg
						displayDetails(true, []VerifyResults{tVerifyResult})
					}
				default:
				}
				// Print overall stat during waiting time and reset OverAllStatTimer
				if time.Since(OverAllStatTimer) > time.Duration(statInterval)*time.Millisecond {
					displayStat(debug, overAllStat{
						dtTasksDone:  dtDoneTasks,
						dtOnGoing:    len(dtOnGoingChan),
						dtCached:     len(dtTaskCacher) + len(dtTaskChan),
						dltTasksDone: dltDoneTasks,
						dltOnGoing:   len(dltOnGoingChan),
						dltCached:    len(dltTaskCacher) + len(dltTaskChan),
						resultCount:  len(verifyResultsMap),
					})
					OverAllStatTimer = time.Now()
				}
			}
			// DLT task control, when it don't get cancel signal from term, don't result as expected
			if !cancelSigFromTerm && !haveEnoughResult {
				// get more hosts while it's on downlaod-only mode
				if dltOnly && len(dltTaskCacher) == 0 {
					dltTaskCacher = extractCIDRHosts(2 * dltWorkerThread)
					if len(dltTaskCacher) == 0 {
						noMoreSources = true
					}
				}
				// put task to download chan when we have IPs from delay test and the task chan have empty slot
				for len(dltTaskCacher) > 0 && // it has IP in dltTaskCacher
					len(dltTaskChan) < cap(dltTaskChan) && // dltTaskChan is not full
					len(dltTaskChan)+len(dltOnGoingChan)+len(dltResultChan) < cap(dltResultChan) {
					// to prevent overflow of dltResultChan
					// the total IP and task in dltTaskChan, dltOnGoingChan and dltResultChan is less than the capacity of dltResultChan
					dltTaskChan <- dltTaskCacher[0]
					dltTasks += 1
					if len(dltTaskCacher) > 1 {
						dltTaskCacher = dltTaskCacher[1:]
					} else {
						dltTaskCacher = []*string{}
					}
				}
			}
			if (cancelSigFromTerm || haveEnoughResult || noMoreSources) &&
				len(dltOnGoingChan) == 0 &&
				len(dltTaskChan) == 0 &&
				len(dltTaskCacher) == 0 &&
				(dltOnly ||
					(len(dtOnGoingChan) == 0 &&
						len(dtTaskChan) == 0 &&
						len(dtTaskCacher) == 0)) { // terminate
				break LOOP
			}
		}
		// Print overall stat during waiting time and reset OverAllStatTimer
		if time.Since(OverAllStatTimer) > time.Duration(statInterval)*time.Millisecond {
			displayStat(debug, overAllStat{
				dtTasksDone:  dtDoneTasks,
				dtOnGoing:    len(dtOnGoingChan),
				dtCached:     len(dtTaskCacher) + len(dtTaskChan),
				dltTasksDone: dltDoneTasks,
				dltOnGoing:   len(dltOnGoingChan),
				dltCached:    len(dltTaskCacher) + len(dltTaskChan),
				resultCount:  len(verifyResultsMap),
			})
			OverAllStatTimer = time.Now()
		}
		time.Sleep(time.Duration(controlerInterval) * time.Millisecond)
	}
	// update statistic just before quit controller
	displayStat(debug, overAllStat{
		dtTasksDone:  dtDoneTasks,
		dtOnGoing:    len(dtOnGoingChan),
		dtCached:     len(dtTaskCacher) + len(dtTaskChan),
		dltTasksDone: dltDoneTasks,
		dltOnGoing:   len(dltOnGoingChan),
		dltCached:    len(dltTaskCacher) + len(dltTaskChan),
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
	for !terminateComfirm {
		if !(*termAll).HasPendingEvent() {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		ev := (*termAll).PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape:
				if !terminateComfirm && !cancelSigFromTerm && confirmQuit() {
					cancelSigFromTerm = true
				}
				if terminateComfirm {
					break LOOP
				}
			default:
				if terminateComfirm {
					break LOOP
				}
			}
		case *tcell.EventResize:
			initScreen()
		}
	}
	printQuitingCountDown(quitWaitingTime)

}

func confirmQuit() bool {
	printCancelComfirm()
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
			printCancelComfirm()
		}
	}
}

func main() {
	var wg sync.WaitGroup
	var dtTaskChan = make(chan *string, dtWorkerThread)
	var dtResultChan = make(chan singleVerifyResult, dtWorkerThread*4)
	var dtOnGoingChan = make(chan int, dtWorkerThread)
	var dltTaskChan = make(chan *string, dltWorkerThread)
	var dltResultChan = make(chan singleVerifyResult, dltWorkerThread*4)
	var dltOnGoingChan = make(chan int, dltWorkerThread)

	if !noTcell {
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
				go downloadWorker(dtTaskChan, dtResultChan, dtOnGoingChan, &wg, &urlStr, port,
					dtTimeoutDuration, downloadTimeMaxDuration, dtCount, interval, true)
			} else {
				go sslDTWorker(dtTaskChan, dtResultChan, dtOnGoingChan, &wg, &hostName, port,
					dtTimeoutDuration, dtCount, interval)
			}
			wg.Add(1)
		}
	}

	// start download worker if don't do ping only
	if !dtOnly {
		for i := 0; i < dltWorkerThread; i++ {
			go downloadWorker(dltTaskChan, dltResultChan, dltOnGoingChan, &wg, &urlStr, port,
				HttpRspTimeoutDuration, downloadTimeMaxDuration, dltCount, interval, false)
			wg.Add(1)
		}
	}
	wg.Wait()
	close(dtTaskChan)
	close(dtResultChan)
	close(dltTaskChan)
	close(dltResultChan)
	if len(verifyResultsMap) > 0 {
		verifyResultsSlice := make([]VerifyResults, 0)
		for _, v := range verifyResultsMap {
			verifyResultsSlice = append(verifyResultsSlice, v)
		}
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
		myLogger.Println("All Results:\n")
		PrintFinalStat(verifyResultsSlice, dtOnly)

	}
}
