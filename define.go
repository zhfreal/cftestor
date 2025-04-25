package main

import (
	"bufio"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	utls "github.com/refraction-networking/utls"
)

const (
	workerStopSignal        = "0"
	workOnGoing         int = 1
	controllerInterval      = 100               // in millisecond
	statisticIntervalT      = 1000              // in millisecond, valid in tcell mode
	statisticIntervalNT     = 10000             // in millisecond, valid in non-tcell mode
	quitWaitingTime         = 3                 // in second
	downloadBufferSize      = 1024 * 64         // in byte
	fileDefaultSize         = 1024 * 1024 * 300 // in byte
	downloadSizeMin         = 1024 * 1024       // in byte
	defaultDLTUrl           = "https://cf.9999876.xyz/500mb.dat"
	defaultDTUrl            = "https://cf.9999876.xyz/cdn-cgi/trace"
	userAgentChrome         = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
	userAgentFirefox        = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:124.0) Gecko/20100101 Firefox/124.0"
	userAgentEdge           = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
	userAgentSafari         = "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3.1 Safari/605.1.15"

	defaultDBFile        = "ip.db"
	DefaultTestHost      = "cf.9999876.xyz"
	maxHostLen           = 1 << 12
	dtsSSL               = "SSL"
	dtsHTTPS             = "HTTPS"
	runTime              = "cftestor"
	retrieveCount   int  = 32
	TypeIPv4        int8 = 1
	TypeIPv6        int8 = 1 << 1
	TypeIPErr       int8 = 0
	DefaultPort     int  = 443
)

var (
	CFIPV4 = []string{
		"103.21.244.0/24",
		"104.16.0.0/16",
		"104.17.0.0/16",
		"104.18.0.0/16",
		"104.19.0.0/16",
		"104.20.0.0/16",
		"104.21.0.0/17",
		"104.21.192.0/19",
		"104.21.224.0/20",
		"104.22.0.0/18",
		"104.22.64.0/20",
		"104.23.128.0/20",
		"104.23.96.0/19",
		"104.24.0.0/18",
		"104.24.128.0/17",
		"104.24.64.0/19",
		"104.25.0.0/16",
		"104.26.0.0/20",
		"104.27.0.0/17",
		"104.27.192.0/20",
		"108.162.220.0/24",
		"141.101.103.0/24",
		"141.101.104.0/23",
		"141.101.106.0/24",
		"141.101.120.0/22",
		"141.101.90.0/24",
		"162.158.253.0/24",
		"162.159.128.0/21",
		"162.159.136.0/23",
		"162.159.138.0/24",
		"162.159.152.0/23",
		"162.159.160.0/24",
		"162.159.192.0/22",
		"162.159.196.0/24",
		"162.159.200.0/24",
		"162.159.204.0/24",
		"162.159.240.0/20",
		"162.159.36.0/24",
		"162.159.46.0/24",
		"172.64.128.0/19",
		"172.64.160.0/20",
		"172.64.192.0/20",
		"172.64.228.0/20",
		"172.64.68.0/24",
		"172.64.69.0/24",
		"172.64.80.0/20",
		"172.64.96.0/20",
		"172.65.0.0/18",
		"172.65.128.0/20",
		"172.65.160.0/19",
		"172.65.192.0/18",
		"172.65.96.0/19",
		"172.66.40.0/21",
		"172.67.0.0/16",
		"173.245.49.0/24",
		"188.114.96.0/22",
		"190.93.244.0/22",
		"198.41.192.0/20",
		"198.41.208.0/23",
		"198.41.211.0/24",
		"198.41.212.0/24",
		"198.41.214.0/23",
		"198.41.216.0/21",
	}
	CFIPV4FULL = []string{
		"103.21.244.0/22",
		"103.22.200.0/22",
		"103.31.4.0/22",
		"104.16.0.0/13",
		"104.24.0.0/14",
		"108.162.192.0/18",
		"131.0.72.0/22",
		"141.101.64.0/18",
		"162.158.0.0/15",
		"172.64.0.0/13",
		"173.245.48.0/20",
		"188.114.96.0/20",
		"190.93.240.0/20",
		"197.234.240.0/22",
		"198.41.128.0/17",
	}
	CFIPV6 = []string{
		"2606:4700:f1::/48",
		"2606:4700:f4::/48",
		"2606:4700:130::/44",
		"2606:4700:3000::/44",
		"2606:4700:3010::/44",
		"2606:4700:3020::/44",
		"2606:4700:3030::/44",
		"2606:4700:4400::/44",
		"2606:4700:4700::/48",
		"2606:4700:7000::/48",
		"2606:4700:8040::/44",
		"2606:4700:80c0::/44",
		"2606:4700:80f0::/44",
		"2606:4700:81c0::/44",
		"2606:4700:8390::/44",
		"2606:4700:83b0::/44",
		"2606:4700:85c0::/44",
		"2606:4700:85d0::/44",
		"2606:4700:8ca0::/44",
		"2606:4700:8d70::/44",
		"2606:4700:8d90::/44",
		"2606:4700:8dd0::/44",
		"2606:4700:8de0::/44",
		"2606:4700:90c0::/44",
		"2606:4700:90d0::/44",
		"2606:4700:91b0::/44",
		"2606:4700:9640::/44",
		"2606:4700:9760::/44",
		"2606:4700:99e0::/44",
		"2606:4700:9ae0::/44",
		"2606:4700:9c60::/44",
		"2a06:98c1:3100::/44",
		"2a06:98c1:3120::/48",
		"2a06:98c1:3121::/48",
		"2a06:98c1:3122::/48",
		"2a06:98c1:3123::/48",
		"2606:4700::6810:0/111",
		"2606:4700::6812:0/111",
		"2606:4700:10::6814:0/112",
		"2606:4700:10::6816:0/112",
		"2606:4700:10::6817:0/112",
	}
	CFIPV6FULL = []string{
		"2400:cb00::/32",
		"2606:4700::/32",
		"2803:f800::/32",
		"2405:b500::/32",
		"2405:8100::/32",
		"2a06:98c0::/29",
		"2c0f:f248::/32",
	}
	utf8BomBytes    = []byte{0xEF, 0xBB, 0xBF}
	resultCsvHeader = []string{
		"TestTime",
		"IP",
		"DLSpeed(DLS,KB/s)",
		"DelayAvg(DA,ms)",
		"DelaySource(DS)",
		"DTPassedRate(DTPR,%)",
		"DTCount(DTC)",
		"DTPassedCount(DTPC)",
		"DelayMin(DMI,ms)",
		"DelayMax(DMX,ms)",
		"DLTCount(DLTC)",
		"DLTPassedCount(DLTPC)",
		"DLTPassedRate(DLPR,%)",
		"City(Src)",
		"ASN(Src)",
		"Location(CF)",
	}
	// cfURL = "https://speed.cloudflare.com/__down"
	baseCfCDNCgiTraceUrl = "https://ww1.zhfreal.top/cdn-cgi/trace"
)

var (
	maxHostLenBig                           = big.NewInt(maxHostLen)
	ipFile                                  string
	version, buildTag, buildDate, buildHash string = "dev", "dev", "dev", "dev"
	// srcIPRsRaw                              []*ipRange // CIDR slice
	// srcIPRsExtracted                        []net.IP   // net.IP slice
	// srcHosts                                []*string  // slice stored host: <ip>:<port>
	ipStr                                  []string
	dtCount, dtWorkerThread                int
	dltDurMax, dltWorkerThread             int
	dltCount, resultMin                    int
	interval, dtEvaluationDelay, dtTimeout int
	dtStdExp                               float64
	hostName, dltUrl, dtSource, dtUrl      string
	dltTimeout, loop                       int
	loopInterval                           int // in seconds
	dtEvaluationDTPR, dltEvaluationSpeed   float64
	dtHttps, disableDownload               bool
	dtVia                                  string
	dtHttpRspReturnCodeExpected            int
	enableDTEvaluation                     bool
	ipv4Mode, ipv6Mode, dtOnly, dltOnly    bool
	tlsClientID                            utls.ClientHelloID = utls.HelloChrome_Auto
	userAgent                              string             = userAgentChrome
	storeToFile, storeToDB, testAll, debug bool
	resolveLocalASNAndCity, enableStdEv    bool
	resultFile, suffixLabel, dbFile        string
	myLogger                               MyLogger
	loggerLevel                            LogLevel
	httpRspTimeoutDuration                 time.Duration
	dtTimeoutDuration                      time.Duration
	dltDurationInTotal                     time.Duration
	verifyResultsMap                       = make(map[string]VerifyResults)
	myRand                                 = newRand()
	srcIPs                                 = NewSourceIPsWithRand(myRand)
	portStrSlice                           []string
	// LoopStatus                             *SafeLooper
	// titleRuntime                            *string
	// titlePre                                [2][4]string
	// titleTasksStat                          [2]*string
	// detailTitleSlice                        []string
	// resultStrSlice, debugStrSlice           [][]*string
	// termAll                                 *tcell.Screen
	// titleStyle                              = tcell.StyleDefault.Foreground(tcell.ColorBlack.TrueColor()).Background(tcell.ColorWhite)
	// normalStyle                             = tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	// titleStyleCancel                        = tcell.StyleDefault.Foreground(tcell.ColorBlack.TrueColor()).Background(tcell.ColorGray)
	// contentStyle                            = tcell.StyleDefault
	// maxResultsDisplay                       = 10
	// maxDebugDisplay                         = 10
	// titleRuntimeRow                         = 0
	// titlePreRow                             = titleRuntimeRow + 2
	// titleCancelRow                          = titlePreRow + 3
	// titleTasksStatRow                       = titleCancelRow + 2
	// titleResultHintRow                      = titleTasksStatRow + 2
	// titleResultRow                          = titleResultHintRow + 1
	// titleDebugHintRow                       = titleResultRow + maxResultsDisplay + 2
	// titleDebugRow                           = titleDebugHintRow + 1
	// titleCancel                             = "Press ESC to cancel!"
	// titleCancelConfirm                      = "Press ENTER to confirm; Any other key to back!"
	// titleWaitQuit                           = "Waiting for exit..."
	// titleResultHint                         = "Result:"
	// titleDebugHint                          = "Debug Msg:"
	// cancelSigFromTerm                       = false
	// terminateConfirm                        = false
	// resultStatIndent                        = 9
	// dtThreadsNumLen, dltThreadsNumLen       = 0, 0
	// tcellMode   = false
	fastMode    = false
	silenceMode = false
	ResolveLoc  = false
	// statInterval                                   = statisticIntervalNT
	appArt string = `
  ░█▀▀░█▀▀░▀█▀░█▀▀░█▀▀░▀█▀░█▀█░█▀▄
  ░█░░░█▀▀░░█░░█▀▀░▀▀█░░█░░█░█░█▀▄
  ░▀▀▀░▀░░░░▀░░▀▀▀░▀▀▀░░▀░░▀▀▀░▀░▀
`
)

var help = `Usage: ` + runTime + ` [options]
Options:
	-s, --ip            string          Specify IP, CIDR, or host for testing. Examples: "-s 1.0.0.1", "-s 1.0.0.1/32",
                                        "-s 1.0.0.1/24", "-s 1.1.1.1:2053".
	-i, --in            string          Specify a file for testing, containing multiple lines. Each line
                                        represents one IP, CIDR, or host.
	-p, --port          int             Specify port(s) to test. Can be a single port or a range, e.g.,
                                        "-p 443-800,1000:1300;8443|8444 -p 10000-12000|13333".
                                        Ports should support SSL/TLS/HTTPS protocols. Default is 443.
	-m, --dt-thread     int             Number of concurrent threads for Delay Test (DT). Default is 20 threads.
	-t, --dt-timeout    int             Timeout for a single DT in milliseconds. Default is 2000ms for SSL mode
                                        and 5000ms for HTTPS mode. Should not be less than "-k|--evaluate-dt-delay".
	-c, --dt-count      int             Number of DT attempts per IP. Default is 2.
		--hostname      string          Hostname for DT testing. Valid when "--dt-only" is off and
                                        "--dt-via https" is not provided.
		--dt-via <https|tls|ssl>        Perform DT via HTTPS, TLS, or SSL. Default is HTTPS.
		--dt-expect-code <status_code>  Expected HTTP status code for DT testing when using '--dt-via https'.
                                        Default is 200.
		--dt-url        string          Specify a test URL for DT. Ensure the URL is valid, reachable, and uses HTTPS.
                                        For custom ports, use "--port" or provide them with IPs using "-s" or "-i".
		--ev-dt                         Enable DT evaluation. Evaluates delay using "-c|--dt-count <value>".
                                        Without this, DT stops after the first successful attempt. When enabled,
                                        results are evaluated using "-k|--evaluate-dt-delay <value>" and
                                        "-S|--evaluate-dt-dtpr <value>". Default is off.
	-k, --ev-dt-delay   int             Maximum delay for a single DT in milliseconds. Default is 600ms.
		--ev-dt-dtpr    float           Minimum DT pass rate as a percentage. Default is 100% (all DTs must pass).
		--ev-dt-std     float           Expected standard deviation for DT evaluation. If enabled and set to a value
                                        greater than 0, delays are evaluated against this value.
	-n, --dlt-thread    int             Number of concurrent threads for Download Test (DLT). Default is 1.
	-d, --dlt-period    int             Total duration for a single DLT in seconds. Default is 10s.
	-b, --dlt-count     int             Number of DLT attempts per IP. Default is 1.
	-u, --dlt-url       string          Specify a test URL for DLT. Ensure the URL is valid, reachable, and uses HTTPS.
                                        For custom ports, use "--port" or provide them with IPs using "-s" or "-i".
		--dlt-timeout   int             Timeout for HTTP response during DLT in milliseconds. Default is 5000ms.
	-I, --interval      int             Interval between tests in milliseconds. Default is 500ms.
	-l, --speed         float           Minimum download speed filter in KB/s. Default is 6000KB/s.
	-r, --result        int             Maximum number of qualified IPs. Default is 10. Ignored if "--test-all" is set.
		--dt-only                       Perform only DT. By default, both DT and DLT are performed.
		--dlt-only                      Perform only DLT. By default, both DT and DLT are performed.
		--fast                          Enable fast mode using internal IPs for quick detection. Works only when
                                        neither "-s/--ip" nor "-i/--in" is provided. Default is off.
		--loop          int             Number of loop rounds. Default is disabled.
		--loop-interval int             Interval between loops in seconds. Default is 60s.
	-4, --ipv4                          Test only IPv4. If no IPs are specified with "-s" or "-i", tests IPv4
                                        using built-in CloudFlare IPs by default.
	-6, --ipv6                          Test only IPv6. If no IPs are specified with "-s" or "-i", tests IPv6
                                        using built-in CloudFlare IPs with this flag.
		--hello-firefox                 Simulate Firefox for TLS/HTTPS.
		--hello-chrome                  Simulate Chrome for TLS/HTTPS.
		--hello-edge                    Simulate Microsoft Edge for TLS/HTTPS.
		--hello-safari                  Simulate Safari for TLS/HTTPS.
	-a, --test-all                      Test all IPs until none are left. Default is off.
	-w, --to-file                       Write results to a CSV file. Default is off. If enabled and
                                        "-o|--result-file <value>" is not provided, the file is named
                                        "Result_<YYYYMMDDHHMISS>-<HOSTNAME>.csv" and stored in the current directory.
	-o, --out-file      string          Specify the result file name. If not provided and "-w|--to-file" is enabled,
                                        the file is named "Result_<YYYYMMDDHHMISS>-<HOSTNAME>.csv" and stored in
                                        the current directory.
	-e, --to-db                         Write results to an SQLite3 database. Default is off. If enabled and
                                        "-f|--db-file" is not provided, the file is named "ip.db" and stored in
                                        the current directory.
		--local-asn                     Retrieve local ASN and city information. Default is off.
		--resolve-loc                   Attempt to resolve location.
	-f, --db-file       string          Specify the SQLite3 database file name. If not provided and "-e|--to-db"
                                        is enabled, the file is named "ip.db" and stored in the current directory.
	-g, --label         string          Label for the result file name and SQLite3 record. Defaults to the hostname
                                        from "--hostname" or "-u|--url".
	-S, --silence                       Enable silence mode.
	-V, --debug                         Print debug messages. Activates "--debug".
	-v, --version                       Display version information.
`

// type arrayFlags []string

// func (i *arrayFlags) String() string {
// 	return fmt.Sprintf("%v", *i)
// }

// func (i *arrayFlags) Set(value string) error {
// 	*i = append(*i, value)
// 	return nil
// }

// func (i *arrayFlags) Type() string {
// 	return "[]string"
// }

type singleResult struct {
	dTPassed      bool          // Delay Test (DT) passed (yes) or not (no)
	dTDuration    time.Duration // DT time escaped
	httpReqRspDur time.Duration // pure time escaped between http request send and response after tls negotiation
	dLTWasDone    bool          // Download Test (DLT) was done or not
	dLTPassed     bool          // DLT passed or not
	dLTDuration   time.Duration // DLT escaped times
	dLTDataSize   int64         // DLT download data size, in byte
}

type singleVerifyResult struct {
	testTime    time.Time
	host        string
	loc         string
	resultSlice []singleResult
}

type VerifyResults struct {
	testTime time.Time // test time
	ip       *string   // should be <ipv4:port> or <[ipv6]:port>, not just a ip string.
	loc      *string
	dtc      int       // Delay Test(DT) tried count
	dtpc     int       // DT passed count
	dtpr     float64   // DT passed rate, in decimal
	da       float64   // average delay, in ms
	daVar    float64   // variance of average delay
	daStd    float64   // standard deviation of average delay
	dmi      float64   // minimal delay, in ms
	dmx      float64   // max delay, in ms
	dltc     int       // Download Test(DLT) tried count
	dltpc    int       // DLT passed count
	dltpr    float64   // DLT passed rate, in decimal
	dls      float64   // DLT average speed, in KB/s
	dlds     int64     // DLT download data size, in byte
	dltd     float64   // DLT escaped times, in second
	dtDList  []float64 // Delay Test(DT) duration list in milliseconds
}

// combine b into a
// this function will combine the delay test results of b into a.
// It will add the tried count, passed count, and update the passed rate.
// It will also update the average delay, minimal delay, and max delay.
func (a *VerifyResults) combine(b VerifyResults) {
	if a.ip == nil || b.ip == nil || *a.ip != *b.ip {
		return
	}
	if a.testTime.Before(b.testTime) {
		a.testTime = b.testTime
	}
	if b.loc != nil && len(*b.loc) != 0 && (a.loc == nil || len(*a.loc) == 0) {
		a.loc = b.loc
	}
	a.dtc += b.dtc
	a.dtpc += b.dtpc
	if a.dtc > 0 {
		a.dtpr = float64(a.dtpc) / float64(a.dtc)
	}
	a.dtDList = append(a.dtDList, b.dtDList...)
	// remove 0 time form a.dtDList
	for i := 0; i < len(a.dtDList); i++ {
		if a.dtDList[i] == 0 {
			a.dtDList = append(a.dtDList[:i], a.dtDList[i+1:]...)
			i--
		}
	}
	totalDelay := 0.0
	for _, v := range a.dtDList {
		totalDelay += v
	}
	if a.dtpc > 0 && len(a.dtDList) > 0 {
		a.da = totalDelay / float64(len(a.dtDList))
	}
	a.daStd = std(a.dtDList)
	a.daVar = variance(a.dtDList)
	if a.dmi > b.dmi && b.dtpc > 0 {
		a.dmi = b.dmi
	}
	if a.dmx < b.dmx && b.dtpc > 0 {
		a.dmx = b.dmx
	}
	a.dltc += b.dltc
	a.dltpc += b.dltpc
	if a.dltc > 0 {
		a.dltpr = float64(a.dltpc) / float64(a.dltc)
	}
	a.dlds += b.dlds
	a.dltd += b.dltd
	if a.dltpc > 0 && a.dltd > 0 {
		a.dls = float64(a.dlds) / float64(a.dltd) / 1000
	}
}

// do deep copy from original VerifyResults obj into brand new one
// func (a *VerifyResults) copy() VerifyResults {
// 	tIp := *a.ip
// 	tLoc := *a.loc
// 	tDtDList := make([]float64, len(a.dtDList))
// 	copy(tDtDList, a.dtDList)
// 	return VerifyResults{
// 		testTime: a.testTime,
// 		ip:       &tIp,
// 		loc:      &tLoc,
// 		dtc:      a.dtc,
// 		dtpc:     a.dtpc,
// 		dtpr:     a.dtpr,
// 		da:       a.da,
// 		daVar:    a.daVar,
// 		daStd:    a.daStd,
// 		dmi:      a.dmi,
// 		dmx:      a.dmx,
// 		dltc:     a.dltc,
// 		dltpc:    a.dltpc,
// 		dltpr:    a.dltpr,
// 		dls:      a.dls,
// 		dlds:     a.dlds,
// 		dltd:     a.dltd,
// 		dtDList:  tDtDList,
// 	}

// }

type resultSpeedSorter []VerifyResults

func (a resultSpeedSorter) Len() int           { return len(a) }
func (a resultSpeedSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a resultSpeedSorter) Less(i, j int) bool { return a[i].dls < a[j].dls }

type overAllStat struct {
	dtTasksDone  int
	dtOnGoing    int
	dtCached     int
	dltTasksDone int
	dltOnGoing   int
	dltCached    int
	resultCount  int
	remain       int
}

type ipRange struct {
	IPStart   net.IP
	IPEnd     net.IP
	Len       *big.Int
	Extracted bool
}

func (ipr *ipRange) isValid() bool {
	if ipr == nil || ipr.IPStart == nil || ipr.IPEnd == nil || ipr.Extracted {
		return false
	}
	if len(ipr.IPStart) != len(ipr.IPEnd) {
		return false
	} else if len(ipr.IPStart) != net.IPv4len && len(ipr.IPStart) != net.IPv6len {
		return false
	} else {
		for i := 0; i < len(ipr.IPStart); i++ {
			if (ipr.IPStart)[i] > (ipr.IPEnd)[i] {
				return false
			}
		}
	}
	return true
}

func (ipr *ipRange) IsValid() bool {
	return ipr.isValid()
}

func (ipr *ipRange) length() *big.Int {
	if !ipr.isValid() {
		return big.NewInt(0)
	}
	var newLenBytes = make([]byte, len(ipr.IPEnd), cap(ipr.IPEnd))
	reduce := 0
	for i := len(ipr.IPStart) - 1; i >= 0; i-- {
		m := (ipr.IPStart)[i]
		n := (ipr.IPEnd)[i]
		newValue := int(n) - int(m) - reduce
		// n < m + reduce, borrow from i - 1
		if newValue < 0 {
			reduce = 1
			newValue += int(1 << 8)
		} else {
			// reset reduce
			reduce = 0
		}
		newLenBytes[i] = byte(newValue)
	}
	newLen := big.NewInt(0).SetBytes(newLenBytes)
	// add 1 more
	newLen = newLen.Add(newLen, big.NewInt(1))
	return newLen
}

func (ipr *ipRange) Length() *big.Int {
	return ipr.length()
}

func (ipr *ipRange) isV4() bool {
	if !ipr.isValid() {
		return false
	}
	return len(ipr.IPStart) == net.IPv4len
}

func (ipr *ipRange) isV6() bool {
	if !ipr.isValid() {
		return false
	}
	return len(ipr.IPStart) == net.IPv6len
}

func (ipr *ipRange) IsV4() bool {
	return ipr.isV4()
}

func (ipr *ipRange) IsV6() bool {
	return ipr.isV6()
}

func (ipr *ipRange) init(StartIP net.IP, EndIP net.IP) *ipRange {
	t_s_startIP := StartIP
	if t_s_startIP.To4() != nil {
		t_s_startIP = t_s_startIP.To4()
	}
	t_s_endIP := EndIP
	if t_s_endIP.To4() != nil {
		t_s_endIP = t_s_endIP.To4()
	}
	ipr.IPStart = t_s_startIP
	ipr.IPEnd = t_s_endIP

	ipr.Extracted = false
	if ipr.isValid() {
		ipr.Len = ipr.length()
		return ipr
	}
	return nil
}

func (ipr *ipRange) String() string {
	if !ipr.isValid() {
		return "null"
	}
	return fmt.Sprintf("Start With: %s; End With: %s; Length: %s; Extracted: %t",
		(ipr.IPStart).String(), (ipr.IPEnd).String(), (ipr.length()).String(), ipr.Extracted)
}

func (ipr *ipRange) Extract(num int) (IPList []net.IP) {
	if !ipr.isValid() {
		return
	}
	// num should greater than 0
	if num <= 0 {
		return
	}
	// no more ip for extracted
	if ipr.Extracted || ipr.Len.Cmp(big.NewInt(0)) == 0 {
		return
	}
	numBig := big.NewInt(int64(num))
	size := ipr.length()
	// no enough IPs to extract
	if size.Cmp(numBig) == -1 {
		num = int(size.Int64())
	}
	newIP := ipr.IPStart
	IPList = append(IPList, newIP)
	num--
	for num > 0 {
		num_in_bytes := makeBytes(uint(1), len(newIP))
		newIP = ipShift(newIP, num_in_bytes)
		// some error shown
		if newIP == nil {
			return
		}
		IPList = append(IPList, newIP)
		num--
	}
	// reset IPStart and Extracted
	// no more IP between newIP and *IPEnd, set Extracted to true
	if newIP.Equal(ipr.IPEnd) {
		ipr.Extracted = true
		ipr.IPStart = newIP
		ipr.Len = big.NewInt(0)
	} else {
		// reset *IPStart to newIP + 1
		num_in_bytes := makeBytes(uint(1), len(newIP))
		ipr.IPStart = ipShift(newIP, num_in_bytes)
		ipr.Len = ipr.length()
	}
	return
}

func (ipr *ipRange) ExtractReverse(num int) (IPList []net.IP) {
	if !ipr.isValid() {
		return
	}
	// num should greater than 0
	if num <= 0 {
		return
	}
	// no more ip for extracted
	if ipr.Extracted || ipr.Len.Cmp(big.NewInt(0)) == 0 {
		return
	}
	numBig := big.NewInt(int64(num))
	size := ipr.length()
	// no enough IPs to extract
	if size.Cmp(numBig) == -1 {
		return
	}
	newIP := ipr.IPEnd
	IPList = append(IPList, newIP)
	num--
	for num > 0 {
		num_in_bytes := makeBytes(uint(1), len(newIP))
		newIP = ipShiftReverse(newIP, num_in_bytes)
		// some error shown
		if newIP == nil {
			return
		}
		IPList = append(IPList, newIP)
		num--
	}
	// reset IPStart and Extracted
	// no more IP between *IPStart and newIP, set Extracted to true
	if newIP.Equal(ipr.IPStart) {
		ipr.Extracted = true
		ipr.Len = big.NewInt(0)
		ipr.IPEnd = newIP
	} else {
		// reset *IPEnd to newIP - 1
		num_in_bytes := makeBytes(uint(1), len(newIP))
		ipr.IPEnd = ipShiftReverse(newIP, num_in_bytes)
		ipr.Len = ipr.length()
	}
	return
}

func (ipr *ipRange) ExtractAll() (IPList []net.IP) {
	// we limit the max result length to MaxHostLen (currently, 65536), if it's to big, return nil
	// or it's don't have any IPS to extract, return nil
	if ipr.Extracted || ipr.Len.Cmp(big.NewInt(0)) == 0 || ipr.Len.Cmp(big.NewInt(maxHostLen)) == 1 {
		return
	}
	return ipr.Extract(int(ipr.Len.Int64()))
}

func (ipr *ipRange) GetRandomX(num int) (IPList []net.IP) {
	// or it's don't have any IPS to extract, return nil
	if ipr.Extracted || ipr.Len.Cmp(big.NewInt(0)) == 0 {
		return
	}
	// we extract all while ipr don't have enough ips for extracted
	if big.NewInt(int64(num)).Cmp(ipr.Len) >= 0 {
		m := ipr.ExtractAll()
		if m == nil {
			return
		}
		for i := 0; i < len(m); i++ {
			IPList = append(IPList, m[i])
		}
		// shuffle
		myRand.Shuffle(len(IPList), func(i, j int) {
			IPList[i], IPList[j] = IPList[j], IPList[i]
		})
		// we done here
		return
	}
	// get randomly
	i := 0
	for i < num {
		n := big.NewInt(0)
		n = n.Rand(myRand, ipr.Len)
		num_in_bytes := fillBytes(n.Bytes(), len(ipr.IPStart))
		newIP := ipShift(ipr.IPStart, num_in_bytes)
		if newIP != nil {
			IPList = append(IPList, newIP)
			i++
		}
	}
	return
}

type SafeLooper struct {
	mu sync.Mutex
	// t: target loop rounds, t <= 0 means disabled
	// c: current loop round, when t > 0 and c == 0 means loop is valid but not start yet
	// when t > 0 and c >= 1 and c<= t means loop is running, and in round c
	// when t > 0 and c > t means loop is done
	t, c     int
	interval int // interval in milliseconds
}

func (s *SafeLooper) Valid() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status() > -1
}

func (s *SafeLooper) Loop() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.t <= 0 {
		return false
	}
	if s.c >= s.t {
		return false
	}
	s.c++
	return true
}

func (s *SafeLooper) Ready() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status() == 0
}

func (s *SafeLooper) InLooping() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status() == 1
}

func (s *SafeLooper) Finished() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status() == 2
}

// Status returns -1 if the SafeLooper is not valid, 0 if it has just been Enabled,
// 1 if it is looping and 2 if it has finished looping.
func (s *SafeLooper) status() int {
	if s.t <= 0 {
		return -1
	}
	if s.c == 0 {
		return 0
	}
	if s.c <= s.t {
		return 1
	}
	return 2
}

func (s *SafeLooper) Status() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status()
}

func (s *SafeLooper) Ok() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status() > 0
}

func (s *SafeLooper) SetInterval(interval int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interval = interval
}

func (s *SafeLooper) GetInterval() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.interval
}

func (s *SafeLooper) GetRound() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.c
}

func (s *SafeLooper) Sleep() {
	s.mu.Lock()
	defer s.mu.Unlock()
	time.Sleep(time.Duration(s.interval) * time.Millisecond)
}

func (s *SafeLooper) SleepInterval(interval int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	time.Sleep(time.Duration(interval) * time.Millisecond)
}

func NewSafeLooper(t int) *SafeLooper {
	c := 0
	if t <= 0 {
		c = -1
	}
	return &SafeLooper{
		t:        t,
		c:        c,
		interval: 1000,
	}
}

func NewSafeLooperWithInterval(t, interval int) *SafeLooper {
	s := NewSafeLooper(t)
	s.SetInterval(interval)
	return s
}

type sourceIPs struct {
	mu               sync.Mutex
	srcHosts         []*string
	srcIPRsRaw       []*ipRange
	srcIPRsExtracted []net.IP
	ports            []int
	tRnd             *rand.Rand
}

func (s *sourceIPs) Len() *big.Int {
	s.mu.Lock()
	defer s.mu.Unlock()
	t_qty := big.NewInt(0)
	for i := 0; i < len(s.srcIPRsRaw); i++ {
		t_qty = t_qty.Add(t_qty, s.srcIPRsRaw[i].Length())
	}
	t_qty = t_qty.Add(t_qty, big.NewInt(int64(len(s.srcIPRsExtracted))))
	t_qty = t_qty.Add(t_qty, big.NewInt(int64(len(s.srcHosts))))
	return t_qty
}

func (s *sourceIPs) LenInt() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	t_qty := 0
	t_qty += len(s.srcIPRsRaw)
	t_qty += len(s.srcIPRsExtracted)
	t_qty += len(s.srcHosts)
	return t_qty
}

func (s *sourceIPs) add(IPs string, mode int8) error {
	ips := strings.TrimSpace(IPs)
	ips = strings.Split(ips, "#")[0]
	if isValidIPs(ips) {
		tV := getIPsVer(ips)
		if tV == TypeIPErr {
			return fmt.Errorf("\"%v\" is invalid", ips)
		}
		// when IPs is not the target version, return without any error
		if (tV & mode) != tV {
			return nil
		}
		ipr := NewIPRangeFromCIDR(&ips)
		if ipr == nil {
			return fmt.Errorf("\"%v\" is invalid", ips)
		}
		// when it do not testAll and ipr is not bigger than maxHostLenBig, extract to to cache
		if ipr.Len.Cmp(maxHostLenBig) < 1 {
			s.srcIPRsExtracted = append(s.srcIPRsExtracted, ipr.ExtractAll()...)
		} else {
			// when it do not perform tealAll or not bigger than maxHostLenBig, just put it to srcIPRs
			s.srcIPRsRaw = append(s.srcIPRsRaw, ipr)
		}
	} else if isValidHost(ips) {
		tV := getHostVer(ips)
		if tV == TypeIPErr {
			return fmt.Errorf("\"%v\" is invalid", ips)
		}
		// when IPs is not the target version, return without any error
		if (tV & mode) != tV {
			return nil
		}
		s.srcHosts = append(s.srcHosts, &ips)
	} else {
		return fmt.Errorf("the input \"%v\" is neither a valid IP, CIDR, nor a host", ips)
	}
	return nil
}

func (s *sourceIPs) Add(IPs string, mode int8) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.add(IPs, mode)
}

func (s *sourceIPs) AddFromSlice(ipsSlice []string, mode int8) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ips := range ipsSlice {
		err := s.add(ips, mode)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *sourceIPs) AddFromFile(filename string, mode int8) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tFile, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("file \"%s\" is not accessible", filename)
	}
	scanner := bufio.NewScanner(tFile)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		tIp := strings.TrimSpace(scanner.Text())
		if len(tIp) == 0 {
			continue
		}
		err := s.add(tIp, mode)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *sourceIPs) AddPorts(srcPorts []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	port_regex := regexp.MustCompile(`[,;|]+`)
	port_range_regex := regexp.MustCompile(`\d+[-:]\d+`)
	port_range_split_regex := regexp.MustCompile(`[-:]`)
	if len(srcPorts) > 0 {
		for _, portStr := range srcPorts {
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
						s.ports = append(s.ports, i)
					}
				} else { // it's a single port
					port, err := strconv.Atoi(t_port_str)
					if err != nil || port < 1 || port > 65535 {
						myLogger.Fatalf("\"-p|--port %v\" is invalid!\n", t_port_str)
					}
					s.ports = append(s.ports, port)
				}
			}
		}

	}
	if len(s.ports) == 0 {
		s.ports = append(s.ports, DefaultPort)
	}
	// clean ports, make them unique
	s.ports = uniqueIntSlice(s.ports)
}

func (s *sourceIPs) RetrieveSome(amount int, isRand bool) (targetIPs []*string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var t_target = []*string{}
	targetIPs = append(targetIPs, s.retrieveHosts(amount)...)
	t_target = append(t_target, s.retrieveIPsFromIPR(amount, isRand)...)
	for _, ipStr := range t_target {
		for _, port := range s.ports {
			host := genHostFromIPStrPort(*ipStr, port)
			if len(host) > 0 {
				targetIPs = append(targetIPs, &host)
			}
		}
	}
	return
}

func (s *sourceIPs) retrieveHosts(amount int) (targetHosts []*string) {
	if amount <= 0 || len(s.srcHosts) == 0 {
		return
	}
	t_amount := amount
	if len(s.srcHosts) < amount {
		t_amount = len(s.srcHosts)
	}
	targetHosts = append(targetHosts, s.srcHosts[:t_amount]...)
	if len(s.srcHosts) <= amount {
		s.srcHosts = []*string{}
	} else {
		s.srcHosts = s.srcHosts[t_amount:]
	}
	return
}

func (s *sourceIPs) retrieveIPsFromIPR(amount int, isRandom bool) (targetIPs []*string) {
	if amount <= 0 || amount < retrieveCount {
		amount = retrieveCount
	}

	t_ips := []net.IP{}
	for _, ipr := range s.srcIPRsRaw {
		if isRandom {
			t_ips = append(t_ips, ipr.GetRandomX(amount)...)
		} else {
			t_ips = append(t_ips, ipr.Extract(amount)...)
		}
	}
	if len(s.srcIPRsExtracted) > 0 {
		if len(s.srcIPRsExtracted) <= amount {
			t_ips = append(t_ips, s.srcIPRsExtracted...)
			s.srcIPRsExtracted = []net.IP{}
		} else {
			t_ips = append(t_ips, s.srcIPRsExtracted[0:amount]...)
			s.srcIPRsExtracted = s.srcIPRsExtracted[amount:]
		}
	}
	for _, t_ip := range t_ips {
		tIP := t_ip.String()
		targetIPs = append(targetIPs, &tIP)
	}
	// randomize
	myRand.Shuffle(len(targetIPs), func(m, n int) {
		targetIPs[m], targetIPs[n] = targetIPs[n], targetIPs[m]
	})
	return
}

func (s *sourceIPs) Shuffle() {
	s.mu.Lock()
	defer s.mu.Unlock()
	myRand.Shuffle(len(s.srcHosts), func(m, n int) {
		s.srcHosts[m], s.srcHosts[n] = s.srcHosts[n], s.srcHosts[m]
	})
	myRand.Shuffle(len(s.srcIPRsRaw), func(m, n int) {
		s.srcIPRsRaw[m], s.srcIPRsRaw[n] = s.srcIPRsRaw[n], s.srcIPRsRaw[m]
	})
	myRand.Shuffle(len(s.srcIPRsExtracted), func(m, n int) {
		s.srcIPRsExtracted[m], s.srcIPRsExtracted[n] = s.srcIPRsExtracted[n], s.srcIPRsExtracted[m]
	})
}

func (s *sourceIPs) SetRand(mRnd *rand.Rand) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tRnd = mRnd
}

// After reset, all will be empty, so you should using Add(), AddFromSlice(), AddFromFile(), and AddPorts
// to initialize
func (s *sourceIPs) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.srcHosts = []*string{}
	s.srcIPRsRaw = []*ipRange{}
	s.srcIPRsExtracted = []net.IP{}
	s.ports = []int{}
}

func NewSourceIPs() *sourceIPs {
	return &sourceIPs{
		srcHosts:         make([]*string, 0),
		srcIPRsRaw:       make([]*ipRange, 0),
		srcIPRsExtracted: make([]net.IP, 0),
		ports:            []int{},
		tRnd:             newRand(),
	}
}

func NewSourceIPsWithRand(tRnd *rand.Rand) *sourceIPs {
	mSrc := NewSourceIPs()
	mSrc.SetRand(tRnd)
	return mSrc
}

func CopySourceIPs(src *sourceIPs) *sourceIPs {
	mSrc := NewSourceIPs()
	mSrc.srcHosts = append(mSrc.srcHosts, src.srcHosts...)
	mSrc.srcIPRsRaw = append(mSrc.srcIPRsRaw, src.srcIPRsRaw...)
	mSrc.srcIPRsExtracted = append(mSrc.srcIPRsExtracted, src.srcIPRsExtracted...)
	mSrc.ports = append(mSrc.ports, src.ports...)
	mSrc.tRnd = newRand()
	return mSrc
}

type task struct {
	host        *string
	max_failure int
}

func (t *task) GetHost() *string {
	return t.host
}

func (t *task) GetMaxFailure() int {
	return t.max_failure
}

func NewTask(host *string, max_failure int) *task {
	return &task{
		host:        host,
		max_failure: max_failure,
	}
}
