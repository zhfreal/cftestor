package config

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

	"cftestor/internal/utils"
	utls "github.com/metacubex/utls"
)

const (
	WorkerStopSignal        = "0"
	WorkOnGoing         int = 1
	ControllerInterval      = 100               // in millisecond
	StatisticIntervalT      = 1000              // in millisecond, valid in tcell mode
	StatisticIntervalNT     = 10000             // in millisecond, valid in non-tcell mode
	QuitWaitingTime         = 3                 // in second
	DownloadBufferSize      = 1024 * 64         // in byte
	FileDefaultSize         = 1024 * 1024 * 300 // in byte
	DownloadSizeMin         = 1024 * 1024       // in byte
	DefaultDLTUrl           = "https://speed.cloudflare.com/__down?bytes=99999999"
	DefaultDTUrl            = "https://speed.cloudflare.com/__down?bytes=0"
	UserAgentChrome         = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
	UserAgentFirefox        = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:124.0) Gecko/20100101 Firefox/124.0"
	UserAgentEdge           = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
	UserAgentSafari         = "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_4) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3.1 Safari/605.1.15"

	DefaultDBFile        = "ip.db"
	DefaultTestHost      = "speed.cloudflare.com"
	MaxHostLen           = 1 << 12
	DtsSSL               = "SSL"
	DtsHTTPS             = "HTTPS"
	RunTime              = "cftestor"
	RetrieveCount   int  = 32
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
	UTF8BomBytes    = []byte{0xEF, 0xBB, 0xBF}
	ResultCsvHeader = []string{
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
	BaseCfCDNCgiTraceUrl = "https://speed.cloudflare.com/cdn-cgi/trace"
	SourceLevelUser      = 0
	SourceLevelFast      = 1
	SourceLevelFull      = 2
)

func GetSourceLevelName(level int) string {
	switch level {
	case SourceLevelUser:
		return "user source (-s/-i)"
	case SourceLevelFast:
		return "fast ranges (--fast)"
	case SourceLevelFull:
		return "full ranges (Cloudflare CIDRs)"
	default:
		return fmt.Sprintf("level %d", level)
	}
}

var (
	MaxHostLenBig                            = big.NewInt(MaxHostLen)
	Version, BuildTag, BuildDate, BuildHash string = "dev", "dev", "dev", "dev"
	IPStr                                   []string
	VerifyResultsMap                               = make(map[string]VerifyResults)
	MyRand                                         = utils.NewRand()
	SrcIPs                                         = NewSourceIPsWithRand(MyRand)
	AppArt                                  string = `
  ░█▀▀░█▀▀░▀█▀░█▀▀░█▀▀░▀█▀░█▀█░█▀▄
  ░█░░░█▀▀░░█░░█▀▀░▀▀█░░█░░█░█░█▀▄
  ░▀▀▀░▀░░░░▀░░▀▀▀░▀▀▀░░▀░░▀▀▀░▀░▀
`
)

type AppConfig struct {
	IPFile                      string
	IPFiles                     []string
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
	IntervalDuration            time.Duration
	DTEvaluationDelayDuration   time.Duration
	LoopIntervalDuration        time.Duration
	TestTimeoutDuration         time.Duration
	lastInterval                  int
	lastIntervalDuration          time.Duration
	lastLoopInterval              int
	lastLoopIntervalDuration      time.Duration
	lastTestTimeout               int
	lastTestTimeoutDuration       time.Duration
	lastDTTimeout                 int
	lastDTTimeoutDuration         time.Duration
	lastDTEvaluationDelay         int
	lastDTEvaluationDelayDuration time.Duration
	lastDLTDurMax                 int
	lastDLTDurationInTotal        time.Duration
	lastDLTTimeout                int
	lastHttpRspTimeoutDuration    time.Duration
	PortStrSlice                []string
	FastMode                    bool
	SilenceMode                 bool
	ResolveLoc                  bool
	ResolveLocPar               int
	NoCache                     bool
	Supplement                  bool
	SupplementIPv4              bool
	SupplementIPv6              bool
	OutboundMark                uint32
	OutboundMarkSet             bool
	OutboundInterface           string
	OutboundInterfaceName       string
	OutboundInterfaceIndex      int
	OutboundSourceIP            net.IP
	OutboundSourceZone          string
	FetchIPv6File               string
	FetchIPv4File               string
	FetchCFDomainsFile          string
	DNSServer                   string
	TrancoLimit                 int
}

var Config AppConfig = DefaultConfig()

var Help = AppArt + `
  Find and verify the best Cloudflare CDN edge nodes for your network.
  https://github.com/zhfreal/cftestor

Usage: cftestor [options]

Core Options:
    -s, --ip           strings    Specify IP, CIDR, or host:port. Examples: "-s 1.0.0.1", "-s 1.0.0.1/24",
                                  "-s 1.1.1.1:2053", "-s example.com:443". Can be provided multiple times.
    -i, --in           strings    Path to file(s) containing IPs, CIDRs, or host:port entries. Can be provided multiple times.
    -p, --port         strings    Specify port(s) to test. Supports single ports, ranges, and lists (e.g.,
                                  "443", "80-443", "443,8443"). Default: 443.
    -a, --test-all                Test all provided IPs until none remain. Default: off.
    -r, --result       int        Target number of final qualified results. Default: 10.
        --fast                    Use a limited set of internal Cloudflare IPs for quick scanning.
    -4, --ipv4                    Test IPv4 only. Default: on (if no IPs specified).
    -6, --ipv6                    Test IPv6 only. Default: off. DNS hosts are resolved by the dialer.
        --fetch-ipv4   string     Fetch active Cloudflare IPv4 CIDRs dynamically, save to file, and exit.
        --fetch-ipv6   string     Fetch active Cloudflare IPv6 CIDRs dynamically, save to file, and exit.
        --fetch-cf-domains string Fetch, verify, and save top domains using Cloudflare CDN to a file, and exit.
        --dns          string     Custom DNS server for dynamic fetching.
    -C, --no-cache                Bypass CDN/Proxy caching for custom URLs (ignored for defaults).

Network Options:
        --mark        string      Set Linux socket fwmark for outbound packets. Supports decimal and hex.
        --xmark       string      Alias for --mark.
        --interface   string      Bind outbound packets to an interface name, interface index, or local source IP.

Delay Test (DT) Options:
    -m, --dt-thread    int        Number of concurrent DT workers. Default: 20.
    -t, --dt-timeout   duration   Timeout for a single DT attempt (e.g. 50ms, 2s; default: 2000ms TLS/SSL, 5000ms HTTPS).
    -c, --dt-count     int        Number of DT attempts per candidate. Default: 4.
        --dt-via       string     DT protocol: "https", "tls", or "ssl". Default: https.
        --dt-via-https            Deprecated alias for --dt-via https.
        --dt-url       string     URL to use for HTTPS-based DT. Default: ` + DefaultDTUrl + `
        --hostname     string     SNI hostname for TLS/SSL DT. Default: ` + DefaultTestHost + `
        --dt-expect-code int      Expected HTTP status code for DT. Default: 200.
        --ev-dt                   Enable DT evaluation using all attempts. Default: off.
    -k, --ev-dt-delay  duration   Maximum allowed average DT delay (e.g. 50ms, 600ms; default: 600ms).
        --ev-dt-dtpr   float      Minimum required DT pass rate (percentage). Default: 100.0.
        --ev-dt-std    float      Maximum allowed DT standard deviation. Default: 30.0 (if enabled).

Download Test (DLT) Options:
    -n, --dlt-thread   int        Number of concurrent DLT workers. Default: 1.
    -d, --dlt-period   duration   Maximum duration for one DLT attempt (e.g. 50s, 10s, 10000ms; default: 10s).
    -b, --dlt-count    int        Number of DLT attempts per candidate. Default: 1.
    -u, --dlt-url      string     URL to use for DLT. Default: ` + DefaultDLTUrl + `
        --dlt-timeout  duration   HTTP response timeout for DLT (e.g. 50ms, 5s; default: 5000ms).
    -l, --speed        float      Minimum required download speed in KB/s. Default: 6000.
    -I, --interval     duration   Interval between test attempts (e.g. 50ms, 500ms, 1s; default: 500ms).

Mode Options:
        --dt-only                 Perform Delay Test only.
        --disable-download        Deprecated alias for --dt-only.
        --dlt-only                Perform Download Test only.
        --loop         int        Retest qualified candidates for N confirmation cycles; refill from the original pool
                                  if fewer than --result remain.
        --loop-interval duration  Time to wait between loop cycles (e.g. 50s, 60s, 1m; default: 60s).
        --test-timeout duration   Total test timeout (e.g. 50mins, 50s, 30m; default: 30m).
        --supplement              Enable multi-tier IP source supplementation when target result count is not met:
                                  user sources (-s/-i) -> fast ranges (--fast) -> full Cloudflare CIDRs.
                                  If default full scan is used, tests full Cloudflare CIDRs only.
        --supplement-ipv4         Enable IPv4 for supplementation fallback. Default: true.
        --supplement-ipv6         Enable IPv6 for supplementation fallback. Default: false.

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
        --resolve-loc-par  int    Number of concurrent threads to resolve locations in parallel. Default: 10.
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
        --resolve-loc-threads     Alias for --resolve-loc-par.
        --quiet                   Alias for --silence.

General Options:
    -S, --silence                 Enable silence mode with minimal output.
    -V, --debug                   Print detailed debug logs.
    -v, --version                 Show version information and exit.
    -h, --help                    Show this help message.
`

type SingleResult struct {
	DTPassed      bool
	DTDuration    time.Duration
	HttpReqRspDur time.Duration
	DLTWasDone    bool
	DLTPassed     bool
	DLTDuration   time.Duration
	DLTDataSize   int64
}

type SingleVerifyResult struct {
	TestTime    time.Time
	Host        string
	Loc         string
	ResultSlice []SingleResult
}

type VerifyResults struct {
	TestTime time.Time
	IP       *string
	Loc      *string
	Dtc      int
	Dtpc     int
	Dtpr     float64
	Da       float64
	DaVar    float64
	DaStd    float64
	Dmi      float64
	Dmx      float64
	Dltc     int
	Dltpc    int
	Dltpr    float64
	Dls      float64
	Dlds     int64
	Dltd     float64
	DtDList  []float64
}

func (a *VerifyResults) Combine(b VerifyResults) {
	if a.IP == nil || b.IP == nil || *a.IP != *b.IP {
		return
	}
	if a.TestTime.IsZero() || (!b.TestTime.IsZero() && a.TestTime.Before(b.TestTime)) {
		a.TestTime = b.TestTime
	}
	if b.Loc != nil && len(*b.Loc) != 0 && (a.Loc == nil || len(*a.Loc) == 0) {
		a.Loc = b.Loc
	}
	a.Dtc += b.Dtc
	a.Dtpc += b.Dtpc
	if a.Dtc > 0 {
		a.Dtpr = float64(a.Dtpc) / float64(a.Dtc)
	}
	a.DtDList = append(a.DtDList, b.DtDList...)
	n := 0
	for _, v := range a.DtDList {
		if v > 0 {
			a.DtDList[n] = v
			n++
		}
	}
	a.DtDList = a.DtDList[:n]

	totalDelay := 0.0
	for _, v := range a.DtDList {
		totalDelay += v
	}
	if a.Dtpc > 0 && len(a.DtDList) > 0 {
		a.Da = totalDelay / float64(len(a.DtDList))
	}
	a.DaStd = utils.Std(a.DtDList)
	a.DaVar = utils.Variance(a.DtDList)
	if b.Dtpc > 0 {
		if a.Dmi <= 0.0 || (b.Dmi > 0.0 && b.Dmi < a.Dmi) {
			a.Dmi = b.Dmi
		}
		if a.Dmx <= 0.0 || (b.Dmx > 0.0 && b.Dmx > a.Dmx) {
			a.Dmx = b.Dmx
		}
	}
	a.Dltc += b.Dltc
	a.Dltpc += b.Dltpc
	if a.Dltc > 0 {
		a.Dltpr = float64(a.Dltpc) / float64(a.Dltc)
	}
	a.Dlds += b.Dlds
	a.Dltd += b.Dltd
	if a.Dltpc > 0 && a.Dltd > 0 {
		a.Dls = float64(a.Dlds) / float64(a.Dltd) / 1000
	}
}

type ResultSpeedSorter []VerifyResults

func (a ResultSpeedSorter) Len() int           { return len(a) }
func (a ResultSpeedSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ResultSpeedSorter) Less(i, j int) bool { return a[i].Dls < a[j].Dls }

type OverAllStat struct {
	DtTasksDone  int
	DtOnGoing    int
	DtCached     int
	DltTasksDone int
	DltOnGoing   int
	DltCached    int
	ResultCount  int
	Remain       int
}

type SafeLooper struct {
	mu       sync.Mutex
	t, c     int
	interval int
	duration time.Duration
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
	s.duration = time.Duration(interv) * time.Millisecond
}

func (s *SafeLooper) SetDuration(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.duration = d
	s.interval = int(d / time.Millisecond)
	if s.interval == 0 && d > 0 {
		s.interval = 1
	}
}

func (s *SafeLooper) GetInterval() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.interval
}

func (s *SafeLooper) GetDuration() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.duration > 0 {
		return s.duration
	}
	return time.Duration(s.interval) * time.Millisecond
}

func (s *SafeLooper) GetRound() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.c
}

func (s *SafeLooper) Sleep() {
	s.SleepWithCheck(nil)
}

func (s *SafeLooper) SleepWithCheck(shouldStop func() bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	dur := s.duration
	if dur <= 0 && s.interval > 0 {
		dur = time.Duration(s.interval) * time.Millisecond
	}
	s.mu.Unlock()
	if dur <= 0 {
		return
	}
	if shouldStop == nil {
		time.Sleep(dur)
		return
	}
	deadline := time.Now().Add(dur)
	for time.Now().Before(deadline) {
		if shouldStop() {
			return
		}
		chunk := time.Until(deadline)
		if chunk > 100*time.Millisecond {
			chunk = 100 * time.Millisecond
		}
		time.Sleep(chunk)
	}
}

func (s *SafeLooper) SleepInterval(interv int) {
	if interv > 0 {
		time.Sleep(time.Duration(interv) * time.Millisecond)
	}
}

func (s *SafeLooper) SleepDuration(d time.Duration) {
	if d > 0 {
		time.Sleep(d)
	}
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
		duration: 1000 * time.Millisecond,
	}
}

func NewSafeLooperWithInterval(t, interv int) *SafeLooper {
	s := NewSafeLooper(t)
	s.SetInterval(interv)
	return s
}

func NewSafeLooperWithDuration(t int, d time.Duration) *SafeLooper {
	s := NewSafeLooper(t)
	s.SetDuration(d)
	return s
}

type SourceIPs struct {
	mu                  sync.Mutex
	srcHosts            []*string
	srcIPRsRawV4        []*utils.IPRange
	srcIPRsRawV6        []*utils.IPRange
	srcIPRsExtractedV4  []net.IP
	srcIPRsExtractedV6  []net.IP
	Ports               []int
	tRnd                *rand.Rand
}

func (s *SourceIPs) TotalHosts() *big.Int {
	s.mu.Lock()
	defer s.mu.Unlock()
	t_qty := big.NewInt(0)
	for i := 0; i < len(s.srcIPRsRawV4); i++ {
		t_qty = t_qty.Add(t_qty, s.srcIPRsRawV4[i].Length())
	}
	for i := 0; i < len(s.srcIPRsRawV6); i++ {
		t_qty = t_qty.Add(t_qty, s.srcIPRsRawV6[i].Length())
	}
	t_qty = t_qty.Add(t_qty, big.NewInt(int64(len(s.srcIPRsExtractedV4)+len(s.srcIPRsExtractedV6))))
	portCount := int64(len(s.Ports))
	if portCount > 1 {
		t_qty = t_qty.Mul(t_qty, big.NewInt(portCount))
	}
	t_qty = t_qty.Add(t_qty, big.NewInt(int64(len(s.srcHosts))))
	return t_qty
}

func (s *SourceIPs) Len() *big.Int {
	return s.TotalHosts()
}

func (s *SourceIPs) LenInt() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	t_qty := 0
	t_qty += len(s.srcIPRsRawV4)
	t_qty += len(s.srcIPRsRawV6)
	t_qty += len(s.srcIPRsExtractedV4)
	t_qty += len(s.srcIPRsExtractedV6)
	t_qty += len(s.srcHosts)
	return t_qty
}

func (s *SourceIPs) IsEmpty() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.srcIPRsRawV4) == 0 && len(s.srcIPRsRawV6) == 0 &&
		len(s.srcIPRsExtractedV4) == 0 && len(s.srcIPRsExtractedV6) == 0 &&
		len(s.srcHosts) == 0
}

func (s *SourceIPs) add(IPs string, mode int8) error {
	ips := strings.TrimSpace(IPs)
	ips, _, _ = strings.Cut(ips, "#")
	if utils.IsValidIPs(ips) {
		tV := utils.GetIPsVer(ips)
		if tV == TypeIPErr {
			return fmt.Errorf("\"%v\" is invalid", ips)
		}
		if (tV & mode) != tV {
			return nil
		}
		ipr := utils.NewIPRangeFromCIDR(&ips)
		if ipr == nil {
			return fmt.Errorf("\"%v\" is invalid", ips)
		}
		if ipr.Len.Cmp(MaxHostLenBig) < 1 {
			extracted := ipr.ExtractAll(MaxHostLen)
			if ipr.IsV6() {
				s.srcIPRsExtractedV6 = append(s.srcIPRsExtractedV6, extracted...)
			} else {
				s.srcIPRsExtractedV4 = append(s.srcIPRsExtractedV4, extracted...)
			}
		} else {
			if ipr.IsV6() {
				s.srcIPRsRawV6 = append(s.srcIPRsRawV6, ipr)
			} else {
				s.srcIPRsRawV4 = append(s.srcIPRsRawV4, ipr)
			}
		}
	} else if utils.IsValidHost(ips) {
		tV := utils.GetHostVer(ips)
		if tV == TypeIPErr {
			return fmt.Errorf("\"%v\" is invalid", ips)
		}
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

func (s *SourceIPs) Add(IPs string, mode int8) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.add(IPs, mode)
}

func (s *SourceIPs) AddFromSlice(ipsSlice []string, mode int8) error {
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

func (s *SourceIPs) AddFromFile(filename string, mode int8) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tFile, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("file %q is not accessible: %w", filename, err)
	}
	defer tFile.Close()
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
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file %q: %w", filename, err)
	}
	return nil
}

func (s *SourceIPs) AddPorts(srcPorts []string) error {
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
						s.Ports = append(s.Ports, i)
					}
				} else {
					port, err := strconv.Atoi(portValue)
					if err != nil || port < 1 || port > 65535 {
						return invalidPortFlagError(portValue)
					}
					s.Ports = append(s.Ports, port)
				}
			}
		}
	}
	if len(s.Ports) == 0 {
		s.Ports = append(s.Ports, DefaultPort)
	}
	s.Ports = utils.UniqueIntSlice(s.Ports)
	return nil
}

func invalidPortFlagError(value string) error {
	return fmt.Errorf("invalid value for %q: %q", "-p|--port", value)
}

func (s *SourceIPs) RetrieveSome(amount int, isRand bool) (targetIPs []*string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	numHosts := len(s.srcHosts)
	if numHosts > 0 {
		takeHosts := utils.MinInt(amount, numHosts)
		targetIPs = append(targetIPs, s.srcHosts[:takeHosts]...)
		for i := 0; i < takeHosts; i++ {
			s.srcHosts[i] = nil
		}
		s.srcHosts = s.srcHosts[takeHosts:]
		if len(s.srcHosts) == 0 {
			s.srcHosts = nil
		} else if cap(s.srcHosts) > 1024 && len(s.srcHosts) < cap(s.srcHosts)/4 {
			s.srcHosts = append([]*string(nil), s.srcHosts...)
		}
	}

	left := amount - len(targetIPs)
	if left > 0 {
		t_target := s.retrieveIPsFromIPR(left, isRand)
		for _, ipStr := range t_target {
			for _, port := range s.Ports {
				host := utils.GenHostFromIPStrPort(*ipStr, port)
				if len(host) > 0 {
					targetIPs = append(targetIPs, &host)
				}
			}
		}
	}
	return
}

func (s *SourceIPs) RetrieveSomeNew(amount int) (targetIPs []*string) {
	return s.RetrieveSome(amount, false)
}

func (s *SourceIPs) retrieveHosts(amount int) (targetHosts []*string) {
	if amount <= 0 || len(s.srcHosts) == 0 {
		return
	}
	t_amount := utils.MinInt(amount, len(s.srcHosts))
	targetHosts = append(targetHosts, s.srcHosts[:t_amount]...)
	for i := 0; i < t_amount; i++ {
		s.srcHosts[i] = nil
	}
	s.srcHosts = s.srcHosts[t_amount:]
	if len(s.srcHosts) == 0 {
		s.srcHosts = nil
	} else if cap(s.srcHosts) > 1024 && len(s.srcHosts) < cap(s.srcHosts)/4 {
		s.srcHosts = append([]*string(nil), s.srcHosts...)
	}
	return
}

func (s *SourceIPs) hasIP4() bool {
	return len(s.srcIPRsExtractedV4) > 0 || len(s.srcIPRsRawV4) > 0
}

func (s *SourceIPs) hasIP6() bool {
	return len(s.srcIPRsExtractedV6) > 0 || len(s.srcIPRsRawV6) > 0
}

func (s *SourceIPs) retrieveOneIP(isV6 bool, isRandom bool) net.IP {
	rnd := s.tRnd
	if rnd == nil {
		rnd = MyRand
	}
	if isV6 {
		if n := len(s.srcIPRsExtractedV6); n > 0 {
			if isRandom {
				idx := rnd.Intn(n)
				ip := s.srcIPRsExtractedV6[idx]
				s.srcIPRsExtractedV6[idx] = s.srcIPRsExtractedV6[n-1]
				s.srcIPRsExtractedV6[n-1] = nil
				s.srcIPRsExtractedV6 = s.srcIPRsExtractedV6[:n-1]
				if len(s.srcIPRsExtractedV6) == 0 {
					s.srcIPRsExtractedV6 = nil
				}
				return ip
			}
			ip := s.srcIPRsExtractedV6[0]
			s.srcIPRsExtractedV6[0] = nil
			s.srcIPRsExtractedV6 = s.srcIPRsExtractedV6[1:]
			if len(s.srcIPRsExtractedV6) == 0 {
				s.srcIPRsExtractedV6 = nil
			} else if cap(s.srcIPRsExtractedV6) > 1024 && len(s.srcIPRsExtractedV6) < cap(s.srcIPRsExtractedV6)/4 {
				s.srcIPRsExtractedV6 = append([]net.IP(nil), s.srcIPRsExtractedV6...)
			}
			return ip
		}
		if len(s.srcIPRsRawV6) > 0 {
			idx := 0
			if isRandom {
				idx = rnd.Intn(len(s.srcIPRsRawV6))
			}
			ipr := s.srcIPRsRawV6[idx]
			var extracted []net.IP
			if isRandom {
				extracted = ipr.GetRandomX(rnd, 1)
			} else {
				extracted = ipr.Extract(1)
			}
			if ipr.Len.Cmp(big.NewInt(0)) == 0 {
				lastIdx := len(s.srcIPRsRawV6) - 1
				copy(s.srcIPRsRawV6[idx:], s.srcIPRsRawV6[idx+1:])
				s.srcIPRsRawV6[lastIdx] = nil
				s.srcIPRsRawV6 = s.srcIPRsRawV6[:lastIdx]
				if len(s.srcIPRsRawV6) == 0 {
					s.srcIPRsRawV6 = nil
				}
			}
			if len(extracted) > 0 {
				return extracted[0]
			}
		}
		return nil
	}

	// IPv4
	if n := len(s.srcIPRsExtractedV4); n > 0 {
		if isRandom {
			idx := rnd.Intn(n)
			ip := s.srcIPRsExtractedV4[idx]
			s.srcIPRsExtractedV4[idx] = s.srcIPRsExtractedV4[n-1]
			s.srcIPRsExtractedV4[n-1] = nil
			s.srcIPRsExtractedV4 = s.srcIPRsExtractedV4[:n-1]
			if len(s.srcIPRsExtractedV4) == 0 {
				s.srcIPRsExtractedV4 = nil
			}
			return ip
		}
		ip := s.srcIPRsExtractedV4[0]
		s.srcIPRsExtractedV4[0] = nil
		s.srcIPRsExtractedV4 = s.srcIPRsExtractedV4[1:]
		if len(s.srcIPRsExtractedV4) == 0 {
			s.srcIPRsExtractedV4 = nil
		} else if cap(s.srcIPRsExtractedV4) > 1024 && len(s.srcIPRsExtractedV4) < cap(s.srcIPRsExtractedV4)/4 {
			s.srcIPRsExtractedV4 = append([]net.IP(nil), s.srcIPRsExtractedV4...)
		}
		return ip
	}
	if len(s.srcIPRsRawV4) > 0 {
		idx := 0
		if isRandom {
			idx = rnd.Intn(len(s.srcIPRsRawV4))
		}
		ipr := s.srcIPRsRawV4[idx]
		var extracted []net.IP
		if isRandom {
			extracted = ipr.GetRandomX(rnd, 1)
		} else {
			extracted = ipr.Extract(1)
		}
		if ipr.Len.Cmp(big.NewInt(0)) == 0 {
			lastIdx := len(s.srcIPRsRawV4) - 1
			copy(s.srcIPRsRawV4[idx:], s.srcIPRsRawV4[idx+1:])
			s.srcIPRsRawV4[lastIdx] = nil
			s.srcIPRsRawV4 = s.srcIPRsRawV4[:lastIdx]
			if len(s.srcIPRsRawV4) == 0 {
				s.srcIPRsRawV4 = nil
			}
		}
		if len(extracted) > 0 {
			return extracted[0]
		}
	}
	return nil
}

func (s *SourceIPs) retrieveIPsFromIPR(amount int, isRandom bool) (targetIPs []*string) {
	if amount <= 0 {
		return
	}

	rnd := s.tRnd
	if rnd == nil {
		rnd = MyRand
	}

	t_ips := make([]net.IP, 0, amount)
	for len(t_ips) < amount {
		hasV4 := s.hasIP4()
		hasV6 := s.hasIP6()
		if !hasV4 && !hasV6 {
			break
		}

		var chooseV6 bool
		if hasV4 && hasV6 {
			chooseV6 = rnd.Float64() < 0.5
		} else {
			chooseV6 = hasV6
		}

		ip := s.retrieveOneIP(chooseV6, isRandom)
		if ip != nil {
			t_ips = append(t_ips, ip)
		} else {
			break
		}
	}

	for _, t_ip := range t_ips {
		tIP := t_ip.String()
		targetIPs = append(targetIPs, &tIP)
	}

	rnd.Shuffle(len(targetIPs), func(m, n int) {
		targetIPs[m], targetIPs[n] = targetIPs[n], targetIPs[m]
	})
	return
}

func (s *SourceIPs) Shuffle() {
	s.mu.Lock()
	defer s.mu.Unlock()
	rnd := s.tRnd
	if rnd == nil {
		rnd = MyRand
	}
	rnd.Shuffle(len(s.srcHosts), func(m, n int) {
		s.srcHosts[m], s.srcHosts[n] = s.srcHosts[n], s.srcHosts[m]
	})
	rnd.Shuffle(len(s.srcIPRsRawV4), func(m, n int) {
		s.srcIPRsRawV4[m], s.srcIPRsRawV4[n] = s.srcIPRsRawV4[n], s.srcIPRsRawV4[m]
	})
	rnd.Shuffle(len(s.srcIPRsRawV6), func(m, n int) {
		s.srcIPRsRawV6[m], s.srcIPRsRawV6[n] = s.srcIPRsRawV6[n], s.srcIPRsRawV6[m]
	})
	rnd.Shuffle(len(s.srcIPRsExtractedV4), func(m, n int) {
		s.srcIPRsExtractedV4[m], s.srcIPRsExtractedV4[n] = s.srcIPRsExtractedV4[n], s.srcIPRsExtractedV4[m]
	})
	rnd.Shuffle(len(s.srcIPRsExtractedV6), func(m, n int) {
		s.srcIPRsExtractedV6[m], s.srcIPRsExtractedV6[n] = s.srcIPRsExtractedV6[n], s.srcIPRsExtractedV6[m]
	})
}

func (s *SourceIPs) SetRand(mRnd *rand.Rand) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tRnd = mRnd
}

func (s *SourceIPs) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.srcHosts = []*string{}
	s.srcIPRsRawV4 = []*utils.IPRange{}
	s.srcIPRsRawV6 = []*utils.IPRange{}
	s.srcIPRsExtractedV4 = []net.IP{}
	s.srcIPRsExtractedV6 = []net.IP{}
	s.Ports = []int{}
}

func NewSourceIPs() *SourceIPs {
	return &SourceIPs{
		srcHosts:           make([]*string, 0),
		srcIPRsRawV4:       make([]*utils.IPRange, 0),
		srcIPRsRawV6:       make([]*utils.IPRange, 0),
		srcIPRsExtractedV4: make([]net.IP, 0),
		srcIPRsExtractedV6: make([]net.IP, 0),
		Ports:              []int{},
		tRnd:               utils.NewRand(),
	}
}

func NewSourceIPsWithRand(tRnd *rand.Rand) *SourceIPs {
	mSrc := NewSourceIPs()
	mSrc.SetRand(tRnd)
	return mSrc
}

func CopySourceIPs(src *SourceIPs) *SourceIPs {
	src.mu.Lock()
	defer src.mu.Unlock()
	mSrc := NewSourceIPs()
	mSrc.srcHosts = append(mSrc.srcHosts, src.srcHosts...)
	mSrc.srcIPRsRawV4 = append(mSrc.srcIPRsRawV4, src.srcIPRsRawV4...)
	mSrc.srcIPRsRawV6 = append(mSrc.srcIPRsRawV6, src.srcIPRsRawV6...)
	mSrc.srcIPRsExtractedV4 = append(mSrc.srcIPRsExtractedV4, src.srcIPRsExtractedV4...)
	mSrc.srcIPRsExtractedV6 = append(mSrc.srcIPRsExtractedV6, src.srcIPRsExtractedV6...)
	mSrc.Ports = append(mSrc.Ports, src.Ports...)
	mSrc.tRnd = utils.NewRand()
	return mSrc
}

type Task struct {
	Host        *string
	Max_failure int
}

func (t *Task) GetHost() *string {
	return t.Host
}

func (t *Task) GetMaxFailure() int {
	return t.Max_failure
}

func NewTask(host *string, max_failure int) *Task {
	return &Task{
		Host:        host,
		Max_failure: max_failure,
	}
}
