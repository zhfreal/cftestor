package main

import (
	"context"
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	LogLevelDebug   = 31
	LogLevelInfo    = 15
	LogLevelWarning = 7
	LogLevelError   = 3
	LogLevelFatal   = 1
	MyLoggerSpacer  = "    "
)

var (
	CFIPV4 = []string{
		"1.1.1.0/24",
		"1.0.0.0/24",
		"1.1.1.1/32",
		"1.0.0.1/32",
		"173.245.48.0/20",
		"103.21.244.0/22",
		"103.22.200.0/22",
		"103.31.4.0/22",
		"141.101.64.0/18",
		"108.162.192.0/18",
		"190.93.240.0/20",
		"188.114.96.0/20",
		"197.234.240.0/22",
		"198.41.128.0/17",
		"162.158.0.0/15",
		"104.16.0.0/13",
		"104.24.0.0/14",
		"172.64.0.0/13",
		"131.0.72.0/22",
	}
	CFIPV6 = []string{
		"2606:4700:10::6814:0/112",
		"2606:4700:10::ac43:0/112",
		"2606:4700:3000::/48",
		"2606:4700:3001::/48",
		"2606:4700:3002::/48",
		"2606:4700:3003::/48",
		"2606:4700:3004::/48",
		"2606:4700:3005::/48",
		"2606:4700:3006::/48",
		"2606:4700:3007::/48",
		"2606:4700:3008::/48",
		"2606:4700:3009::/48",
		"2606:4700:3010::/48",
		"2606:4700:3011::/48",
		"2606:4700:3012::/48",
		"2606:4700:3013::/48",
		"2606:4700:3014::/48",
		"2606:4700:3015::/48",
		"2606:4700:3016::/48",
		"2606:4700:3017::/48",
		"2606:4700:3018::/48",
		"2606:4700:3019::/48",
		"2606:4700:3020::/48",
		"2606:4700:3021::/48",
		"2606:4700:3022::/48",
		"2606:4700:3023::/48",
		"2606:4700:3024::/48",
		"2606:4700:3025::/48",
		"2606:4700:3026::/48",
		"2606:4700:3027::/48",
		"2606:4700:3028::/48",
		"2606:4700:3029::/48",
		"2606:4700:3030::/48",
		"2606:4700:3031::/48",
		"2606:4700:3032::/48",
		"2606:4700:3033::/48",
		"2606:4700:3034::/48",
		"2606:4700:3035::/48",
		"2606:4700:3036::/48",
		"2606:4700:3037::/48",
		"2606:4700:3038::/48",
		"2606:4700:3039::/48",
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
	Utf8BomBytes    = []byte{0xEF, 0xBB, 0xBF}
	ResultCsvHeader = []string{"TestTime", "IP", "PingRTTinAvg(ms)", "DownloadSpeed(KB/s)", "PingTryOut", "PingSuccess", "PingPingSuccRatio(%)", "PingRTTMin(ms)", "PingRTTMax(ms)", "DownloadTryOut", "DownloadSuccess", "DownloadSuccRatio(%)"}
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (i *arrayFlags) Type() string {
	return "[]string"
}

type LogLevel int

type ContentType int

type LoggerContent struct {
	LogLevel LogLevel
	V        []VerifyResults
}

type MyLogger struct {
	LoggerLevel     LogLevel
	LatestLogLength int
	Space           string
	Pattern         []int
	HeaderPrinted   bool
}

type ResultHttp struct {
	dnsStartAt      time.Time
	dnsEndAt        time.Time
	tcpStartAt      time.Time
	tcpEndAt        time.Time
	tlsStartAt      time.Time
	tlsEndAt        time.Time
	httpReqAt       time.Time
	httpRspAt       time.Time
	bodyReadStartAt time.Time
	bodyReadEndAt   time.Time
}

type SingleResultSlice struct {
	PingSuccess       bool
	PingTimeDuration  time.Duration
	DownloadPerformed bool
	DownloadSuccess   bool
	DownloadDuration  time.Duration
	DownloadSize      int64
}

type SingleVerifyResult struct {
	TestTime    time.Time
	IP          net.IP
	ResultSlice []SingleResultSlice
}

type VerifyResults struct {
	TestTime             time.Time
	IP                   string
	PingCount            int
	PingSuccessCount     int
	PingSuccessRate      float64
	PingDurationAvg      float64
	PingDurationMin      float64
	PingDurationMax      float64
	DownloadCount        int
	DownloadSuccessCount int
	DownloadSuccessRatio float64
	DownloadSpeedAvg     float64
	DownloadSize         int64
	DownloadDurationSec  float64
}

type ResultSpeedSorter []VerifyResults

func (a ResultSpeedSorter) Len() int           { return len(a) }
func (a ResultSpeedSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ResultSpeedSorter) Less(i, j int) bool { return a[i].DownloadSpeedAvg < a[j].DownloadSpeedAvg }

type OverAllStat struct {
	AllPingTasks     int
	PingTaskDone     int
	AllDownloadTasks int
	DownloadTaskDone int
	IPsForPing       int
	IPsForDown       int
	VerifyResults    int
}

func (myLogger *MyLogger) getLogLevelString(lv LogLevel) string {
	if lv == LogLevelDebug {
		return "DEBUG"
	}
	if lv == LogLevelInfo {
		return "INFO"
	}
	if lv == LogLevelWarning {
		return "WARNING"
	}
	if lv == LogLevelError {
		return "ERROR"
	}
	if lv == LogLevelFatal {
		return "FATAL"
	}
	if myLogger.LoggerLevel == LogLevelDebug {
		return "DEBUG"
	}
	if myLogger.LoggerLevel == LogLevelInfo {
		return "INFO"
	}
	if myLogger.LoggerLevel == LogLevelWarning {
		return "WARNING"
	}
	if myLogger.LoggerLevel == LogLevelError {
		return "ERROR"
	}
	if myLogger.LoggerLevel == LogLevelFatal {
		return "FATAL"
	}
	return "INFO"
}

func (myLogger *MyLogger) getPattern(ipv6Mode bool) []int {
	if ipv6Mode {
		return []int{19, 5, 39, 32, 38, 27, 27, 30, 32, 32, 27, 27, 32}
	}
	return []int{19, 5, 20, 32, 38, 27, 27, 30, 32, 32, 27, 27, 32}
}

func getTimeNowStr() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func getTimeNowStrSuffix() string {
	//s := time.Now().Format("20060102150405.999")
	s := time.Now().Format("20060102150405")
	return strings.ReplaceAll(s, ".", "")
}

func (myLogger *MyLogger) NewLogger(lv LogLevel) MyLogger {
	return MyLogger{lv, 0, MyLoggerSpacer, []int{}, false}
}

func (myLogger *MyLogger) print_indent_newline(lv LogLevel, log_str string, newline bool, align bool) {
	if myLogger.LatestLogLength > 0 {
		EraseLine(myLogger.LatestLogLength)
	}
	fmt.Print(getTimeNowStr())
	fmt.Print(myLogger.Space)
	t_log_type_str := myLogger.getLogLevelString(lv)
	if align {
		fmt.Printf("%-8v", t_log_type_str)
	} else {
		fmt.Print(t_log_type_str)
	}
	fmt.Print(myLogger.Space)
	fmt.Print(log_str)
	if newline {
		fmt.Println()
		myLogger.LatestLogLength = 0
	}
}

func (myLogger *MyLogger) debug(debugStr string, newline bool) {
	myLogger.print_indent_newline(LogLevelDebug, debugStr, newline, false)
}

func (myLogger *MyLogger) debugIndent(debugStr string, newline bool) {
	myLogger.print_indent_newline(LogLevelDebug, debugStr, newline, true)
}

func (myLogger *MyLogger) info(infoStr string, newline bool) {
	myLogger.print_indent_newline(LogLevelInfo, infoStr, newline, false)
}

func (myLogger *MyLogger) infoIndent(infoStr string, newline bool) {
	myLogger.print_indent_newline(LogLevelInfo, infoStr, newline, true)
}

func (myLogger *MyLogger) warning(warnStr string, newline bool) {
	myLogger.print_indent_newline(LogLevelWarning, warnStr, newline, false)
}

func (myLogger *MyLogger) warningIndent(warnStr string, newline bool) {
	myLogger.print_indent_newline(LogLevelWarning, warnStr, newline, true)
}

func (myLogger *MyLogger) error(errStr string, newline bool) {
	myLogger.print_indent_newline(LogLevelError, errStr, newline, false)
}

func (myLogger *MyLogger) errorIndent(errStr string, newline bool) {
	myLogger.print_indent_newline(LogLevelError, errStr, newline, true)
}

func (myLogger *MyLogger) fatal(fatalStr string, newline bool) {
	myLogger.print_indent_newline(LogLevelFatal, fatalStr, newline, false)
	os.Exit(1)
}

func (myLogger *MyLogger) fatalIndent(fatalStr string, newline bool) {
	myLogger.print_indent_newline(LogLevelFatal, fatalStr, newline, true)
	os.Exit(1)
}

func (myLogger *MyLogger) Debug(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, MyLoggerSpacer)
	}
	s = strings.TrimSpace(s)
	myLogger.debug(s, true)
}

func (myLogger *MyLogger) DebugIndent(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, MyLoggerSpacer)
	}
	s = strings.TrimSpace(s)
	myLogger.debugIndent(s, true)
}

func (myLogger *MyLogger) Info(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, MyLoggerSpacer)
	}
	s = strings.TrimSpace(s)
	myLogger.info(s, true)
}

func (myLogger *MyLogger) InfoIndent(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, MyLoggerSpacer)
	}
	s = strings.TrimSpace(s)
	myLogger.infoIndent(s, true)
}

func (myLogger *MyLogger) Warning(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, MyLoggerSpacer)
	}
	s = strings.TrimSpace(s)
	myLogger.warning(s, true)
}

func (myLogger *MyLogger) WarningIndent(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, MyLoggerSpacer)
	}
	s = strings.TrimSpace(s)
	myLogger.warningIndent(s, true)
}

func (myLogger *MyLogger) Error(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, MyLoggerSpacer)
	}
	s = strings.TrimSpace(s)
	myLogger.error(s, true)
}

func (myLogger *MyLogger) ErrorIndent(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, MyLoggerSpacer)
	}
	s = strings.TrimSpace(s)
	myLogger.errorIndent(s, true)
}

func (myLogger *MyLogger) Fatal(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, MyLoggerSpacer)
	}
	s = strings.TrimSpace(s)
	myLogger.fatal(s, true)
}

func (myLogger *MyLogger) FatalIndent(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, MyLoggerSpacer)
	}
	s = strings.TrimSpace(s)
	myLogger.fatalIndent(s, true)
}

func (myLogger *MyLogger) Println(logLevel LogLevel, info string) {
	if logLevel == LogLevelDebug {
		myLogger.debug(info, true)
	} else if logLevel == LogLevelInfo {
		myLogger.info(info, true)
	} else if logLevel == LogLevelWarning {
		myLogger.warning(info, true)
	} else if logLevel == LogLevelError {
		myLogger.error(info, true)
	} else if logLevel == LogLevelFatal {
		myLogger.fatal(info, true)
	} else {
		return
	}
}

func (myLogger *MyLogger) PrintlnIndent(logLevel LogLevel, info string) {
	if logLevel == LogLevelDebug {
		myLogger.debugIndent(info, true)
	} else if logLevel == LogLevelInfo {
		myLogger.infoIndent(info, true)
	} else if logLevel == LogLevelWarning {
		myLogger.warningIndent(info, true)
	} else if logLevel == LogLevelError {
		myLogger.errorIndent(info, true)
	} else if logLevel == LogLevelFatal {
		myLogger.fatalIndent(info, true)
	} else {
		return
	}
}

func (myLogger *MyLogger) Print(logLevel LogLevel, info string) {
	if logLevel == LogLevelDebug {
		myLogger.debug(info, false)
	} else if logLevel == LogLevelInfo {
		myLogger.info(info, false)
	} else if logLevel == LogLevelWarning {
		myLogger.warning(info, true)
	} else if logLevel == LogLevelError {
		myLogger.error(info, false)
	} else if logLevel == LogLevelFatal {
		myLogger.fatal(info, false)
	} else {
		return
	}
}

func (myLogger *MyLogger) PrintIntent(logLevel LogLevel, info string) {
	if logLevel == LogLevelDebug {
		myLogger.debugIndent(info, false)
	} else if logLevel == LogLevelInfo {
		myLogger.infoIndent(info, false)
	} else if logLevel == LogLevelWarning {
		myLogger.warningIndent(info, true)
	} else if logLevel == LogLevelError {
		myLogger.errorIndent(info, false)
	} else if logLevel == LogLevelFatal {
		myLogger.fatalIndent(info, false)
	} else {
		return
	}
}

func (myLogger *MyLogger) PrintSingleStat(v LoggerContent, oV OverAllStat) {
	myLogger.PrintStat(v, disableDownload)
	myLogger.PrintOverAllStat(oV)
}

func (myLogger *MyLogger) PrintStat(v LoggerContent, disableDownload bool) {
	// no data for print
	if len(v.V) == 0 {
		return
	}
	// check log level
	if (myLogger.LoggerLevel & v.LogLevel) != v.LogLevel {
		return
	}
	// append enough pattern
	if len(myLogger.Pattern) == 0 {
		myLogger.Pattern = myLogger.getPattern(false)
	}
	// fix space
	if len(myLogger.Space) == 0 {
		myLogger.Space = MyLoggerSpacer
	}
	if myLogger.LatestLogLength > 0 {
		EraseLine(myLogger.LatestLogLength)
	}
	lc := v.V
	var ipv6 = false
	for i := 0; i < len(lc); i++ {
		tIP := net.ParseIP(lc[i].IP)
		if tIP.To4() == nil {
			ipv6 = true
			break
		}
	}
	if !myLogger.HeaderPrinted {
		if !ipv6 {
			myLogger.PrintIntent(LogLevelInfo, fmt.Sprintf("%-15v%s", "IP", myLogger.Space))
		} else {
			myLogger.PrintIntent(LogLevelInfo, fmt.Sprintf("%-39v%s", "IP", myLogger.Space))
		}
		if !disableDownload {
			fmt.Printf("%-11v%s", "Speed(KB/s)", myLogger.Space)
		}

		fmt.Printf("%-11v%s", "PingRTT(ms)", myLogger.Space)
		fmt.Printf("%-11v%s", "PingSR(%)", myLogger.Space)
		// close line, LatestLogLength should be 0
		fmt.Println()
		myLogger.HeaderPrinted = true
	}
	for i := 0; i < len(lc); i++ {
		if !ipv6 {
			myLogger.PrintIntent(v.LogLevel, fmt.Sprintf("%-15v%s", lc[i].IP, myLogger.Space))
		} else {
			myLogger.PrintIntent(v.LogLevel, fmt.Sprintf("%-39v%s", lc[i].IP, myLogger.Space))
		}
		if !disableDownload {
			fmt.Printf("%-11.2f%s", lc[i].DownloadSpeedAvg, myLogger.Space)
		}
		fmt.Printf("%-11.0f%s", lc[i].PingDurationAvg, myLogger.Space)
		fmt.Printf("%-11.2f%s", lc[i].PingSuccessRate*100, myLogger.Space)
		// close line, LatestLogLength should be 0
		fmt.Println()
	}
	myLogger.LatestLogLength = 0
}

func (myLogger *MyLogger) PrintOverAllStat(v OverAllStat) {
	// append enough pattern
	if len(myLogger.Pattern) == 0 {
		myLogger.Pattern = myLogger.getPattern(false)
	}
	// fix space
	if len(myLogger.Space) == 0 {
		myLogger.Space = MyLoggerSpacer
	}
	var t = make([]string, 0)
	t = append(t, getTimeNowStr()+myLogger.Space)
	t = append(t, myLogger.getLogLevelString(LogLevelInfo)+myLogger.Space)
	t = append(t, fmt.Sprintf("TotalQualified:%-5d%s", v.VerifyResults, myLogger.Space))
	if ! downloadOnly {
		t = append(t, fmt.Sprintf("TotalforPingTest:%-5d%s", v.IPsForPing+v.AllPingTasks-v.PingTaskDone, myLogger.Space))
		t = append(t, fmt.Sprintf("TotalPingTested:%-5d%s", v.PingTaskDone, myLogger.Space))
	}
	if ! pingOnly {
		t = append(t, fmt.Sprintf("TotalforSpeedTest:%-5d%s", v.IPsForDown+v.AllDownloadTasks-v.DownloadTaskDone, myLogger.Space))
		t = append(t, fmt.Sprintf("TotalSpeedTested:%-5d%s", v.DownloadTaskDone, myLogger.Space))
	}
	//fix the latest un-closed line
	if myLogger.LatestLogLength > 0 {
		EraseLine(myLogger.LatestLogLength)
	}
	// zero padding according pattern, and print
	var thisLength = 0
	for tI := 0; tI < len(t); tI++ {
		// fix pattern, when pattern don't have enough space
		tV := t[tI]
		fmt.Print(tV)
		thisLength += len(tV)
	}
	// do not close line, LatestLogLength should be the length of this line
	//fmt.Print("\n")
	myLogger.LatestLogLength = thisLength
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func WriteResult(data []VerifyResults, filePath string) {
	var fp = &os.File{}
	var err error
	var w = &csv.Writer{}
	if !fileExists(filePath) {
		fp, err = os.Create(filePath)
		if err != nil {
			log.Fatalf("Create File %v failed with: %v", filePath, err)
		}
		wn, wErr := fp.Write(Utf8BomBytes)
		if wn != len(Utf8BomBytes) && wErr != nil {
			log.Fatalf("Write csv File %v failed with: %v", filePath, err)
		}
		w = csv.NewWriter(fp)
		err = w.Write(ResultCsvHeader)
		if err != nil {
			log.Fatalf("Write csv File %v failed with: %v", filePath, err)
		}
	} else {
		fp, err = os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, os.FileMode(0644))
		if err != nil {
			log.Fatalf("Open File %v failed with: %v", filePath, err)
		}
		w = csv.NewWriter(fp)
	}
	defer func() { _ = fp.Close() }()
	for _, tD := range data {
		err = w.Write([]string{
			tD.TestTime.Format("2006-01-02 15:04:05"),
			tD.IP,
			fmt.Sprintf("%.0f", tD.PingDurationAvg),
			fmt.Sprintf("%.2f", tD.DownloadSpeedAvg),
			fmt.Sprintf("%d", tD.PingCount),
			fmt.Sprintf("%d", tD.PingSuccessCount),
			fmt.Sprintf("%.2f", tD.PingSuccessRate*100),
			fmt.Sprintf("%.0f", tD.PingDurationMin),
			fmt.Sprintf("%.0f", tD.PingDurationMax),
			fmt.Sprintf("%d", tD.DownloadCount),
			fmt.Sprintf("%d", tD.DownloadSuccessCount),
			fmt.Sprintf("%.2f", tD.DownloadSuccessRatio*100),
		})
		if err != nil {
			log.Fatalf("Write csv File %v failed with: %v", filePath, err)
		}
	}
	w.Flush()
}

func ParseUrl(urlStr string) (tHostName string, tPort int) {
	urlStr = strings.TrimSpace(urlStr)
	if len(urlStr) == 0 {
		urlStr = DefaultTestUrl
	}
	u, err := url.ParseRequestURI(urlStr)
	if err != nil || u == nil || len(u.Host) == 0 {
		myLogger.Fatal(fmt.Sprintf("url is not valid: %s\n", urlStr))
		// it will never get here, just fool IDE
		panic(u)
	}
	tHost := strings.Split(u.Host, ":")
	tHostName = tHost[0]
	if len(tHost) > 1 {
		tPort, err = strconv.Atoi(tHost[1])
		if err != nil {
			myLogger.Fatal(fmt.Sprintf("url is not valid: %s\n", urlStr))
		}
	} else {
		if u.Scheme == "http" {
			tPort = 80
		} else if u.Scheme == "https" {
			tPort = 443
		}
	}
	return
}

func initRandSeed() {
	rand.Seed(time.Now().UnixNano())
}

func getASNAndCity() (ASN int, city string) {
	if defaultASN > 0 {
		ASN = defaultASN
		city = defaultCity
		return
	}
	for i := 0; i < 3; i++ {
		response, err := http.Get("https://speed.cloudflare.com/__down")
		// pingect is failed(network error), won't continue
		if err != nil {
			myLogger.Error(fmt.Sprintf("An error occurred while request ASN and city info from cloudflare: %v\n", err))
			time.Sleep(time.Duration(interval) * time.Millisecond)
			continue
		}
		if response == nil {
			myLogger.Error("An error occurred while request ASN and city info from cloudflare: response is empty")
			time.Sleep(time.Duration(interval) * time.Millisecond)
			continue
		}
		//fmt.Printf("%T - %v", response.Header, response.Header)
		if values, ok := response.Header["Cf-Meta-Asn"]; ok {
			if len(values) > 0 {
				if ASN, err = strconv.Atoi(values[0]); err != nil {
					myLogger.Error(fmt.Sprintf("An error occurred while convert ASN for header: %T, %v", values[0], values[0]))
				}
			}
		}
		if values, ok := response.Header["Cf-Meta-City"]; ok {
			if len(values) > 0 {
				city = values[0]
			}
		}
		defaultASN = ASN
		defaultCity = city
		break
	}
	return
}

func PrintFinalStat(v []VerifyResults, disableDownload bool) {
	// no data for print
	if len(v) == 0 {
		return
	}
	var ipv6 = false
	for i := 0; i < len(v); i++ {
		tIP := net.ParseIP(v[i].IP)
		if tIP.To4() == nil {
			ipv6 = true
			break
		}
	}
	fmt.Printf("%-19v%s", "TestTime", myLogger.Space)
	if !ipv6 {
		fmt.Printf("%-15v%s", "IP", myLogger.Space)
	} else {
		fmt.Printf("%-39v%s", "IP", myLogger.Space)
	}
	if !disableDownload {
		fmt.Printf("%-11v%s", "Speed(KB/s)", myLogger.Space)
	}
	fmt.Printf("%-11v%s", "PingRTT(ms)", myLogger.Space)
	fmt.Printf("%-11v%s", "PingSR(%)", myLogger.Space)
	// close line, LatestLogLength should be 0
	fmt.Println()
	for i := 0; i < len(v); i++ {
		fmt.Printf("%-19v%s", v[i].TestTime.Format("2006-01-02 15:04:05"), myLogger.Space)
		if !ipv6 {
			fmt.Printf("%-15v%s", v[i].IP, myLogger.Space)
		} else {
			fmt.Printf("%-39v%s", v[i].IP, myLogger.Space)
		}
		if !disableDownload {
			fmt.Printf("%-11.2f%s", v[i].DownloadSpeedAvg, myLogger.Space)
		}
		fmt.Printf("%-11.0f%s", v[i].PingDurationAvg, myLogger.Space)
		fmt.Printf("%-11.2f%s", v[i].PingSuccessRate*100, myLogger.Space)
		// close line, LatestLogLength should be 0
		fmt.Println()
	}
}

func InsertIntoDb(verifyResultsSlice []VerifyResults, dbFile string) {
	if len(verifyResultsSlice) > 0 && storeToDB {
		dbRecords := make([]CFTestDetail, 0)
		ASN, city := getASNAndCity()
		for _, v := range verifyResultsSlice {
			record := CFTestDetail{}
			record.ASN = ASN
			record.City = city
			record.Label = suffixLabel
			record.TestTimeStr = v.TestTime.Format("2006-01-02 15:04:05")
			record.IP = v.IP
			record.PingCount = v.PingCount
			record.PingSuccessCount = v.PingSuccessCount
			record.PingSuccessRate = v.PingSuccessRate
			record.PingDurationAvg = v.PingDurationAvg
			record.PingDurationMin = v.PingDurationMin
			record.PingDurationMax = v.PingDurationMax
			record.DownloadCount = v.DownloadCount
			record.DownloadSuccessCount = v.DownloadSuccessCount
			record.DownloadSuccessRatio = v.DownloadSuccessRatio
			record.DownloadSpeedAvg = v.DownloadSpeedAvg
			record.DownloadSize = v.DownloadSize
			record.DownloadDurationSec = v.DownloadDurationSec
			dbRecords = append(dbRecords, record)
		}
		InsertData(dbRecords, dbFile)
	}
}

func WithHTTPStat(ctx context.Context, r *ResultHttp) context.Context {
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		DNSStart: func(i httptrace.DNSStartInfo) {
			r.dnsStartAt = time.Now()
		},
		DNSDone: func(i httptrace.DNSDoneInfo) {
			r.dnsEndAt = time.Now()
		},
		ConnectStart: func(_, _ string) {
			r.tcpStartAt = time.Now()
			// When connecting to IP (When no DNS lookup)
			if r.dnsStartAt.IsZero() {
				r.dnsStartAt = r.tcpStartAt
				r.dnsEndAt = r.tcpStartAt
			}
		},
		ConnectDone: func(network, addr string, err error) {
			r.tcpEndAt = time.Now()
		},
		TLSHandshakeStart: func() {
			r.tlsStartAt = time.Now()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			r.tlsEndAt = time.Now()
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			r.httpReqAt = time.Now()
			if r.dnsStartAt.IsZero() && r.tcpStartAt.IsZero() {
				now := r.httpReqAt
				r.dnsStartAt = now
				r.dnsEndAt = now
				r.tcpStartAt = now
				r.tcpEndAt = now
			}
		},
		GotFirstResponseByte: func() {
			r.httpRspAt = time.Now()
			r.bodyReadStartAt = r.httpRspAt
		},
	})
}

func GetDialContextByAddr(addrPort string) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		c, e := (&net.Dialer{}).DialContext(ctx, network, addrPort)
		return c, e
	}
}

func singleResultStatistic(out SingleVerifyResult, statisticDownload bool) VerifyResults {
	var tVerifyResult = VerifyResults{out.TestTime, "", 0, 0, 0.0,
		0.0, 0.0, 0.0, 0,
		0, 0.0, 0.0, 0, 0.0}
	tVerifyResult.IP = out.IP.String()
	if len(out.ResultSlice) == 0 {
		return tVerifyResult
	}
	tVerifyResult.PingCount = len(out.ResultSlice)
	var tDurationsAll = 0.0
	var tDownloadDurations float64
	var tDownloadSizes int64
	for _, v := range out.ResultSlice {
		if v.PingSuccess {
			tVerifyResult.PingSuccessCount += 1
			tVerifyResult.DownloadCount += 1
			tDuration := float64(v.PingTimeDuration) / float64(time.Millisecond)
			tDurationsAll = tDurationsAll + tDuration
			if tDuration > tVerifyResult.PingDurationMax {
				tVerifyResult.PingDurationMax = tDuration
			}
			if tVerifyResult.PingDurationMin <= 0.0 || tDuration < tVerifyResult.PingDurationMin {
				tVerifyResult.PingDurationMin = tDuration
			}
			if statisticDownload && v.DownloadPerformed && v.DownloadSuccess {
				tVerifyResult.DownloadSuccessCount += 1
				tDownloadDurations += math.Round(float64(v.DownloadDuration) / float64(time.Second))
				tDownloadSizes += v.DownloadSize
			}
		}
	}
	if tVerifyResult.PingSuccessCount > 0 {
		tVerifyResult.PingDurationAvg = tDurationsAll / float64(tVerifyResult.PingSuccessCount)
		tVerifyResult.PingSuccessRate = float64(tVerifyResult.PingSuccessCount) / float64(tVerifyResult.PingCount)
	}
	// we statistic download speed while the downloaded file size is greater than DownloadSizeMin
	if statisticDownload && tVerifyResult.DownloadSuccessCount > 0 && tDownloadSizes > DownloadSizeMin {
		tVerifyResult.DownloadSpeedAvg = float64(tDownloadSizes) / tDownloadDurations / 1000
		tVerifyResult.DownloadSuccessRatio = float64(tVerifyResult.DownloadSuccessCount) / float64(tVerifyResult.DownloadCount)
		tVerifyResult.DownloadSize = tDownloadSizes
		tVerifyResult.DownloadDurationSec = tDownloadDurations
	}
	return tVerifyResult
}

func EraseLine(n int) {
	if n <= 0 {
		return
	}
	fmt.Printf("%s", strings.Repeat("\b \b", n))
}
