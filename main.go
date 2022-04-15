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
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	flag "github.com/spf13/pflag"
)

const (
	downloadBufferSize    = 1024 * 16
	workerStopSignal      = "0"
	statisticTimer        = 10
	fileDefaultSize       = 1024 * 1024 * 300
	downloadSizeMin       = 1024 * 1024
	defaultTestUrl        = "https://cf.9999876.xyz/500mb.dat"
	defaultDBFile         = "ip.db"
	DefaultTestHost       = "cf.9999876.xyz"
	downloadControlFactor = 2.0
	maxHostLen            = 1 << 16
	dtsSSL                = "SSL"
	dtsHTTPS              = "HTTPS"
	runTIME               = "cftestor"
	EventFinish           = tcell.Key(32767)
)

var (
	maxHostLenBig                          = big.NewInt(maxHostLen)
	version, ipFile                        string
	srcIPS                                 []string
	srcIPRs                                []ipRange
	srcIPRsCache                           [][]net.IP
	srcFactor                              []float64
	ipStr                                  arrayFlags
	dtCount, dtWorkerThread, port          int
	dltDurMax, dltWorkerThread             int
	dltCount, resultMin, dtPassedRateMin   int
	interval, delayMax, dtTimeout          int
	hostName, urlStr, dtSource             string
	speedMinimal                           float64
	dtHttps, disableDownload, ipv6Mode     bool
	dtOnly, dltOnly                        bool
	storeToFile, storeToDB, testAll, debug bool
	resultFile, suffixLabel, dbFile        string
	myLogger                               MyLogger
	loggerLevel                            LogLevel
	HttpRspTimeoutDuration                 time.Duration
	dtTimeoutDuration                      time.Duration
	downloadTimeMaxDuration                time.Duration
	verifyResultsMap                       = make(map[string]VerifyResults)
	defaultASN                             = 0
	defaultCity                            = ""
	myRand                                 = rand.New(rand.NewSource(0))
	titleRuntime, titlePre                 *string
	titleResult, titleDebug                *string
	titleTasksStat                         *string
	resultStrSlice, debugStrSlice          *[]string
	termAll                                *tcell.Screen
	titleStyle                             = tcell.StyleDefault.Foreground(tcell.ColorBlack.TrueColor()).Background(tcell.ColorWhite)
	titleStyleCancel                       = tcell.StyleDefault.Foreground(tcell.ColorBlack.TrueColor()).Background(tcell.ColorGray)
	defStyle                               = tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	contentStyle                           = tcell.StyleDefault
	maxResultsDisplay, maxDebugDisplay     = 10, 10
	titleRuntimeRow                        = 0
	titleCancelRow                         = titleRuntimeRow + 2
	titlePreRow                            = titleCancelRow + 2
	titleTasksStatRow                      = titlePreRow + 2
	titleResultHintRow                     = titleTasksStatRow + 2
	titleResultRow                         = titleResultHintRow + 1
	titleDebugHintRow                      = titleResultRow + maxResultsDisplay + 2
	titleDebugRow                          = titleDebugHintRow + 1
	taskStatistic                          = overAllStat{0, 0, 0, 0, 0, 0, 0}
	titleCancel                            = "Press ESC to cancel!"
	titleCancelComfirm                     = "Are you sure to cancel it?(ENTER to confirm; Any other key to quit)"
	titleWaitQuit                          = "Waiting for exit..."
	titleExitHint                          = "Press any key to exit!"
	titleResultHint                        = "Results:"
	titleDebugHint                         = "Debug:"
	cancelSigFromTerm                      = false
	terminateComfirm                       = false
)

func print_version() {
	fmt.Printf("cftestor %v\n", version)
}

func init() {
	var printVersion bool

	version = "v1.4.3"
	var help = `
    cftestor ` + version + `
    测试Cloudflare IP的延迟和速度，获取最快的IP！
    https://github.com/zhfreal/cftestor

    参数:
        -s, --ip            string  待测试IP(段)(默认为空)。例如1.0.0.1或者1.0.0.0/32，可重
                                    复使用测试多个IP或者IP段。
        -i, --in            string  IP(段) 数据文件.文本文件，每一行为一个IP或者IP段。
        -m, --dt-thread     int     延时测试线程数量(默认 20)
        -t, --dt-timeout    int     延时超时时间(ms)(默认 1000ms)。当使用"--dt-via-https"时，
                                    应适当加大。
        -c, --dt-try        int     延时测试次数(默认 10)
        -p, --port          int     测速端口(默认 443)。当使用SSL握手方式测试延时且不进行下
                                    载测试时，需要根据此参数测试；其余情况则是使用"--url"提
                                    供的参数进行测试。
            --hostname      string  SSL握手时使用的hostname(默认: "cf.9999876.xyz")仅当
                                    "--dt-only"且不携带"-dt-via-https"时有效。
            --dt-via-https          使用HTTPS请求相应方式进行延时测试开关。默认关闭，即使用
                                    SSL握手方式测试延时
        -n, --dlt-thread    int     下测试线程数(默认 1)
        -d, --dlt-period    int     单次下载测速最长时间(s)(默认 10s)
        -b, --dlt-count     int     尝试下载次数(默认 1)
        -u, --url           string  下载测速地址(默认 "https://cf.9999876.xyz/500mb.dat")。
        -I  --interval      int     测试间隔时间(ms)(默认 100ms)
        -k, --delay-limit   int     平均延时上限(ms)(默认 600ms). 平均延时超过此值不计入结
                                    果集，不进行下载测试。
        -S, --dtpr-limit    int     延迟测试成功率下限，当低于此值时不计入结果集，不进行下
                                    载测试。默认80，即不低于80%。
        -l, --speed         float   下载平均速度下限(KB/s)(默认 2000KB/s). 下载平均速度低于
                                    此值时不计入结果集。
        -r, --result        int     测速结果集数量(默认 10). 当符合条件的IP数量超过此值时，
                                    结束测试。但是如果开启 "--test-all"，此值不生效。
            --dt-only               只进行延迟测试，不进行下载测速开关(默认关闭)
            --dlt-only              不单独使用延迟测试，直接使用下载测试，但-k|--delay-limit
                                    参数仍然可用来过滤结果（默认关闭）
        -6, --ipv6                  测试IPv6开关，表示测试IPv6地址。仅当不携带-s和-i时有效。
                                    默认关闭。
        -a  --test-all              测试全部IP开关。默认关闭。
        -w, --store-to-file         是否将测试结果写入文件开关。默认关闭。
        -o, --result-file   string  输出结果文件。携带此参数将结果输出至本参数对应的文件。
        -e, --store-to-db           是否将结果存入sqlite3数据库开关。默认关闭。 
        -f, --db-file       string  sqlite3数据库文件名称。携带此参数将结果输出至本参数对应
                                    的数据库文件。
        -g, --label         string  输出结果文件后缀或者数据库中数据记录的标签，用于区分测试
                                    目标服务器。默认为"--url"地址的hostname或者"--hostname"。
        -V, --debug                 调试模式
        -v, --version               打印版本
    `

	flag.VarP(&ipStr, "ip", "s", "待测试IP或者地址段，例如1.0.0.1或者1.0.0.0/24")
	flag.StringVarP(&ipFile, "in", "i", "", "IP 数据文件")

	flag.IntVarP(&dtWorkerThread, "dt-thread", "m", 20, "Delay测试线程数")
	flag.IntVarP(&dtTimeout, "dt-timeout", "t", 1000, "Delay超时时间(ms)")
	flag.IntVarP(&dtCount, "dt-count", "c", 10, "Delay测速次数")
	flag.IntVarP(&port, "port", "p", 443, "延迟测速端口")
	flag.StringVar(&hostName, "hostname", DefaultTestHost, "SSL握手对应的hostname")
	flag.BoolVar(&dtHttps, "dt-via-https", false, "使用https连接方式进行Delay测试，默认是使用SSL握手方式")

	flag.IntVarP(&dltWorkerThread, "dlt-thread", "n", 1, "下测试线程数")
	flag.IntVarP(&dltDurMax, "dl-period", "d", 10, "单次下载测速最长时间(s)")
	flag.IntVarP(&dltCount, "dlt-count", "b", 1, "尝试下载次数")
	flag.StringVarP(&urlStr, "url", "u", defaultTestUrl, "下载测速地址")
	flag.IntVarP(&interval, "interval", "I", 100, "间隔时间(ms)")

	flag.IntVarP(&delayMax, "delay-limit", "k", 600, "平均延迟上限(ms)")
	flag.IntVarP(&dtPassedRateMin, "dtpr-limit", "S", 80, "延迟测试成功率下限(%)")
	flag.Float64VarP(&speedMinimal, "speed", "l", 6000, "下载速度下限(KB/s)")
	flag.IntVarP(&resultMin, "result", "r", 10, "测速结果集数量")

	flag.BoolVar(&disableDownload, "disable-download", false, "禁用下载测速。已废弃，请使用--dt-only。")
	flag.BoolVar(&dtOnly, "dt-only", false, "仅延迟测试，禁用速率测速")
	flag.BoolVar(&dltOnly, "dlt-only", false, "直接使用速率测试，不预先使用单独的延迟测速")
	flag.BoolVarP(&ipv6Mode, "ipv6", "6", false, "测试IPv6")
	flag.BoolVarP(&testAll, "test-all", "a", false, "测速全部IP")

	flag.BoolVarP(&storeToFile, "store-to-file", "w", false, "是否将测试结果写入文件")
	flag.StringVarP(&resultFile, "result-file", "o", "", "输出结果文件")
	flag.BoolVarP(&storeToDB, "store-to-db", "e", false, "结果写入sqlite数据库")
	flag.StringVarP(&dbFile, "db-file", "f", "", "sqlite数据库文件")
	flag.StringVarP(&suffixLabel, "label", "g", "", "输出结果文件后缀或者数据库中数据记录的标签")

	flag.BoolVarP(&debug, "debug", "V", false, "调试模式")
	flag.BoolVarP(&printVersion, "version", "v", false, "打印程序版本")
	flag.Usage = func() { fmt.Print(help) }
	flag.Parse()

	if printVersion {
		print_version()
		os.Exit(0)
	}
	if disableDownload {
		dtOnly = true
		println("Warning! \"--disable-download\" is deprecated, use \"--dt-only\" instead!")
	}
	if dtOnly && dltOnly {
		println("")
		println("--dt-only和--dlt-only不能同时使用")
		println("")
		println(version)
		os.Exit(1)
	}

	// initialize myLogger
	if debug {
		loggerLevel = logLevelDebug
	} else {
		loggerLevel = logLevelInfo
	}
	myLogger = myLogger.newLogger(loggerLevel)
	// trim whitespace
	ipFile = strings.TrimSpace(ipFile)
	resultFile = strings.TrimSpace(resultFile)
	suffixLabel = strings.TrimSpace(suffixLabel)
	hostName = strings.TrimSpace(hostName)
	urlStr = strings.TrimSpace(urlStr)
	dbFile = strings.TrimSpace(dbFile)

	if len(ipStr) != 0 {
		srcIPS = append(srcIPS, ipStr...)
	}
	if len(ipFile) != 0 {
		file, err := os.Open(ipFile)
		if err != nil {
			panic(err)
		}
		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			tIp := strings.TrimSpace(scanner.Text())
			if len(tIp) == 0 {
				continue
			}
			if isValidIPs(tIp) {
				srcIPS = append(srcIPS, tIp)
			} else {
				myLogger.Error(tIp + " is not IP or CIDR.")
			}
		}
	}
	if len(ipStr) == 0 && len(ipFile) == 0 {
		if !ipv6Mode {
			srcIPS = append(srcIPS, CFIPV4...)
		} else {
			srcIPS = append(srcIPS, CFIPV6...)
		}
	}
	// init srcIPR
	for i := 0; i < len(srcIPS); i++ {
		srcIPRs = append(srcIPRs, *NewIPRangeFromCIDR(srcIPS[i]))
		srcIPRsCache = append(srcIPRsCache, []net.IP{})
	}
	//caculate factor
	t_all_len := big.NewInt(0)
	for i := 0; i < len(srcIPRs); i++ {
		ipr := &srcIPRs[i]
		t_all_len = new(big.Int).Add(t_all_len, ipr.Len)
	}
	// init srcIPRCache, make cache when length < MaxHostLen
	// caculate factor
	t_all_len_float := new(big.Float).SetInt(t_all_len)
	for i := 0; i < len(srcIPRs); i++ {
		ipr := &srcIPRs[i]
		t_v_ratio, _ := new(big.Float).Quo(new(big.Float).SetInt(ipr.Len), t_all_len_float).Float64()
		srcFactor = append(srcFactor, t_v_ratio)
		if ipr.Len.Cmp(maxHostLenBig) < 1 {
			srcIPRsCache[i] = ipr.ExtractAll()
		}
	}
	if dtWorkerThread <= 0 {
		dtWorkerThread = 100
	}
	if resultMin <= 0 {
		resultMin = 20
	}
	if dtCount <= 0 {
		dtCount = 60
	}
	if dltWorkerThread <= 0 {
		dltWorkerThread = 1
	}
	if dltCount <= 0 {
		dltCount = 4
	}

	if dltDurMax <= 0 {
		dltDurMax = 10
	}

	if delayMax <= 0 {
		delayMax = 9999
	}
	if interval <= 0 {
		interval = 500
	}
	if speedMinimal < 0 {
		speedMinimal = 0
	}

	dtTimeoutDuration = time.Duration(dtTimeout) * time.Millisecond
	// if we ping via ssl negotiation and don't perform download test, we need check hostname and port
	if !dtHttps && dtOnly {
		//ping via ssl negotiation
		if len(hostName) == 0 {
			myLogger.Fatal("--hostname can not be empty. \n" + help)
		}
		if port < 1 || port > 65535 {
			port = 443
		}
	} else {
		// we perform download test or just ping via https request
		hostName, port = ParseUrl(urlStr)
		HttpRspTimeoutDuration = dtTimeoutDuration
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
	// init
	tResultStrSlice := make([]string, 0)
	resultStrSlice = &tResultStrSlice
	tDebugStrSlice := make([]string, 0)
	debugStrSlice = &tDebugStrSlice
	// reset the position of debugHint and debugTitle
	if resultMin < maxResultsDisplay {
		maxResultsDisplay = resultMin
		titleDebugHintRow = titleResultRow + maxResultsDisplay + 2
		titleDebugRow = titleDebugHintRow + 1
	}
	initRandSeed()
	initTitleStr()
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
	(*termAll).SetStyle(defStyle)
	(*termAll).Sync()
}

func controllerWorker(dtTaskChan *chan string, dtResultChan *chan singleVerifyResult, dltTaskChan *chan string,
	dltResultChan *chan singleVerifyResult, wg *sync.WaitGroup) {
	defer func() {
		// send terminate sigal to
		terminateComfirm = true
		wg.Done()
	}()
	dtTasks := 0
	dltTasks := 0
	dtDoneTasks := 0
	dtTaskCacher := make([]string, 0)
	dltDoneasks := 0
	dltTaskCacher := make([]string, 0)
	cacheResultMap := make(map[string]VerifyResults)
	haveEnoughResult := false
	noMoreSources := false
	OverAllStatTimer := time.Now()

LOOP:
	for {
		// cancel from terminal, or have enough results
		// flush ping and download task chan
		if cancelSigFromTerm || haveEnoughResult {
			if !dltOnly {
				for len(*dtTaskChan) > 0 {
					<-(*dtTaskChan)
				}
			}
			if !dtOnly {
				for len(*dltTaskChan) > 0 {
					<-(*dltTaskChan)
				}
			}
			// update tasks statistic
			updateTasksStat(dtTasks, dtDoneTasks,
				dltTasks, dltDoneasks, len(dtTaskCacher),
				len(dltTaskCacher), len(verifyResultsMap))
			OverAllStatTimer = time.Now()
			break LOOP
		}
		// there are ping thread working
		if !dltOnly {
			// check ping test result
			select {
			case dtResult := <-*dtResultChan:
				// if ip not test then put it into dltTaskChan
				dtDoneTasks += 1
				var tVerifyResult = singleResultStatistic(dtResult, false)
				if tVerifyResult.da > 0.0 && tVerifyResult.da <= float64(delayMax) && tVerifyResult.dtpr*100.0 >= float64(dtPassedRateMin) {
					if !dtOnly { // there are download test ongoing
						// put ping test result to cacheResultMap for later
						cacheResultMap[tVerifyResult.ip] = tVerifyResult
						dltTaskCacher = append(dltTaskCacher, tVerifyResult.ip)
					} else { // Download test disabled
						// print
						// myLogger.PrintSingleStat(loggerContent{logLevelInfo, []VerifyResults{tVerifyResult}},
						//     overAllStat{dtTasks, dtDoneTasks,
						//         dltTasks, dltDoneasks, len(dtTaskCacher),
						//         len(dltTaskCacher), len(verifyResultsMap)})
						updateResult(tVerifyResult, dtTasks, dtDoneTasks,
							dltTasks, dltDoneasks, len(dtTaskCacher),
							len(dltTaskCacher), len(verifyResultsMap))
						// reset timer
						OverAllStatTimer = time.Now()
						verifyResultsMap[tVerifyResult.ip] = tVerifyResult
						// we have expected result, break LOOP
						if !testAll && len(verifyResultsMap) >= resultMin {
							haveEnoughResult = true
						}

					}
				} else if debug { // debug print
					// myLogger.PrintSingleStat(loggerContent{logLevelDebug, []VerifyResults{tVerifyResult}},
					//     overAllStat{dtTasks, dtDoneTasks,
					//         dltTasks, dltDoneasks, len(dtTaskCacher),
					//         len(dltTaskCacher), len(verifyResultsMap)})
					updateDebug(tVerifyResult, dtTasks, dtDoneTasks,
						dltTasks, dltDoneasks, len(dtTaskCacher),
						len(dltTaskCacher), len(verifyResultsMap))
					// reset timer
					OverAllStatTimer = time.Now()
				}
			default:
			}
			// ping task control
			if !noMoreSources && !cancelSigFromTerm && !haveEnoughResult && (dtTasks-dtDoneTasks) < dtWorkerThread {
				// get more Hosts while we don't have enough hosts in dtTaskCacher
				if len(dtTaskCacher) == 0 {
					dtTaskCacher = extractCIDRHosts(2*dtWorkerThread, !testAll)
					// if no more hosts, but just in dt-only mode, we set noMoSources to true
					if len(dtTaskCacher) == 0 {
						noMoreSources = true
					}
				}
				// when it's dt-only mode or, download tasks pool has less hosts than downloadThread
				// we put ping task into dtTaskCacher
				// simplify algorithm
				if dtOnly || (len(dltTaskCacher)+dltTasks-dltDoneasks) < dltWorkerThread {
					for i := 0; i < dtWorkerThread; i++ {
						if len(dtTaskCacher) == 0 {
							break
						}
						dtTasks += 1
						*dtTaskChan <- dtTaskCacher[0]
						if len(dtTaskCacher) > 1 {
							dtTaskCacher = dtTaskCacher[1:]
						} else {
							dtTaskCacher = []string{}
						}
					}
					// update stat
					updateTasksStat(dtTasks, dtDoneTasks,
						dltTasks, dltDoneasks, len(dtTaskCacher),
						len(dltTaskCacher), len(verifyResultsMap))
				}
			}
			// we did all ping works in dt-only mode
			if dtOnly && (cancelSigFromTerm || haveEnoughResult || noMoreSources) && dtTasks <= dtDoneTasks {
				break LOOP
			}
		}
		// there are download thread working
		if !dtOnly {
			select {
			// check download result
			case out := <-*dltResultChan:
				dltDoneasks += 1
				var tVerifyResult = singleResultStatistic(out, true)
				var v = VerifyResults{}
				if dltOnly {
					v = tVerifyResult
				} else {
					v = cacheResultMap[tVerifyResult.ip]
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
				// check when delay test duration and download speed
				if tVerifyResult.dls >= speedMinimal && tVerifyResult.dlds > downloadSizeMin {
					// put tVerifyResult into verifyResultsMap
					verifyResultsMap[tVerifyResult.ip] = v
					// we have expected result
					if !testAll && len(verifyResultsMap) >= resultMin {
						haveEnoughResult = true
					}
					// myLogger.PrintSingleStat(loggerContent{logLevelInfo, []VerifyResults{v}},
					//     overAllStat{dtTasks, dtDoneTasks,
					//         dltTasks, dltDoneasks, len(dtTaskCacher),
					//         len(dltTaskCacher), len(verifyResultsMap)})
					updateResult(tVerifyResult, dtTasks, dtDoneTasks,
						dltTasks, dltDoneasks, len(dtTaskCacher),
						len(dltTaskCacher), len(verifyResultsMap))
					OverAllStatTimer = time.Now()
				} else if debug { // debug print
					// myLogger.PrintSingleStat(loggerContent{logLevelDebug, []VerifyResults{v}},
					//     overAllStat{dtTasks, dtDoneTasks,
					//         dltTasks, dltDoneasks, len(dtTaskCacher),
					//         len(dltTaskCacher), len(verifyResultsMap)})
					updateDebug(tVerifyResult, dtTasks, dtDoneTasks,
						dltTasks, dltDoneasks, len(dtTaskCacher),
						len(dltTaskCacher), len(verifyResultsMap))
					OverAllStatTimer = time.Now()
				}

			default:
			}
			// Download task control
			// put task to queue if it has enough ips
			if !cancelSigFromTerm && !haveEnoughResult {
				// get more hosts while in downlaodOnly mode
				if dltOnly && len(dltTaskCacher) == 0 {
					dltTaskCacher = extractCIDRHosts(5*dltWorkerThread, !testAll)
					if len(dltTaskCacher) == 0 {
						noMoreSources = true
					}
				}
				// put task to download chan when we have IPs from delay test and the download thread is available
				if len(dltTaskCacher) > 0 && (dltTasks-dltDoneasks) < dltWorkerThread {
					for i := 0; i < dltWorkerThread; i++ {
						if len(dltTaskCacher) == 0 {
							break
						}
						*dltTaskChan <- dltTaskCacher[0]
						dltTasks += 1
						if len(dltTaskCacher) > 1 {
							dltTaskCacher = dltTaskCacher[1:]
						} else {
							dltTaskCacher = []string{}
						}
					}
					// update stat
					updateTasksStat(dtTasks, dtDoneTasks,
						dltTasks, dltDoneasks, len(dtTaskCacher),
						len(dltTaskCacher), len(verifyResultsMap))
				}
			}
			if dltTasks <= dltDoneasks {
				if dltOnly { // we done in dlt-only mode
					if cancelSigFromTerm || haveEnoughResult || noMoreSources {
						break LOOP
					}
				} else if dtTasks <= dtDoneTasks { // we done in ping and download co-working mode
					if cancelSigFromTerm || haveEnoughResult || (noMoreSources && len(dltTaskCacher) == 0) {
						break LOOP
					}
				}
			}
		}
		// Print overall stat during waiting time and reset OverAllStatTimer
		if time.Since(OverAllStatTimer) > time.Duration(statisticTimer)*time.Second {
			// myLogger.PrintOverAllStat(overAllStat{dtTasks, dtDoneTasks,
			//     dltTasks, dltDoneasks, len(dtTaskCacher),
			//     len(dltTaskCacher), len(verifyResultsMap)})
			updateTasksStat(dtTasks, dtDoneTasks,
				dltTasks, dltDoneasks, len(dtTaskCacher),
				len(dltTaskCacher), len(verifyResultsMap))
			OverAllStatTimer = time.Now()
		}
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	// myLogger.PrintOverAllStat(overAllStat{dtTasks, dtDoneTasks,
	//     dltTasks, dltDoneasks, len(dtTaskCacher),
	//     len(dltTaskCacher), len(verifyResultsMap)})
	updateTasksStat(dtTasks, dtDoneTasks,
		dltTasks, dltDoneasks, len(dtTaskCacher),
		len(dltTaskCacher), len(verifyResultsMap))
	// put stop signal to all delay test workers and download worker
	if !dltOnly {
		for i := 0; i < dtWorkerThread; i++ {
			*dtTaskChan <- workerStopSignal
		}
	}
	if !dtOnly {
		for i := 0; i < dltWorkerThread; i++ {
			*dltTaskChan <- workerStopSignal
		}
	}
	// hint for exit
	printExitHint()
	// (*termAll).PostEvent(EventFinish)
	// fmt.Println()
}

func termControl(wg *sync.WaitGroup) {
	initScreen()
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
	(*termAll).Fini()
	(*wg).Done()
	fmt.Println(titleWaitQuit)
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
		}
	}
}

func main() {
	var wg sync.WaitGroup
	var dtTaskChan = make(chan string, dtWorkerThread*2)
	var dtResultChan = make(chan singleVerifyResult, dtWorkerThread*2)
	var dltTaskChan = make(chan string, dltWorkerThread*2)
	var dltResultChan = make(chan singleVerifyResult, dltWorkerThread*2)

	go termControl(&wg)
	wg.Add(1)
	// start controller worker
	go controllerWorker(&dtTaskChan, &dtResultChan, &dltTaskChan, &dltResultChan, &wg)

	wg.Add(1)
	// start ping worker
	if !dltOnly {
		for i := 0; i < dtWorkerThread; i++ {
			if dtHttps {
				go downloadWorker(dtTaskChan, dtResultChan, &wg, urlStr, port,
					dtTimeoutDuration, downloadTimeMaxDuration, dtCount, interval, true)
			} else {
				go sslDTWorker(dtTaskChan, dtResultChan, &wg, hostName, port,
					dtTimeoutDuration, dtCount, interval)
			}
			wg.Add(1)
		}
	}

	// start download worker if don't do ping only
	if !dtOnly {
		for i := 0; i < dltWorkerThread; i++ {
			go downloadWorker(dltTaskChan, dltResultChan, &wg, urlStr, port,
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
		// sort by speed
		sort.Sort(sort.Reverse(resultSpeedSorter(verifyResultsSlice)))
		myLogger.headerPrinted = false
		myLogger.Println(logLevelInfo, "All Results:")
		fmt.Println()
		PrintFinalStat(verifyResultsSlice, dtOnly)
		// write to csv file
		if storeToFile {
			WriteResult(verifyResultsSlice, resultFile)
		}
		// write to db
		if storeToDB {
			InsertIntoDb(verifyResultsSlice, dbFile)
		}
	}
}
