package main

import (
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"time"

	"github.com/gdamore/tcell/v2"
	utls "github.com/refraction-networking/utls"
)

const (
	workerStopSignal    = "0"
	workOnGoing         = 1
	controllerInterval  = 100               // in millisecond
	statisticIntervalT  = 1000              // in millisecond, valid in tcell mode
	statisticIntervalNT = 10000             // in millisecond, valid in non-tcell mode
	quitWaitingTime     = 3                 // in second
	downloadBufferSize  = 1024 * 64         // in byte
	fileDefaultSize     = 1024 * 1024 * 300 // in byte
	downloadSizeMin     = 1024 * 1024       // in byte
	defaultDLTUrl       = "https://cf.9999876.xyz/500mb.dat"
	defaultDTUrl        = "https://cf.9999876.xyz/test.dat"
	userAgentChrome     = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
	userAgentFirefox    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:124.0) Gecko/20100101 Firefox/124.0"
	userAgentEdge       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
	userAgentSafari     = "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3.1 Safari/605.1.15"

	defaultDBFile       = "ip.db"
	DefaultTestHost     = "cf.9999876.xyz"
	maxHostLen          = 1 << 12
	dtsSSL              = "SSL"
	dtsHTTPS            = "HTTPS"
	runTime             = "cftestor"
	retrieveCount   int = 32
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
	tlsClientID                             utls.ClientHelloID = utls.HelloChrome_Auto
	userAgent                               string             = userAgentChrome
	storeToFile, storeToDB, testAll, debug  bool
	resultFile, suffixLabel, dbFile         string
	myLogger                                MyLogger
	loggerLevel                             LogLevel
	httpRspTimeoutDuration                  time.Duration
	dtTimeoutDuration                       time.Duration
	dltTimeDurationMax                      time.Duration
	verifyResultsMap                        = make(map[*string]VerifyResults)
	// defaultASN                              = 0
	// defaultCity                             = ""
	myRand                            = rand.New(rand.NewSource(0))
	titleRuntime                      *string
	titlePre                          [2][4]string
	titleTasksStat                    [2]*string
	detailTitleSlice                  []string
	resultStrSlice, debugStrSlice     [][]*string
	termAll                           *tcell.Screen
	titleStyle                        = tcell.StyleDefault.Foreground(tcell.ColorBlack.TrueColor()).Background(tcell.ColorWhite)
	normalStyle                       = tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite)
	titleStyleCancel                  = tcell.StyleDefault.Foreground(tcell.ColorBlack.TrueColor()).Background(tcell.ColorGray)
	contentStyle                      = tcell.StyleDefault
	maxResultsDisplay                 = 10
	maxDebugDisplay                   = 10
	titleRuntimeRow                   = 0
	titlePreRow                       = titleRuntimeRow + 2
	titleCancelRow                    = titlePreRow + 3
	titleTasksStatRow                 = titleCancelRow + 2
	titleResultHintRow                = titleTasksStatRow + 2
	titleResultRow                    = titleResultHintRow + 1
	titleDebugHintRow                 = titleResultRow + maxResultsDisplay + 2
	titleDebugRow                     = titleDebugHintRow + 1
	titleCancel                       = "Press ESC to cancel!"
	titleCancelConfirm                = "Press ENTER to confirm; Any other key to back!"
	titleWaitQuit                     = "Waiting for exit..."
	titleResultHint                   = "Result:"
	titleDebugHint                    = "Debug Msg:"
	cancelSigFromTerm                 = false
	terminateConfirm                  = false
	resultStatIndent                  = 9
	dtThreadsNumLen, dltThreadsNumLen = 0, 0
	tcellMode                         = false
	fastMode                          = false
	silenceMode                       = false
	statInterval                      = statisticIntervalNT
	// titleExitHint                          = "Press any key to exit!"
	appArt string = `
  ░█▀▀░█▀▀░▀█▀░█▀▀░█▀▀░▀█▀░█▀█░█▀▄
  ░█░░░█▀▀░░█░░█▀▀░▀▀█░░█░░█░█░█▀▄
  ░▀▀▀░▀░░░░▀░░▀▀▀░▀▀▀░░▀░░▀▀▀░▀░▀
`
)

var help = `Usage: ` + runTime + ` [options]
options:
    -s, --ip           string  Specify IP, CIDR, or host for test. E.g.: "-s 1.0.0.1", "-s 1.0.0.1/32",
                               "-s 1.0.0.1/24", "-s 1.1.1.1:2053".
    -i, --in           string  Specify file for test, which contains multiple lines. Each line
                               represent one IP, CIDR, host.
    -p, --port         int     Port to test, could be specific one or more ports at same time. Can be
                               specific like "-p 443-800,1000:1300;8443|8444 -p 10000-12000|13333".
                               These ports should be working via SSL/TLS/HTTPS protocol,  default 443.
    -m, --dt-thread    int     Number of concurrent threads for Delay Test(DT). How many IPs can
                               be perform DT at the same time. Default 20 threads.
    -t, --dt-timeout   int     Timeout for single DT, unit ms, default 1000ms. A single SSL/TLS
                               or HTTPS request and response should be finished before timeout.
                               It should not be less than "-k|--evaluate-dt-delay", It should be
                               longer when we perform https connections test by "-dt-via-https"
                               than when we perform SSL/TLS test by default.
    -c, --dt-count     int     Tries of DT for a IP, default 2.
        --hostname     string  Hostname for DT test. It's valid when "--dt-only" is no and "--dt-via https"
                               is not provided.
        --dt-via https|tls|ssl DT via https or SSL/TLS shaking hands, "--dt-via <https|tls|ssl>"
                               default https.
        --dt-url       string  Specify test URL for DT.
        --ev-dt                Evaluate DT, we'll try "-c|--dt-count <value>" to evaluate delay;
                               if we don't turn this on, we'll stop DT after we got the first
                               successfull DT; if we turn this on, we'll evaluate the test result
                               through average delay of singe DT and statistic of all successfull
                               DT by these two thresholds "-k|--evaluate-dt-delay <value>" and
                               "-S|--evaluate-dt-dtpr <value>", default turn off.
    -k, --ev-dt-delay  int     single DT's delay should not bigger than this, unit ms, default 600ms.
    -S, --ev-dt-dtpr   float   The DT pass rate should not lower than this, default 100, means 100%, all
                               DT must be below "-k|--evaluate-dt-delay <value>".
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
        --fast                 Fast mode, use inner IPs for fast detection. Just when neither "-s/--ip"
                               nor "-i/--in" is provided, and this flag is provided. It will be working
                               Disabled by default.
    -4, --ipv4                 Just test IPv4. When we don't specify IPs to test by "-s" or "-i",
                               then it will do IPv4 test from build-in IPs from CloudFlare by default.
    -6, --ipv6                 Just test IPv6. When we don't specify IPs to test by "-s" or "-i",
                               then it will do IPv6 test from build-in IPs from CloudFlare by using
                               this flag.
        --hello-firefox        Work as firefox to perform tls/https
        --hello-chrome         Work as Chrome to perform tls/https
        --hello-edge           Work as Microsoft Edge to perform tls/https
        --hello-safari         Work as safari to perform tls/https
    -a  --test-all             Test all IPs until no more IP left. It's disabled by default.
    -w, --to-file              Write result to csv file, disabled by default. If it is provided and
                               "-o|--result-file <value>" is not provided, the result file will be named
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

type arrayFlags []string

func (i *arrayFlags) String() string {
	return fmt.Sprintf("%v", *i)
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *arrayFlags) Type() string {
	return "[]string"
}

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
	resultSlice []singleResult
}

type VerifyResults struct {
	testTime time.Time // test time
	ip       *string   // should be <ipv4:port> or <[ipv6]:port>, not just a ip string.
	dtc      int       // Delay Test(DT) tried count
	dtpc     int       // DT passed count
	dtpr     float64   // DT passed rate, in decimal
	da       float64   // average delay, in ms
	dmi      float64   // minimal delay, in ms
	dmx      float64   // max delay, in ms
	dltc     int       // Download Test(DLT) tried count
	dltpc    int       // DLT passed count
	dltpr    float64   // DLT passed rate, in decimal
	dls      float64   // DLT average speed, in KB/s
	dlds     int64     // DLT download data size, in byte
	dltd     float64   // DLT escaped times, in second
}

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
		// some error ocurred
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
