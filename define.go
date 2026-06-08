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
	defaultDLTUrl           = "https://speed.cloudflare.com/__down?bytes=250000000"
	defaultDTUrl            = "https://speed.cloudflare.com/__down?bytes=0"
	userAgentChrome         = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
	userAgentFirefox        = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:124.0) Gecko/20100101 Firefox/124.0"
	userAgentEdge           = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
	userAgentSafari         = "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3.1 Safari/605.1.15"

	defaultDBFile        = "ip.db"
	DefaultTestHost      = "speed.cloudflare.com"
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
	baseCfCDNCgiTraceUrl = "https://speed.cloudflare.com/cdn-cgi/trace"
)

var (
	maxHostLenBig                                  = big.NewInt(maxHostLen)
	version, buildTag, buildDate, buildHash string = "dev", "dev", "dev", "dev"
	ipStr                                   []string
	myLogger                                MyLogger
	loggerLevel                             LogLevel
	verifyResultsMap                               = make(map[string]VerifyResults)
	myRand                                         = newRand()
	srcIPs                                         = NewSourceIPsWithRand(myRand)
	appArt                                  string = `
  ░█▀▀░█▀▀░▀█▀░█▀▀░█▀▀░▀█▀░█▀█░█▀▄
  ░█░░░█▀▀░░█░░█▀▀░▀▀█░░█░░█░█░█▀▄
  ░▀▀▀░▀░░░░▀░░▀▀▀░▀▀▀░░▀░░▀▀▀░▀░▀
`
)

type AppConfig struct {
	IPFile                      string
	DTCount                     int
	DTWorkerThread              int
	DLTDurMax                   int
	DLTWorkerThread             int
	DLTCount                    int
	ResultMin                   int
	Interval                    int
	DTEvaluationDelay           int
	DTTimeout                   int
	DTStdExp                    float64
	HostName                    string
	DLTUrl                      string
	DTSource                    string
	DTUrl                       string
	DLTTimeout                  int
	Loop                        int
	TestTimeout                 int
	LoopInterval                int
	DTEvaluationDTPR            float64
	DLTEvaluationSpeed          float64
	DTHttps                     bool
	DisableDownload             bool
	DTVia                       string
	DTHttpRspReturnCodeExpected int
	EnableDTEvaluation          bool
	IPv4Mode                    bool
	IPv6Mode                    bool
	DTOnly                      bool
	DLTOnly                     bool
	TLSClientID                 utls.ClientHelloID
	UserAgent                   string
	StoreToFile                 bool
	StoreToDB                   bool
	TestAll                     bool
	Debug                       bool
	ResolveLocalASNAndCity      bool
	EnableStdEv                 bool
	ResultFile                  string
	SuffixLabel                 string
	DBFile                      string
	HttpRspTimeoutDuration      time.Duration
	DTTimeoutDuration           time.Duration
	DLTDurationInTotal          time.Duration
	PortStrSlice                []string
	FastMode                    bool
	SilenceMode                 bool
	ResolveLoc                  bool
	NoCache                     bool
	OutboundMark                uint32
	OutboundMarkSet             bool
	OutboundInterface           string
	OutboundInterfaceName       string
	OutboundInterfaceIndex      int
	OutboundSourceIP            net.IP
	OutboundSourceZone          string
}

var Config AppConfig = DefaultConfig()

var help = appArt + `
  Find and verify the best Cloudflare CDN edge nodes for your network.
  https://github.com/zhfreal/cftestor

Usage: cftestor [options]

Core Options:
    -s, --ip           strings    Specify IP, CIDR, or host:port. Examples: "-s 1.0.0.1", "-s 1.0.0.1/24",
                                  "-s 1.1.1.1:2053", "-s example.com:443". Can be provided multiple times.
    -i, --in           string     Path to a file containing IPs, CIDRs, or host:port entries (one per line).
    -p, --port         strings    Specify port(s) to test. Supports single ports, ranges, and lists (e.g.,
                                  "443", "80-443", "443,8443"). Default: 443.
    -a, --test-all                Test all provided IPs until none remain. Default: off.
    -r, --result       int        Target number of final qualified results. Default: 10.
        --fast                    Use a limited set of internal Cloudflare IPs for quick scanning.
    -4, --ipv4                    Test IPv4 only. Default: on (if no IPs specified).
    -6, --ipv6                    Test IPv6 only. Default: off. DNS hosts are resolved by the dialer.
    -C, --no-cache                Bypass CDN/Proxy caching for custom URLs (ignored for defaults).

Network Options:
        --mark        string      Set Linux socket fwmark for outbound packets. Supports decimal and hex.
        --xmark       string      Alias for --mark.
        --interface   string      Bind outbound packets to an interface name, interface index, or local source IP.

Delay Test (DT) Options:
    -m, --dt-thread    int        Number of concurrent DT workers. Default: 20.
    -t, --dt-timeout   int        Timeout for a single DT attempt in ms. Default: 2000 (TLS/SSL) or 5000 (HTTPS).
    -c, --dt-count     int        Number of DT attempts per candidate. Default: 4.
        --dt-via       string     DT protocol: "https", "tls", or "ssl". Default: https.
        --dt-via-https            Deprecated alias for --dt-via https.
        --dt-url       string     URL to use for HTTPS-based DT. Default: ` + defaultDTUrl + `
        --hostname     string     SNI hostname for TLS/SSL DT. Default: ` + DefaultTestHost + `
        --dt-expect-code int      Expected HTTP status code for DT. Default: 200.
        --ev-dt                   Enable DT evaluation using all attempts. Default: off.
    -k, --ev-dt-delay  int        Maximum allowed average DT delay in ms. Default: 600.
        --ev-dt-dtpr   float      Minimum required DT pass rate (percentage). Default: 100.0.
        --ev-dt-std    float      Maximum allowed DT standard deviation. Default: 30.0 (if enabled).

Download Test (DLT) Options:
    -n, --dlt-thread   int        Number of concurrent DLT workers. Default: 1.
    -d, --dlt-period   int        Maximum duration for one DLT attempt in seconds. Default: 10.
    -b, --dlt-count    int        Number of DLT attempts per candidate. Default: 1.
    -u, --dlt-url      string     URL to use for DLT. Default: ` + defaultDLTUrl + `
        --dlt-timeout  int        HTTP response timeout for DLT in ms. Default: 5000.
    -l, --speed        float      Minimum required download speed in KB/s. Default: 6000.
    -I, --interval     int        Interval between test attempts in ms. Default: 500.

Mode Options:
        --dt-only                 Perform Delay Test only.
        --disable-download        Deprecated alias for --dt-only.
        --dlt-only                Perform Download Test only.
        --loop         int        Retest qualified candidates for N confirmation cycles; refill from the original pool
                                  if fewer than --result remain.
        --loop-interval int       Seconds to wait between loop cycles. Default: 60.
        --test-timeout int        Total test timeout in minutes. Default: 30.

Fingerprinting Options:
        --hello-firefox           Simulate Firefox TLS fingerprint.
        --hello-chrome            Simulate Chrome TLS fingerprint (Default).
        --hello-edge              Simulate Edge TLS fingerprint.
        --hello-safari            Simulate Safari TLS fingerprint.

Output & Storage Options:
    -w, --to-file                 Save results to a CSV file.
    -o, --out-file     string     Path for the output CSV file.
    -e, --to-db                   Save results to a SQLite3 database.
    -f, --db-file      string     Path for the SQLite3 database file.
    -g, --label        string     Label for records (defaults to hostname).
        --resolve-loc             Attempt to resolve and display Cloudflare location.
        --local-asn               Retrieve and store local ASN/city info.

Alias Options:
        --source                  Alias for --ip.
        --source-file             Alias for --in.
        --result-count            Alias for --result.
        --dt-workers              Alias for --dt-thread.
        --dt-timeout-ms           Alias for --dt-timeout.
        --dt-attempts             Alias for --dt-count.
        --dt-protocol             Alias for --dt-via.
        --sni-hostname            Alias for --hostname.
        --dt-status-code          Alias for --dt-expect-code.
        --dt-evaluate             Alias for --ev-dt.
        --dt-max-delay            Alias for --ev-dt-delay.
        --dt-min-pass-rate        Alias for --ev-dt-dtpr.
        --dt-max-stddev           Alias for --ev-dt-std.
        --dlt-workers             Alias for --dlt-thread.
        --dlt-duration            Alias for --dlt-period.
        --dlt-attempts            Alias for --dlt-count.
        --dlt-timeout-ms          Alias for --dlt-timeout.
        --test-interval-ms        Alias for --interval.
        --min-speed               Alias for --speed.
        --to-csv                  Alias for --to-file.
        --csv-file                Alias for --out-file.
        --to-sqlite               Alias for --to-db.
        --sqlite-file             Alias for --db-file.
        --record-label            Alias for --label.
        --resolve-location        Alias for --resolve-loc.
        --quiet                   Alias for --silence.

General Options:
    -S, --silence                 Enable silence mode with minimal output.
    -V, --debug                   Print detailed debug logs.
    -v, --version                 Show version information and exit.
    -h, --help                    Show this help message.
`

// type arrayFlags []string

// func (i *arrayFlags) String() string {
//     return fmt.Sprintf("%v", *i)
// }

// func (i *arrayFlags) Set(value string) error {
//     *i = append(*i, value)
//     return nil
// }

// func (i *arrayFlags) Type() string {
//     return "[]string"
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
//     tIp := *a.ip
//     tLoc := *a.loc
//     tDtDList := make([]float64, len(a.dtDList))
//     copy(tDtDList, a.dtDList)
//     return VerifyResults{
//         testTime: a.testTime,
//         ip:       &tIp,
//         loc:      &tLoc,
//         dtc:      a.dtc,
//         dtpc:     a.dtpc,
//         dtpr:     a.dtpr,
//         da:       a.da,
//         daVar:    a.daVar,
//         daStd:    a.daStd,
//         dmi:      a.dmi,
//         dmx:      a.dmx,
//         dltc:     a.dltc,
//         dltpc:    a.dltpc,
//         dltpr:    a.dltpr,
//         dls:      a.dls,
//         dlds:     a.dlds,
//         dltd:     a.dltd,
//         dtDList:  tDtDList,
//     }

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
	if !ipr.isValid() || num <= 0 || ipr.Extracted || ipr.Len.Cmp(big.NewInt(0)) == 0 {
		return
	}
	numBig := big.NewInt(int64(num))
	if ipr.Len.Cmp(numBig) == -1 {
		num = int(ipr.Len.Int64())
		numBig = big.NewInt(int64(num))
	}

	for i := 0; i < num; i++ {
		n := big.NewInt(int64(i))
		num_in_bytes := fillBytes(n.Bytes(), len(ipr.IPStart))
		newIP := ipShift(ipr.IPStart, num_in_bytes)
		if newIP != nil {
			IPList = append(IPList, newIP)
		}
	}

	// reset IPStart and Extracted
	if numBig.Cmp(ipr.Len) == 0 {
		ipr.Extracted = true
		ipr.Len = big.NewInt(0)
		ipr.IPStart = ipr.IPEnd
	} else {
		num_in_bytes := fillBytes(numBig.Bytes(), len(ipr.IPStart))
		ipr.IPStart = ipShift(ipr.IPStart, num_in_bytes)
		ipr.Len = ipr.length()
	}
	return
}

func (ipr *ipRange) ExtractReverse(num int) (IPList []net.IP) {
	if !ipr.isValid() || num <= 0 || ipr.Extracted || ipr.Len.Cmp(big.NewInt(0)) == 0 {
		return
	}
	numBig := big.NewInt(int64(num))
	if ipr.Len.Cmp(numBig) == -1 {
		num = int(ipr.Len.Int64())
		numBig = big.NewInt(int64(num))
	}

	for i := 0; i < num; i++ {
		n := big.NewInt(int64(i))
		num_in_bytes := fillBytes(n.Bytes(), len(ipr.IPEnd))
		newIP := ipShiftReverse(ipr.IPEnd, num_in_bytes)
		if newIP != nil {
			IPList = append(IPList, newIP)
		}
	}

	// reset IPEnd and Extracted
	if numBig.Cmp(ipr.Len) == 0 {
		ipr.Extracted = true
		ipr.Len = big.NewInt(0)
		ipr.IPEnd = ipr.IPStart
	} else {
		num_in_bytes := fillBytes(numBig.Bytes(), len(ipr.IPEnd))
		ipr.IPEnd = ipShiftReverse(ipr.IPEnd, num_in_bytes)
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
		IPList = append(IPList, m...)
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

func (s *SafeLooper) SetInterval(interv int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interval = interv
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

func (s *SafeLooper) SleepInterval(interv int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	time.Sleep(time.Duration(interv) * time.Millisecond)
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

func NewSafeLooperWithInterval(t, interv int) *SafeLooper {
	s := NewSafeLooper(t)
	s.SetInterval(interv)
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

func (s *sourceIPs) IsEmpty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.srcIPRsRaw) == 0 && len(s.srcIPRsExtracted) == 0 && len(s.srcHosts) == 0
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
		// DNS hosts are family-agnostic until resolved by the dialer, so keep them
		// for either IPv4 or IPv6 scans. IP literal hosts still obey the mode filter.
		isDNSHost := tV == (TypeIPv4 | TypeIPv6)
		if !isDNSHost && (tV&mode) != tV {
			return nil
		}
		s.srcHosts = append(s.srcHosts, &ips)
	} else {
		return fmt.Errorf("the input %q is not a valid IP, CIDR, or host:port", ips)
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
		return fmt.Errorf("file %q is not accessible: %w", filename, err)
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

func (s *sourceIPs) AddPorts(srcPorts []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	portRegex := regexp.MustCompile(`[,;|]+`)
	portRangeRegex := regexp.MustCompile(`^\d+[-:]\d+$`)
	portRangeSplitRegex := regexp.MustCompile(`[-:]`)
	if len(srcPorts) > 0 {
		for _, portStr := range srcPorts {
			portStrSlice := portRegex.Split(portStr, -1)
			for _, portValue := range portStrSlice {
				portValue = strings.TrimSpace(portValue)
				if len(portValue) == 0 {
					continue
				}
				if portRangeRegex.MatchString(portValue) {
					portList := portRangeSplitRegex.Split(portValue, -1)
					if len(portList) != 2 {
						return invalidPortFlagError(portValue)
					}
					startPort, err := strconv.Atoi(portList[0])
					if err != nil {
						return invalidPortFlagError(portList[0])
					}
					endPort, err := strconv.Atoi(portList[1])
					if err != nil {
						return invalidPortFlagError(portList[1])
					}
					if startPort > endPort || startPort < 1 || endPort > 65535 {
						return invalidPortFlagError(portValue)
					}
					for i := startPort; i <= endPort; i++ {
						s.ports = append(s.ports, i)
					}
				} else {
					port, err := strconv.Atoi(portValue)
					if err != nil || port < 1 || port > 65535 {
						return invalidPortFlagError(portValue)
					}
					s.ports = append(s.ports, port)
				}
			}
		}
	}
	if len(s.ports) == 0 {
		s.ports = append(s.ports, DefaultPort)
	}
	s.ports = uniqueIntSlice(s.ports)
	return nil
}

func invalidPortFlagError(value string) error {
	return fmt.Errorf("invalid value for %q: %q", "-p|--port", value)
}

func (s *sourceIPs) RetrieveSome(amount int, isRand bool) (targetIPs []*string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// To satisfy the user's need for diversity across ALL sources,
	// we sample from both hosts and CIDRs.
	numHosts := len(s.srcHosts)
	if numHosts > 0 {
		// Take a portion for hosts, but leave room for CIDRs
		takeHosts := min(amount/2, numHosts)
		if takeHosts == 0 {
			takeHosts = 1
		}
		targetIPs = append(targetIPs, s.srcHosts[:takeHosts]...)
		s.srcHosts = s.srcHosts[takeHosts:]
	}

	left := amount - len(targetIPs)
	if left > 0 {
		t_target := s.retrieveIPsFromIPR(left, isRand)
		for _, ipStr := range t_target {
			for _, port := range s.ports {
				host := genHostFromIPStrPort(*ipStr, port)
				if len(host) > 0 {
					targetIPs = append(targetIPs, &host)
				}
			}
		}
	}
	return
}

func (s *sourceIPs) RetrieveSomeNew(amount int) (targetIPs []*string) {
	return s.RetrieveSome(amount, false)
}

func (s *sourceIPs) retrieveHosts(amount int) (targetHosts []*string) {
	if amount <= 0 || len(s.srcHosts) == 0 {
		return
	}
	t_amount := min(amount, len(s.srcHosts))
	targetHosts = append(targetHosts, s.srcHosts[:t_amount]...)
	s.srcHosts = s.srcHosts[t_amount:]
	return
}

func (s *sourceIPs) retrieveIPsFromIPR(amount int, isRandom bool) (targetIPs []*string) {
	if amount <= 0 {
		return
	}

	numRaw := len(s.srcIPRsRaw)
	hasExtracted := len(s.srcIPRsExtracted) > 0
	totalGroups := numRaw
	if hasExtracted {
		totalGroups++
	}
	if totalGroups == 0 {
		return nil
	}

	// Calculate a fair share per group to ensure diversity
	perGroup := amount / totalGroups
	if perGroup == 0 {
		perGroup = 1
	}

	// Shuffle indices to ensure fairness across multiple calls
	indices := make([]int, totalGroups)
	for i := range indices {
		indices[i] = i
	}
	myRand.Shuffle(len(indices), func(i, j int) {
		indices[i], indices[j] = indices[j], indices[i]
	})

	t_ips := make([]net.IP, 0, amount)
	for i, idx := range indices {
		if len(t_ips) >= amount {
			break
		}

		need := amount - len(t_ips)
		take := perGroup
		// On the last group, take whatever is left in the budget
		if i == len(indices)-1 {
			take = need
		}
		if take > need {
			take = need
		}

		if hasExtracted && idx == numRaw {
			// Pre-extracted pool (small CIDRs)
			actualTake := min(take, len(s.srcIPRsExtracted))
			t_ips = append(t_ips, s.srcIPRsExtracted[:actualTake]...)
			s.srcIPRsExtracted = s.srcIPRsExtracted[actualTake:]
		} else {
			// Raw CIDR range (large CIDRs)
			ipr := s.srcIPRsRaw[idx]
			var extracted []net.IP
			if isRandom {
				extracted = ipr.GetRandomX(take)
			} else {
				extracted = ipr.Extract(take)
			}
			t_ips = append(t_ips, extracted...)
		}
	}

	// Cleanup empty ranges
	for i := 0; i < len(s.srcIPRsRaw); i++ {
		if s.srcIPRsRaw[i].Len.Cmp(big.NewInt(0)) == 0 {
			s.srcIPRsRaw = append(s.srcIPRsRaw[:i], s.srcIPRsRaw[i+1:]...)
			i--
		}
	}

	for _, t_ip := range t_ips {
		tIP := t_ip.String()
		targetIPs = append(targetIPs, &tIP)
	}

	// Shuffle the final batch for extra randomization
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
