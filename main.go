package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	flag "github.com/spf13/pflag"
)

const (
	DownloadBufferSize    = 1024 * 16
	WorkerStopSignal      = "DONE"
	StatisticTimer        = 10
	FileDefaultSize       = 1024 * 1024 * 300
	DownloadSizeMin       = 1024 * 1024
	DefaultTestUrl        = "https://cf.9999876.xyz/500mb.dat"
	DefaultDBFile         = "ip.db"
	DefaultTestHost       = "cf.9999876.xyz"
	DownloadControlFactor = 2.0
)

var (
	version, ipFile                        string
	srcIPS, ips                            []string
	ipStr                                  arrayFlags
	pingTry, pingWorkerThread, port        int
	downloadTimer, downloadThread          int
	downloadTry, resultMax, successMin     int
	interval, delayMax, pingTimeout        int
	hostName, urlStr                       string
	speedMinStatic                         float64
	pingViaHttp, disableDownload, ipv6Mode bool
	pingOnly, downloadOnly                 bool
	storeToFile, storeToDB, testAll, debug bool
	resultFile, suffixLabel, dbFile        string
	myLogger                               MyLogger
	loggerLevel                            LogLevel
	HttpRspTimeoutDuration                 time.Duration
	pingTimeoutDuration                    time.Duration
	downloadTimeMaxDuration                time.Duration
	verifyResultsMap                       = make(map[string]VerifyResults)
	defaultASN                             = 0
	defaultCity                            = ""
)

func init() {
	var printVersion bool

	version = "v1.2.7"
	var help = `
    cftestor ` + version + `
    测试Cloudflare IP的延迟和速度，获取最快的IP！
    https://github.com/zhfreal/cftestor

    参数:
        -s, --ip                    string  待测试IP(段)(默认为空)。例如1.0.0.1或者1.0.0.0/32，可重复使用测试多个IP或者IP段。
        -i, --in                    string  IP(段) 数据文件.文本文件，每一行为一个IP或者IP段。
        -m, --ping-thread           int     延时测试线程数量(默认 10)
        -t, --ping-timeout          int     延时超时时间(ms)(默认 1000ms)。当使用"--ping-via-http"时，应适当加大。
        -c, --ping-try              int     延时测试次数(默认 10)
        -p, --port                  int     测速端口(默认 443)。当使用SSL握手方式测试延时且不进行下载测试时，需要根据此
                                            参数测试；其余情况则是使用"--url"提供的参数进行测试。
            --hostname              string  SSL握手时使用的hostname(默认: "` + DefaultTestHost + `")
                                            当使用SSL握手方式测试延时且不进行下载测试时，需要根据此参数测试；其余情况则是使用
                                            "--url"提供的参数进行测试。
            --ping-via-http                 使用HTTP请求方式进行延时测试开关(默认关闭，即使用SSL握手方式测试延时)
                                            当使用此模式时，"--ping-timeout"应适当加大；另外请根据自身服务器的情
                                            况，以及CF对实际访问量的限制，降低--ping-thread值，避免访问量过大，
                                            造成测试结果偏低。
        -n, --download-thread       int     下测试线程数(默认 1)
        -d, --download-max-duration int     单次下载测速最长时间(s)(默认 10s)
        -b, --download-try          int     尝试下载次数(默认 1)
        -u, --url                   string  下载测速地址(默认 "` + DefaultTestUrl + `")。
                                            自定义下载文件建议使用压缩文件，避免CF或者HTTP容器设置压缩时使测试速度
                                            异常大；另外请在CF上关闭对此文件的缓存或者在服务器上将此文件加上用户名
                                            和密码实现访问控制，这样可以测试经过CF后到实际服务器整个链路的速度。当
                                            在服务器上对下载文件加上用户名和密码的访问控制时，可以如下格式传入url:
                                            "https://<用户名>:<密码>@cf.9999876.xyz/500mb.dat", "<用户名>"
                                            和"<密码>"请用实际值替换。
        -I  --interval              int     测试间隔时间(ms)(默认 100ms)
        -k, --delay-limit           int     平均延时上限(ms)(默认 800ms). 平均延时超过此值不计入结果集，不再进行下载测试。
        -S, --success-limit         int     延迟测试成功率下限，默认80（即延迟测试成功率不低于80%）
        -l, --speed                 float   下载平均速度下限(KB/s)(默认 2000KB/s). 下载平均速度低于此值时不计入结果集。
        -r, --result                int     测速结果集数量(默认 10). 当符合条件的IP数量超过此值时，结束测试。但是如果开启
                                            "--testall"，此值不再生效。
            --ping-only                     只进行延迟测试，不进行下载测速开关(默认关闭)
            --download-only                 不单独使用延迟测试，直接使用下载测试，但-k|--time-limit参数仍然可用来过滤结果（默认关闭）
        -6, --ipv6                          测试IPv6开关(默认关闭，即进行IPv4测试，仅不携带-i且不携带-s时有效)
        -a  --test-all                      测试全部IP开关(默认关闭，仅不携带-s且不携带-i时有效)
        -w, --store-to-file                 是否将测试结果写入文件开关(默认关闭)。当携带此参数且不携带-o参数时，输出文件名称自动生成。
        -o, --result-file           string  输出结果文件。携带此参数将结果输出至本参数对应的文件。
        -e, --store-to-db                   是否将结果存入sqlite3数据库开关（默认关闭）。  此参数打开且不携带"--db-file"参数时，
                                            数据库文件默认为"ip.db"。
        -f, --db-file               string  sqlite3数据库文件名称。携带此参数将结果输出至本参数对应的数据库文件。
        -g, --label                 string  输出结果文件后缀或者数据库中数据记录的标签  用于区分测试目标服务器。携带此参数时，
                                            在自动存储文件名模式下，文件名自 动附加此值，数据库中Lable字段为此值。但如果携带
                                            "--result-file"时，此参数对文件名无效。当不携带此参数时，自动结果文件名后缀和数
                                            据库记录的标签为"--hostname"或者"--url"对应的域名。
        -V, --debug                         调试模式
        -v, --version                       打印程序版本
    `

	flag.VarP(&ipStr, "ip", "s", "待测试IP或者地址段，例如1.0.0.1或者1.0.0.0/24")
	flag.StringVarP(&ipFile, "in", "i", "", "IP 数据文件")

	flag.IntVarP(&pingWorkerThread, "ping-thread", "m", 10, "ping测试线程数")
	flag.IntVarP(&pingTimeout, "ping-timeout", "t", 1000, "ping超时时间(ms)")
	flag.IntVarP(&pingTry, "ping-try", "c", 10, "ping测速次数")
	flag.IntVarP(&port, "port", "p", 443, "延迟测速端口")
	flag.StringVar(&hostName, "hostname", DefaultTestHost, "SSL握手对应的hostname")
	flag.BoolVar(&pingViaHttp, "ping-via-http", false, "使用连接方式进行延时测试，默认是使用SSL握手方式")

	flag.IntVarP(&downloadThread, "download-thread", "n", 1, "下测试线程数")
	flag.IntVarP(&downloadTimer, "download-max-duration", "d", 10, "单次下载测速最长时间(s)")
	flag.IntVarP(&downloadTry, "download-try", "b", 1, "尝试下载次数")
	flag.StringVarP(&urlStr, "url", "u", DefaultTestUrl, "下载测速地址")
	flag.IntVarP(&interval, "interval", "I", 100, "间隔时间(ms)")

	flag.IntVarP(&delayMax, "delay-limit", "k", 800, "平均延迟上限(ms)")
	flag.IntVarP(&successMin, "success-limit", "S", 80, "延迟测试成功率下限(%)")
	flag.Float64VarP(&speedMinStatic, "speed", "l", 6000, "下载速度下限(KB/s)")
	flag.IntVarP(&resultMax, "result", "r", 10, "测速结果集数量")

	flag.BoolVar(&disableDownload, "disable-download", false, "禁用下载测速。已废弃，请使用--ping-only。")
	flag.BoolVar(&pingOnly, "ping-only", false, "仅延迟测试，禁用速率测速")
	flag.BoolVar(&downloadOnly, "download-only", false, "直接使用速率测试，不预先使用单独的延迟测速， -k|--time-limit")
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
		println(version)
		os.Exit(0)
	}
	if disableDownload {
		pingOnly = true
		println("Warning! \"--disable-download\" is deprecated，use \"--ping-only\" instead！")
	}
	if pingOnly && downloadOnly {
		println("")
		println("--ping-only和--download-only不能同时使用")
		println("")
		println(version)
		os.Exit(1)
	}

	// initialize myLogger
	if debug {
		loggerLevel = LogLevelDebug
	} else {
		loggerLevel = LogLevelInfo
	}
	myLogger = myLogger.NewLogger(loggerLevel)
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
			if !IsValidIPs(tIp) {
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
	if pingWorkerThread <= 0 {
		pingWorkerThread = 100
	}
	if resultMax <= 0 {
		resultMax = 20
	}
	// get test ips
	if !ipv6Mode { // testAll set resultLimitation as length of source ips
		ips = GetTestIPs(srcIPS, 0)
	} else { // just for ipv6-mode
		ips = GetTestIPs(srcIPS, pingWorkerThread*100)
	}
	if testAll && !ipv6Mode {
		resultMax = len(ips)
	}
	// fix pingWorkerThread
	if len(ips) < pingWorkerThread {
		pingWorkerThread = len(ips)
	}
	// fix downloadThread
	if len(ips) < downloadThread {
		downloadThread = len(ips)
	}
	if pingTry <= 0 {
		pingTry = 60
	}
	if downloadThread <= 0 {
		downloadThread = 1
	}
	if downloadTry <= 0 {
		downloadTry = 4
	}

	if downloadTimer <= 0 {
		downloadTimer = 10
	}

	if delayMax <= 0 {
		delayMax = 9999
	}
	if interval <= 0 {
		interval = 500
	}
	if speedMinStatic < 0 {
		speedMinStatic = 0
	}

	pingTimeoutDuration = time.Duration(pingTimeout) * time.Millisecond
	// if we ping via ssl negotiation and don't perform download test, we need check hostname and port
	if !pingViaHttp && pingOnly {
		//ping via ssl negotiation
		if len(hostName) == 0 {
			myLogger.Fatal("--hostname can not be empty. \n" + help)
		}
		if port < 1 || port > 65535 {
			port = 443
		}
	} else {
		// we perform download test or just ping via http request
		hostName, port = ParseUrl(urlStr)
		HttpRspTimeoutDuration = pingTimeoutDuration
	}
	// we set HttpRspTimeoutDuration to 2 times of pingTimeoutDuration if we don't perform ping via http
	if !pingViaHttp {
		HttpRspTimeoutDuration = pingTimeoutDuration * 2
	} else {
		HttpRspTimeoutDuration = pingTimeoutDuration
	}
	downloadTimeMaxDuration = time.Duration(downloadTimer) * time.Second
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
			dbFile = DefaultDBFile
		}
	}
}

func main() {
	initRandSeed()
	var wg sync.WaitGroup

	var pingTaskChan = make(chan string, pingWorkerThread*2)
	var pingTaskOutChan = make(chan SingleVerifyResult, pingWorkerThread*2)
	var downloadTaskChan = make(chan string, downloadThread*2)
	var downloadOutChan = make(chan SingleVerifyResult, downloadThread*2)

	if pingOnly {
		if !pingViaHttp {
			myLogger.Info(fmt.Sprintf("Start Ping Test —— Ping-Via-SSL  PingRTTMax(ms):%v  Result:%v  PingThread:%v",
				delayMax, resultMax, pingWorkerThread))
		} else {
			myLogger.Info(fmt.Sprintf("Start Ping Test —— Ping-Via-HTTP  PingRTTMax(ms):%v  ResultLimit:%v  PingTestThread:%v",
				delayMax, resultMax, pingWorkerThread))
		}

	} else if downloadOnly {
		if !pingViaHttp {
			myLogger.Info(fmt.Sprintf("Start Speed Test —— PingRTTMax(ms):%v  SpeedMin(kB/s):%v  Result:%v  SpeedThread:%v\n",
				delayMax, speedMinStatic, resultMax, downloadThread))
		} else {
			myLogger.Info(fmt.Sprintf("Start Speed Test —— PingRTTMax(ms):%v  SpeedMin(kB/s):%v  Result:%v  SpeedThread:%v\n",
				delayMax, speedMinStatic, resultMax, downloadThread))
		}
	} else {
		if !pingViaHttp {
			myLogger.Info(fmt.Sprintf("Start Ping and Speed Test —— Ping-Via-SSL  PingRTTMax(ms):%v  SpeedMin(kB/s):%v  Result:%v  PingThread:%v  SpeedThread:%v\n",
				delayMax, speedMinStatic, resultMax, pingWorkerThread, downloadThread))
		} else {
			myLogger.Info(fmt.Sprintf("Start Ping and Speed Test —— Ping-Via-HTTP  PingRTTMax(ms):%v  SpeedMin(kB/s):%v  Result:%v  PingThread:%v  SpeedThread:%v\n",
				delayMax, speedMinStatic, resultMax, pingWorkerThread, downloadThread))
		}
	}
	fmt.Println()
	// start controller worker
	if !downloadOnly {
		go controllerWorker(pingTaskChan, pingTaskOutChan, downloadTaskChan, downloadOutChan, &wg)
	} else {
		go controllerWorkerDownloadOnly(downloadTaskChan, downloadOutChan, &wg)
	}

	wg.Add(1)
	// start ping worker
	if !downloadOnly {
		for i := 0; i < pingWorkerThread; i++ {
			if pingViaHttp {
				go DownloadWorker(pingTaskChan, pingTaskOutChan, &wg, urlStr, port,
					pingTimeoutDuration, downloadTimeMaxDuration, pingTry, interval, true)
			} else {
				go TcppingWorker(pingTaskChan, pingTaskOutChan, &wg, hostName, port,
					pingTimeoutDuration, pingTry, interval)
			}
			wg.Add(1)
		}
	}

	// start download worker if don't do ping only
	if !pingOnly {
		for i := 0; i < downloadThread; i++ {
			go DownloadWorker(downloadTaskChan, downloadOutChan, &wg, urlStr, port,
				HttpRspTimeoutDuration, downloadTimeMaxDuration, downloadTry, interval, false)
			wg.Add(1)
		}
	}
	wg.Wait()
	close(pingTaskChan)
	close(pingTaskOutChan)
	close(downloadTaskChan)
	close(downloadOutChan)
	verifyResultsSlice := make([]VerifyResults, 0)
	fmt.Println()
	fmt.Println()
	for _, v := range verifyResultsMap {
		verifyResultsSlice = append(verifyResultsSlice, v)
	}
	// sort by speed
	sort.Sort(sort.Reverse(ResultSpeedSorter(verifyResultsSlice)))
	myLogger.HeaderPrinted = false
	myLogger.Println(LogLevelInfo, "All Result(Reverse Order):")
	fmt.Println()
	PrintFinalStat(verifyResultsSlice, pingOnly)
}

func controllerWorker(pingTaskChan chan string, pingTaskOutChan chan SingleVerifyResult, downloadTaskChan chan string,
	downloadOutChan chan SingleVerifyResult, wg *sync.WaitGroup) {
	defer wg.Done()

	var allPingTasks, allDownloadTasks, pingTaskDone, downloadTaskDone int
	allPingTasks = 0
	allDownloadTasks = 0
	pingTaskDone = 0
	downloadTaskDone = 0
	var downloadQueueBuffer = make([]string, 0)
	var cacheResultMap = make(map[string]VerifyResults)
	var WorkReadyDone = false
	// initialize task, put ping test task to chan per ping thread
	for len(ips) > 0 && (allPingTasks-pingTaskDone) < pingWorkerThread {
		allPingTasks += 1
		pingTaskChan <- ips[0]
		if len(ips) > 1 {
			ips = ips[1:]
		} else {
			ips = []string{}
		}
	}
	myLogger.PrintOverAllStat(OverAllStat{allPingTasks, pingTaskDone,
		allDownloadTasks, downloadTaskDone, len(ips),
		len(downloadQueueBuffer), len(verifyResultsMap)})
	var OverAllStatTimer = time.Now()

	// estimated time in millisecond for single ping task,
	// estimatedTimeSinglePing := float64((pingTimeout + interval) * pingTry)
	// estimated time in millisecond for single download task,
	// estimatedTimeSingleDownload := float64((downloadTimer * 1000 + interval) * downloadTry)

LOOP:
	for {
		select {
		// check ping test result
		case pingResult := <-pingTaskOutChan:
			// if ip not test then put it into downloadTaskChan
			pingTaskDone += 1
			var tVerifyResult = singleResultStatistic(pingResult, false)
			if debug {
				myLogger.PrintSingleStat(LoggerContent{LogLevelDebug, []VerifyResults{tVerifyResult}},
					OverAllStat{allPingTasks, pingTaskDone,
						allDownloadTasks, downloadTaskDone, len(ips),
						len(downloadQueueBuffer), len(verifyResultsMap)})
				// reset timer
				OverAllStatTimer = time.Now()
			}
			if tVerifyResult.PingDurationAvg > 0.0 && tVerifyResult.PingDurationAvg <= float64(delayMax) && tVerifyResult.PingSuccessRate*100.0 >= float64(successMin) {
				if !pingOnly {
					// put ping test result to cacheResultMap for later
					// if the result is good as expected when we perform download test
					if !WorkReadyDone {
						cacheResultMap[tVerifyResult.IP] = tVerifyResult
						downloadQueueBuffer = append(downloadQueueBuffer, tVerifyResult.IP)
					}
				} else { // Download test disabled
					verifyResultsMap[tVerifyResult.IP] = tVerifyResult
					myLogger.PrintSingleStat(LoggerContent{LogLevelInfo, []VerifyResults{tVerifyResult}},
						OverAllStat{allPingTasks, pingTaskDone,
							allDownloadTasks, downloadTaskDone, len(ips),
							len(downloadQueueBuffer), len(verifyResultsMap)})
					// reset timer
					OverAllStatTimer = time.Now()
					// write result csv
					if storeToFile {
						WriteResult([]VerifyResults{tVerifyResult}, resultFile)
					}
					// write to db
					InsertIntoDb([]VerifyResults{tVerifyResult}, dbFile)
					// all work ready done
					if len(verifyResultsMap) >= resultMax {
						WorkReadyDone = true
						myLogger.PrintOverAllStat(OverAllStat{allPingTasks, pingTaskDone,
							allDownloadTasks, downloadTaskDone, len(ips),
							len(downloadQueueBuffer), len(verifyResultsMap)})
						// reset timer
						OverAllStatTimer = time.Now()
					}
				}
			}
			if pingOnly && pingTaskDone >= allPingTasks {
				WorkReadyDone = true
			}
		default:
			if pingOnly && WorkReadyDone {
				break LOOP
			}
		}
		if !pingOnly {
			// Download task control
			// put task to queue if it has enough ips
			if !WorkReadyDone && len(downloadQueueBuffer) > 0 && (allDownloadTasks-downloadTaskDone) < downloadThread {
				for i := 0; i < downloadThread; i++ {
					if len(downloadQueueBuffer) == 0 {
						break
					}
					downloadTaskChan <- downloadQueueBuffer[0]
					allDownloadTasks += 1
					if len(downloadQueueBuffer) > 1 {
						downloadQueueBuffer = downloadQueueBuffer[1:]
					} else {
						downloadQueueBuffer = []string{}
					}
				}
			}
			select {
			// check download result
			case out := <-downloadOutChan:
				downloadTaskDone += 1
				var tVerifyResult = singleResultStatistic(out, true)
				v := cacheResultMap[tVerifyResult.IP]
				// reset TestTime according download test result
				v.TestTime = tVerifyResult.TestTime
				v.DownloadCount = tVerifyResult.DownloadCount
				v.DownloadSpeedAvg = tVerifyResult.DownloadSpeedAvg
				v.DownloadSuccessCount = tVerifyResult.DownloadSuccessCount
				v.DownloadSuccessRatio = tVerifyResult.DownloadSuccessRatio
				v.DownloadSize = tVerifyResult.DownloadSize
				v.DownloadDurationSec = tVerifyResult.DownloadDurationSec
				// check when pingection duration and download speed
				if tVerifyResult.DownloadSpeedAvg >= speedMinStatic && tVerifyResult.DownloadSize > DownloadSizeMin {
					// put tVerifyResult into verifyResultsMap
					verifyResultsMap[tVerifyResult.IP] = v
					// WorkReadyDone
					if len(verifyResultsMap) >= resultMax {
						WorkReadyDone = true
					}
					// write result csv
					if storeToFile {
						WriteResult([]VerifyResults{v}, resultFile)
					}
					// write to db
					InsertIntoDb([]VerifyResults{tVerifyResult}, dbFile)
					myLogger.PrintSingleStat(LoggerContent{LogLevelInfo, []VerifyResults{v}},
						OverAllStat{allPingTasks, pingTaskDone,
							allDownloadTasks, downloadTaskDone, len(ips),
							len(downloadQueueBuffer), len(verifyResultsMap)})
					OverAllStatTimer = time.Now()
				} else if debug {
					myLogger.PrintSingleStat(LoggerContent{LogLevelDebug, []VerifyResults{v}},
						OverAllStat{allPingTasks, pingTaskDone,
							allDownloadTasks, downloadTaskDone, len(ips),
							len(downloadQueueBuffer), len(verifyResultsMap)})
					OverAllStatTimer = time.Now()
				}
			default:
				if pingTaskDone >= allPingTasks && downloadTaskDone >= allDownloadTasks && (WorkReadyDone || len(ips) == 0) {
					break LOOP
				}
			}
		}
		// Ping task control
		// when all works did not finished, and the ping task pool don't have enough ips, less than amount of ping threads
		// then we put ping task to ping queue
		if !WorkReadyDone && (allPingTasks-pingTaskDone) < pingWorkerThread {
			if len(ips) == 0 && ipv6Mode {
				ips = GetTestIPs(srcIPS, pingWorkerThread*100)
			}
			if len(ips) > 0 {
				// when don't perform download test or, download tasks pool has less than value:DownloadControlFactor*downloadThread
				// we put ping task to download queue
				// if disableDownload || (len(downloadQueueBuffer)+allDownloadTasks-downloadTaskDone) < DownloadControlFactor*downloadThread {
				// simplify algorithm
				if pingOnly || (len(downloadQueueBuffer)+allDownloadTasks-downloadTaskDone) < downloadThread {
					for i := 0; i < pingWorkerThread; i++ {
						if len(ips) == 0 {
							break
						}
						allPingTasks += 1
						pingTaskChan <- ips[0]
						if len(ips) > 1 {
							ips = ips[1:]
						} else {
							ips = []string{}
						}
					}
				}
			}
		}
		// Print overall stat during waiting time and reset OverAllStatTimer
		if time.Since(OverAllStatTimer) > time.Duration(StatisticTimer)*time.Second {
			myLogger.PrintOverAllStat(OverAllStat{allPingTasks, pingTaskDone,
				allDownloadTasks, downloadTaskDone, len(ips),
				len(downloadQueueBuffer), len(verifyResultsMap)})
			OverAllStatTimer = time.Now()
		}
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	myLogger.PrintOverAllStat(OverAllStat{allPingTasks, pingTaskDone,
		allDownloadTasks, downloadTaskDone, len(ips),
		len(downloadQueueBuffer), len(verifyResultsMap)})
	// put stop signal to all pingection workers and download worker
	for i := 0; i < pingWorkerThread; i++ {
		pingTaskChan <- WorkerStopSignal
	}
	if !pingOnly {
		for i := 0; i < downloadThread; i++ {
			downloadTaskChan <- WorkerStopSignal
		}
	}
	fmt.Println()
}

func controllerWorkerDownloadOnly(downloadTaskChan chan string,
	downloadOutChan chan SingleVerifyResult, wg *sync.WaitGroup) {
	defer wg.Done()

	var allDownloadTasks, downloadTaskDone int
	allDownloadTasks = 0
	downloadTaskDone = 0
	var cacheResultMap = make(map[string]VerifyResults)
	var WorkReadyDone = false
	if len(ips) <= 0 {
		return
	}
	// initialize task, put ping test task to chan per ping thread
	for len(ips) > 0 && (allDownloadTasks-downloadTaskDone) < downloadThread {
		allDownloadTasks += 1
		downloadTaskChan <- ips[0]
		if len(ips) > 1 {
			ips = ips[1:]
		} else {
			ips = []string{}
		}
	}
	myLogger.PrintOverAllStat(OverAllStat{0, 0,
		allDownloadTasks, downloadTaskDone, 0,
		len(ips), len(verifyResultsMap)})
	var OverAllStatTimer = time.Now()

	// estimated time in millisecond for single ping task,
	// estimatedTimeSinglePing := float64((pingTimeout + interval) * pingTry)
	// estimated time in millisecond for single download task,
	// estimatedTimeSingleDownload := float64((downloadTimer * 1000 + interval) * downloadTry)

LOOP:
	for {
		select {
		// check download result
		case out := <-downloadOutChan:
			downloadTaskDone += 1
			var tVerifyResult = singleResultStatistic(out, true)
			cacheResultMap[tVerifyResult.IP] = tVerifyResult
			// check when pingection duration and download speed
			if tVerifyResult.PingDurationAvg > 0.0 && tVerifyResult.PingDurationAvg <= float64(delayMax) && tVerifyResult.PingSuccessRate*100.0 >= float64(successMin) && tVerifyResult.DownloadSpeedAvg >= speedMinStatic && tVerifyResult.DownloadSize > DownloadSizeMin {
				// put tVerifyResult into verifyResultsMap
				verifyResultsMap[tVerifyResult.IP] = tVerifyResult
				// WorkReadyDone
				if len(verifyResultsMap) >= resultMax {
					WorkReadyDone = true
				}
				// write result csv
				if storeToFile {
					WriteResult([]VerifyResults{tVerifyResult}, resultFile)
				}
				// write to db
				InsertIntoDb([]VerifyResults{tVerifyResult}, dbFile)
				myLogger.PrintSingleStat(LoggerContent{LogLevelInfo, []VerifyResults{tVerifyResult}},
					OverAllStat{0, 0,
						allDownloadTasks, downloadTaskDone, 0,
						len(ips), len(verifyResultsMap)})
				OverAllStatTimer = time.Now()
			} else if debug {
				myLogger.PrintSingleStat(LoggerContent{LogLevelDebug, []VerifyResults{tVerifyResult}},
					OverAllStat{0, 0,
						allDownloadTasks, downloadTaskDone, 0,
						len(ips), len(verifyResultsMap)})
				OverAllStatTimer = time.Now()
			}
		default:
			if downloadTaskDone >= allDownloadTasks && (WorkReadyDone || len(ips) == 0) {
				break LOOP
			}
		}
		// Download task control
		// put task to queue if it has enough ips
		if !WorkReadyDone && len(ips) > 0 && (allDownloadTasks-downloadTaskDone) < downloadThread {
			for i := 0; i < downloadThread; i++ {
				if len(ips) == 0 {
					break
				}
				downloadTaskChan <- ips[0]
				allDownloadTasks += 1
				if len(ips) > 1 {
					ips = ips[1:]
				} else {
					ips = []string{}
				}
			}
		}
		// Print overall stat during waiting time and reset OverAllStatTimer
		if time.Since(OverAllStatTimer) > time.Duration(StatisticTimer)*time.Second {
			myLogger.PrintOverAllStat(OverAllStat{0, 0,
				allDownloadTasks, downloadTaskDone, 0,
				len(ips), len(verifyResultsMap)})
			OverAllStatTimer = time.Now()
		}
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	myLogger.PrintOverAllStat(OverAllStat{0, 0,
		allDownloadTasks, downloadTaskDone, 0,
		len(ips), len(verifyResultsMap)})
	// put stop signal to all pingection workers and download worker
	if !pingOnly {
		for i := 0; i < downloadThread; i++ {
			downloadTaskChan <- WorkerStopSignal
		}
	}
	fmt.Println()
}
