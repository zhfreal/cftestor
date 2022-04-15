package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

const (
	logLevelDebug   = 31
	logLevelInfo    = 15
	logLevelWarning = 7
	logLevelError   = 3
	logLevelFatal   = 1
	myIndent        = "  "
)

var (
	CFIPV4 = []string{
		"1.1.1.0/24",
		"1.0.0.0/24",
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
		"2606:4700::6810:0/111",
		"2606:4700::6812:0/111",
		"2606:4700:10::6814:0/112",
		"2606:4700:10::6816:0/112",
		"2606:4700:10::6817:0/112",
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
		"DLSpeed(DLS, KB/s)",
		"DelayAvg(DA, ms)",
		"DelaySource(DS)",
		"DTPassedRate(DTPR, %)",
		"DTCount(DTC)",
		"DTPassedCount(DTPC)",
		"DelayMin(DMI, ms)",
		"DelayMax(DMX, ms)",
		"DLTCount(DLTC)",
		"DLTPassedCount(DLTPC)",
		"DLTPassedRate(DLPR, %)",
	}
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

type loggerContent struct {
	loglevel LogLevel
	v        []VerifyResults
}

type MyLogger struct {
	loggerLevel     LogLevel
	latestLogLength int
	indent          string
	pattern         []int
	headerPrinted   bool
}

type singleResult struct {
	dTPassedCount  bool
	dTDuration     time.Duration
	httpRRDuration time.Duration
	dLTWasDone     bool
	dLTPassed      bool
	dLTDuration    time.Duration
	dLTDataSize    int64
}

type singleVerifyResult struct {
	testTime    time.Time
	ip          net.IP
	resultSlice []singleResult
}

type VerifyResults struct {
	testTime time.Time
	ip       string
	dtc      int
	dtpc     int
	dtpr     float64
	da       float64
	dmi      float64
	dmx      float64
	dltc     int
	dltpc    int
	dltpr    float64
	dls      float64
	dlds     int64
	dltd     float64
}

type resultSpeedSorter []VerifyResults

func (a resultSpeedSorter) Len() int           { return len(a) }
func (a resultSpeedSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a resultSpeedSorter) Less(i, j int) bool { return a[i].dls < a[j].dls }

type overAllStat struct {
	allDTTasks    int
	dtTasksDoned  int
	allDLTTasks   int
	dltTasksDoned int
	dtCached      int
	dltCached     int
	verifyResults int
}

func (myLogger *MyLogger) getLogLevelString(lv LogLevel) string {
	if lv == logLevelDebug {
		return "DEBUG"
	}
	if lv == logLevelInfo {
		return "INFO"
	}
	if lv == logLevelWarning {
		return "WARNING"
	}
	if lv == logLevelError {
		return "ERROR"
	}
	if lv == logLevelFatal {
		return "FATAL"
	}
	if myLogger.loggerLevel == logLevelDebug {
		return "DEBUG"
	}
	if myLogger.loggerLevel == logLevelInfo {
		return "INFO"
	}
	if myLogger.loggerLevel == logLevelWarning {
		return "WARNING"
	}
	if myLogger.loggerLevel == logLevelError {
		return "ERROR"
	}
	if myLogger.loggerLevel == logLevelFatal {
		return "FATAL"
	}
	return "INFO"
}

func (myLogger *MyLogger) getPattern() []int {
	if haveIPv6() {
		return []int{19, 5, 39, 32, 38, 27, 27, 30, 32, 32, 27, 27, 32}
	}
	return []int{19, 5, 20, 32, 38, 27, 27, 30, 32, 32, 27, 27, 32}
}

func getTimeNowStr() string {
	return time.Now().Format("15:04:05")
}

func getTimeNowStrSuffix() string {
	//s := time.Now().Format("20060102150405.999")
	s := time.Now().Format("20060102150405")
	return strings.ReplaceAll(s, ".", "")
}

func (myLogger *MyLogger) newLogger(lv LogLevel) MyLogger {
	return MyLogger{lv, 0, myIndent, []int{}, false}
}

func (myLogger *MyLogger) print_indent_newline(lv LogLevel, log_str string, newline bool, align bool) {
	if myLogger.latestLogLength > 0 {
		EraseLine(myLogger.latestLogLength)
	}
	fmt.Print(getTimeNowStr())
	fmt.Print(myLogger.indent)
	t_log_type_str := myLogger.getLogLevelString(lv)
	if align {
		fmt.Printf("%-8v", t_log_type_str)
	} else {
		fmt.Print(t_log_type_str)
	}
	fmt.Print(myLogger.indent)
	fmt.Print(log_str)
	if newline {
		fmt.Println()
		myLogger.latestLogLength = 0
	}
}

func (myLogger *MyLogger) debug(debugStr string, newline bool) {
	myLogger.print_indent_newline(logLevelDebug, debugStr, newline, false)
}

func (myLogger *MyLogger) debugIndent(debugStr string, newline bool) {
	myLogger.print_indent_newline(logLevelDebug, debugStr, newline, true)
}

func (myLogger *MyLogger) info(infoStr string, newline bool) {
	myLogger.print_indent_newline(logLevelInfo, infoStr, newline, false)
}

func (myLogger *MyLogger) infoIndent(infoStr string, newline bool) {
	myLogger.print_indent_newline(logLevelInfo, infoStr, newline, true)
}

func (myLogger *MyLogger) warning(warnStr string, newline bool) {
	myLogger.print_indent_newline(logLevelWarning, warnStr, newline, false)
}

func (myLogger *MyLogger) warningIndent(warnStr string, newline bool) {
	myLogger.print_indent_newline(logLevelWarning, warnStr, newline, true)
}

func (myLogger *MyLogger) error(errStr string, newline bool) {
	myLogger.print_indent_newline(logLevelError, errStr, newline, false)
}

func (myLogger *MyLogger) errorIndent(errStr string, newline bool) {
	myLogger.print_indent_newline(logLevelError, errStr, newline, true)
}

func (myLogger *MyLogger) fatal(fatalStr string, newline bool) {
	myLogger.print_indent_newline(logLevelFatal, fatalStr, newline, false)
	os.Exit(1)
}

func (myLogger *MyLogger) fatalIndent(fatalStr string, newline bool) {
	myLogger.print_indent_newline(logLevelFatal, fatalStr, newline, true)
	os.Exit(1)
}

func (myLogger *MyLogger) Debug(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, myIndent)
	}
	s = strings.TrimSpace(s)
	myLogger.debug(s, true)
}

func (myLogger *MyLogger) DebugIndent(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, myIndent)
	}
	s = strings.TrimSpace(s)
	myLogger.debugIndent(s, true)
}

func (myLogger *MyLogger) Info(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, myIndent)
	}
	s = strings.TrimSpace(s)
	myLogger.info(s, true)
}

func (myLogger *MyLogger) InfoIndent(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, myIndent)
	}
	s = strings.TrimSpace(s)
	myLogger.infoIndent(s, true)
}

func (myLogger *MyLogger) Warning(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, myIndent)
	}
	s = strings.TrimSpace(s)
	myLogger.warning(s, true)
}

func (myLogger *MyLogger) WarningIndent(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, myIndent)
	}
	s = strings.TrimSpace(s)
	myLogger.warningIndent(s, true)
}

func (myLogger *MyLogger) Error(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, myIndent)
	}
	s = strings.TrimSpace(s)
	myLogger.error(s, true)
}

func (myLogger *MyLogger) ErrorIndent(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, myIndent)
	}
	s = strings.TrimSpace(s)
	myLogger.errorIndent(s, true)
}

func (myLogger *MyLogger) Fatal(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, myIndent)
	}
	s = strings.TrimSpace(s)
	myLogger.fatal(s, true)
}

func (myLogger *MyLogger) FatalIndent(info ...interface{}) {
	var s string
	for _, t := range info {
		s += fmt.Sprintf("%v%s", t, myIndent)
	}
	s = strings.TrimSpace(s)
	myLogger.fatalIndent(s, true)
}

func (myLogger *MyLogger) Println(logLevel LogLevel, info string) {
	if logLevel == logLevelDebug {
		myLogger.debug(info, true)
	} else if logLevel == logLevelInfo {
		myLogger.info(info, true)
	} else if logLevel == logLevelWarning {
		myLogger.warning(info, true)
	} else if logLevel == logLevelError {
		myLogger.error(info, true)
	} else if logLevel == logLevelFatal {
		myLogger.fatal(info, true)
	} else {
		return
	}
}

func (myLogger *MyLogger) PrintlnIndent(logLevel LogLevel, info string) {
	if logLevel == logLevelDebug {
		myLogger.debugIndent(info, true)
	} else if logLevel == logLevelInfo {
		myLogger.infoIndent(info, true)
	} else if logLevel == logLevelWarning {
		myLogger.warningIndent(info, true)
	} else if logLevel == logLevelError {
		myLogger.errorIndent(info, true)
	} else if logLevel == logLevelFatal {
		myLogger.fatalIndent(info, true)
	} else {
		return
	}
}

func (myLogger *MyLogger) Print(logLevel LogLevel, info string) {
	if logLevel == logLevelDebug {
		myLogger.debug(info, false)
	} else if logLevel == logLevelInfo {
		myLogger.info(info, false)
	} else if logLevel == logLevelWarning {
		myLogger.warning(info, true)
	} else if logLevel == logLevelError {
		myLogger.error(info, false)
	} else if logLevel == logLevelFatal {
		myLogger.fatal(info, false)
	} else {
		return
	}
}

func (myLogger *MyLogger) PrintIntent(logLevel LogLevel, info string) {
	if logLevel == logLevelDebug {
		myLogger.debugIndent(info, false)
	} else if logLevel == logLevelInfo {
		myLogger.infoIndent(info, false)
	} else if logLevel == logLevelWarning {
		myLogger.warningIndent(info, true)
	} else if logLevel == logLevelError {
		myLogger.errorIndent(info, false)
	} else if logLevel == logLevelFatal {
		myLogger.fatalIndent(info, false)
	} else {
		return
	}
}

func (myLogger *MyLogger) PrintSingleStat(v loggerContent, oV overAllStat) {
	myLogger.PrintStat(v, disableDownload)
	myLogger.PrintOverAllStat(oV)
}

func (myLogger *MyLogger) PrintStat(v loggerContent, disableDownload bool) {
	// no data for print
	if len(v.v) == 0 {
		return
	}
	// check log level
	if (myLogger.loggerLevel & v.loglevel) != v.loglevel {
		return
	}
	// append enough pattern
	if len(myLogger.pattern) == 0 {
		myLogger.pattern = myLogger.getPattern()
	}
	// fix space
	if len(myLogger.indent) == 0 {
		myLogger.indent = myIndent
	}
	if myLogger.latestLogLength > 0 {
		EraseLine(myLogger.latestLogLength)
	}
	lc := v.v
	var ipv6 = false
	for i := 0; i < len(lc); i++ {
		tIP := net.ParseIP(lc[i].ip)
		if tIP.To4() == nil {
			ipv6 = true
			break
		}
	}
	if !myLogger.headerPrinted {
		if !ipv6 {
			myLogger.PrintIntent(logLevelInfo, fmt.Sprintf("%-15v%s", "IP", myLogger.indent))
		} else {
			myLogger.PrintIntent(logLevelInfo, fmt.Sprintf("%-39v%s", "IP", myLogger.indent))
		}
		if !disableDownload {
			fmt.Printf("%-11v%s", "Speed(KB/s)", myLogger.indent)
		}

		fmt.Printf("%-12v%s", "DelayAvg(ms)", myLogger.indent)
		fmt.Printf("%-12v%s", "Stability(%)", myLogger.indent)
		// close line, LatestLogLength should be 0
		fmt.Println()
		myLogger.headerPrinted = true
	}
	for i := 0; i < len(lc); i++ {
		if !ipv6 {
			myLogger.PrintIntent(v.loglevel, fmt.Sprintf("%-15v%s", lc[i].ip, myLogger.indent))
		} else {
			myLogger.PrintIntent(v.loglevel, fmt.Sprintf("%-39v%s", lc[i].ip, myLogger.indent))
		}
		if !disableDownload {
			fmt.Printf("%-11.2f%s", lc[i].dls, myLogger.indent)
		}
		fmt.Printf("%-12.0f%s", lc[i].da, myLogger.indent)
		fmt.Printf("%-12.2f%s", lc[i].dtpr*100, myLogger.indent)
		// close line, LatestLogLength should be 0
		fmt.Println()
	}
	myLogger.latestLogLength = 0
}

func (myLogger *MyLogger) PrintOverAllStat(v overAllStat) {
	// append enough pattern
	if len(myLogger.pattern) == 0 {
		myLogger.pattern = myLogger.getPattern()
	}
	// fix space
	if len(myLogger.indent) == 0 {
		myLogger.indent = myIndent
	}
	var t = make([]string, 0)
	t = append(t, getTimeNowStr()+myLogger.indent)
	t = append(t, myLogger.getLogLevelString(logLevelInfo)+myLogger.indent)
	t = append(t, fmt.Sprintf("Qualified:%-5d%s", v.verifyResults, myLogger.indent))
	if !dltOnly {
		t = append(t, fmt.Sprintf("IPsCached(delay):%-5d%s", v.dtCached+v.allDTTasks-v.dtTasksDoned, myLogger.indent))
		t = append(t, fmt.Sprintf("IPsTested(delay):%-5d%s", v.dtTasksDoned, myLogger.indent))
	}
	if !dtOnly {
		t = append(t, fmt.Sprintf("IPsCached(speed):%-5d%s", v.dltCached+v.allDLTTasks-v.dltTasksDoned, myLogger.indent))
		t = append(t, fmt.Sprintf("IPsTested(speed):%-5d%s", v.dltTasksDoned, myLogger.indent))
	}
	//fix the latest un-closed line
	if myLogger.latestLogLength > 0 {
		EraseLine(myLogger.latestLogLength)
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
	myLogger.latestLogLength = thisLength
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
		wn, wErr := fp.Write(utf8BomBytes)
		if wn != len(utf8BomBytes) && wErr != nil {
			log.Fatalf("Write csv File %v failed with: %v", filePath, err)
		}
		w = csv.NewWriter(fp)
		err = w.Write(resultCsvHeader)
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
	// "TestTime",
	// "IP",
	// "DLSpeed(DLS, KB/s)",
	// "DelayAvg(DA, ms)",
	// "DelaySource(DS)",
	// "DTPassedRate(DTPR, %)",
	// "DTCount(DTC)",
	// "DTPassedCount(DTPC)",
	// "DelayMin(DMI, ms)",
	// "DelayMax(DMX, ms)",
	// "DLTCount(DLTC)",
	// "DLTPassedCount(DLTPC)",
	// "DLTPassedRate(DLTPR, %)"
	for _, tD := range data {
		err = w.Write([]string{
			tD.testTime.Format("2006-01-02 15:04:05"),
			tD.ip,
			fmt.Sprintf("%.2f", tD.dls),
			fmt.Sprintf("%.0f", tD.da),
			dtSource,
			fmt.Sprintf("%.2f", tD.dtpr*100),
			fmt.Sprintf("%d", tD.dtc),
			fmt.Sprintf("%d", tD.dtpc),
			fmt.Sprintf("%.0f", tD.dmi),
			fmt.Sprintf("%.0f", tD.dmx),
			fmt.Sprintf("%d", tD.dltc),
			fmt.Sprintf("%d", tD.dltpc),
			fmt.Sprintf("%.2f", tD.dltpr*100),
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
		urlStr = defaultTestUrl
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
	myRand.Seed(time.Now().UnixNano())
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
		tIP := net.ParseIP(v[i].ip)
		if tIP.To4() == nil {
			ipv6 = true
			break
		}
	}
	fmt.Printf("%-8v%s", "TestTime", myLogger.indent)
	if !ipv6 {
		fmt.Printf("%-15v%s", "IP", myLogger.indent)
	} else {
		fmt.Printf("%-39v%s", "IP", myLogger.indent)
	}
	if !disableDownload {
		fmt.Printf("%-11v%s", "Speed(KB/s)", myLogger.indent)
	}
	fmt.Printf("%-12v%s", "DelayAvg(ms)", myLogger.indent)
	fmt.Printf("%-12v%s", "Stability(%)", myLogger.indent)
	// close line, LatestLogLength should be 0
	fmt.Println()
	for i := 0; i < len(v); i++ {
		fmt.Printf("%-8v%s", v[i].testTime.Format("15:04:05"), myLogger.indent)
		if !ipv6 {
			fmt.Printf("%-15v%s", v[i].ip, myLogger.indent)
		} else {
			fmt.Printf("%-39v%s", v[i].ip, myLogger.indent)
		}
		if !disableDownload {
			fmt.Printf("%-11.2f%s", v[i].dls, myLogger.indent)
		}
		fmt.Printf("%-12.0f%s", v[i].da, myLogger.indent)
		fmt.Printf("%-12.2f%s", v[i].dtpr*100, myLogger.indent)
		// close line, LatestLogLength should be 0
		fmt.Println()
	}
}

func InsertIntoDb(verifyResultsSlice []VerifyResults, dbFile string) {
	if len(verifyResultsSlice) > 0 && storeToDB {
		dbRecords := make([]cfTestDetail, 0)
		ASN, city := getASNAndCity()
		for _, v := range verifyResultsSlice {
			record := cfTestDetail{}
			record.asn = ASN
			record.city = city
			record.label = suffixLabel
			record.testTimeStr = v.testTime.Format("2006-01-02 15:04:05")
			record.ip = v.ip
			record.dtc = v.dtc
			record.dtpc = v.dtpc
			record.dtpr = v.dtpr
			record.da = v.da
			record.dmi = v.dmi
			record.dmx = v.dmx
			record.dltc = v.dltc
			record.dltpc = v.dltpc
			record.dltpr = v.dltpr
			record.dls = v.dls
			record.dlds = v.dlds
			record.dltd = v.dltd
			dbRecords = append(dbRecords, record)
		}
		insertData(dbRecords, dbFile)
	}
}

func GetDialContextByAddr(addrPort string) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		c, e := (&net.Dialer{}).DialContext(ctx, network, addrPort)
		return c, e
	}
}

func singleResultStatistic(out singleVerifyResult, statisticDownload bool) VerifyResults {
	var tVerifyResult = VerifyResults{out.testTime, "",
		0, 0, 0.0, 0.0, 0.0, 0.0,
		0, 0, 0.0, 0.0, 0, 0.0}
	tVerifyResult.ip = out.ip.String()
	if len(out.resultSlice) == 0 {
		return tVerifyResult
	}
	tVerifyResult.dtc = len(out.resultSlice)
	var tDurationsAll = 0.0
	var tDownloadDurations float64
	var tDownloadSizes int64
	for _, v := range out.resultSlice {
		if v.dTPassedCount {
			tVerifyResult.dtpc += 1
			tVerifyResult.dltc += 1
			tDuration := float64(v.dTDuration) / float64(time.Millisecond)
			// if pingViaHttps, it should add the http duration
			if dtHttps {
				tDuration = +float64(v.httpRRDuration) / float64(time.Millisecond)
			}
			tDurationsAll += tDuration
			if tDuration > tVerifyResult.dmx {
				tVerifyResult.dmx = tDuration
			}
			if tVerifyResult.dmi <= 0.0 || tDuration < tVerifyResult.dmi {
				tVerifyResult.dmi = tDuration
			}
			if statisticDownload && v.dLTWasDone && v.dLTPassed {
				tVerifyResult.dltpc += 1
				tDownloadDurations += math.Round(float64(v.dLTDuration) / float64(time.Second))
				tDownloadSizes += v.dLTDataSize
			}
		}
	}
	if tVerifyResult.dtpc > 0 {
		tVerifyResult.da = tDurationsAll / float64(tVerifyResult.dtpc)
		tVerifyResult.dtpr = float64(tVerifyResult.dtpc) / float64(tVerifyResult.dtc)
	}
	// we statistic download speed while the downloaded file size is greater than DownloadSizeMin
	if statisticDownload && tVerifyResult.dltpc > 0 && tDownloadSizes > downloadSizeMin {
		tVerifyResult.dls = float64(tDownloadSizes) / tDownloadDurations / 1000
		tVerifyResult.dltpr = float64(tVerifyResult.dltpc) / float64(tVerifyResult.dltc)
		tVerifyResult.dlds = tDownloadSizes
		tVerifyResult.dltd = tDownloadDurations
	}
	return tVerifyResult
}

func EraseLine(n int) {
	if n <= 0 {
		return
	}
	fmt.Printf("%s", strings.Repeat("\b \b", n))
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

func NewIPRangeFromIP(StartIP net.IP, EndIP net.IP) *ipRange {
	return new(ipRange).init(StartIP, EndIP)
}

func NewIPRangeFromString(StartIPStr string, EndIPStr string) *ipRange {
	StartIP := net.ParseIP(StartIPStr)
	EndIP := net.ParseIP(EndIPStr)
	return new(ipRange).init(StartIP, EndIP)
}

func NewIPRangeFromCIDR(cidr string) *ipRange {
	ip, ipCIDR, err := net.ParseCIDR(cidr)
	// there an error during parsing process, or its an ip
	if err != nil {
		tIP := net.ParseIP(cidr)
		// pure ipv6 address
		if tIP != nil {
			return new(ipRange).init(tIP, tIP)
		}
		return nil
	}
	// 192.168.1.1/24 is equal to 192.168.1.1/32, just return 192.168.1.1
	if !ip.Equal(ipCIDR.IP) {
		return new(ipRange).init(ip, ip)
	}
	var StartIP net.IP
	if len(ipCIDR.IP) == net.IPv4len {
		StartIP = ip.To4()
	} else {
		StartIP = ip.To16()
	}
	EndIP := net.IP(make([]byte, len(StartIP)))
	for i := 0; i < len(StartIP); i++ {
		EndIP[i] = StartIP[i] | ^ipCIDR.Mask[i]
	}
	return new(ipRange).init(StartIP, EndIP)
}

func fillBytes(s []byte, l int) []byte {
	if len(s) >= l {
		return s
	}
	newBytes := make([]byte, l)
	i := len(s) - 1
	j := len(newBytes) - 1
	for i >= 0 {
		newBytes[j] = s[i]
		i--
		j--
	}
	return newBytes
}

func makeBytes(num uint, len int) []byte {
	newBytes := make([]byte, len)
	for i := int(len - 1); i >= 0; i-- {
		newBytes[i] = byte(num % (1 << 8))
		num = num >> 8
	}
	// beyond the boundry
	if num > 0 {
		return nil
	}
	return newBytes
}

func ipShift(ip net.IP, num []byte) net.IP {
	// it's too big
	if len(num) > len(ip) {
		return nil
	}
	newIP := net.IP(make([]byte, len(ip)))
	_ = copy(newIP, ip)
	num = fillBytes(num, len(newIP))
	add := uint(0)
	lengthForThis := len(newIP)
	for i := lengthForThis - 1; i >= 0; i-- {
		m := uint(newIP[i]) + uint(num[i]) + add
		add = m >> 8
		newIP[i] = byte(m % (1 << 8))
	}
	// beyond the boundary
	if add > 0 {
		return nil
	} else {
		return newIP
	}
}

func ipShiftReverse(ip net.IP, num []byte) net.IP {
	if len(num) > len(ip) {
		return nil
	}
	newIP := net.IP(make([]byte, len(ip)))
	_ = copy(newIP, ip)
	num = fillBytes(num, len(newIP))
	reduce := 0
	lengthForThis := len(newIP)
	for i := lengthForThis - 1; i >= 0; i-- {
		m := int(newIP[i]) - int(num[i]) - reduce
		if m >= 0 {
			newIP[i] = byte(m)
		} else {
			newIP[i] = byte(1<<8 + m)
			reduce = 1
		}
	}
	// beyond the boundary
	if reduce > 0 {
		return nil
	}
	return newIP
}

func (ipr *ipRange) Extract(num int) (IPList []net.IP) {
	if !ipr.isValid() {
		return nil
	}
	// num should greate than 0
	if num <= 0 {
		return nil
	}
	// no more ip for extracted
	if ipr.Extracted || ipr.Len.Cmp(big.NewInt(0)) == 0 {
		return nil
	}
	numBig := big.NewInt(int64(num))
	size := ipr.length()
	// no enough IPs to extract
	if size.Cmp(numBig) == -1 {
		return nil
	}
	newIP := ipr.IPStart
	IPList = append(IPList, newIP)
	num--
	for num > 0 {
		num_in_bytes := makeBytes(uint(1), len(newIP))
		newIP = ipShift(newIP, num_in_bytes)
		// some error occured
		if newIP == nil {
			return nil
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
		return nil
	}
	// num should greate than 0
	if num <= 0 {
		return nil
	}
	// no more ip for extracted
	if ipr.Extracted || ipr.Len.Cmp(big.NewInt(0)) == 0 {
		return nil
	}
	numBig := big.NewInt(int64(num))
	size := ipr.length()
	// no enough IPs to extract
	if size.Cmp(numBig) == -1 {
		return nil
	}
	newIP := ipr.IPEnd
	IPList = append(IPList, newIP)
	num--
	for num > 0 {
		num_in_bytes := makeBytes(uint(1), len(newIP))
		newIP = ipShiftReverse(newIP, num_in_bytes)
		// some error ocurred
		if newIP == nil {
			return nil
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
	// we limit the max retult length to MaxHostLen (currently, 65536), if it's to big, return nil
	// or it's don't have any IPS to extract, return nil
	if ipr.Extracted || ipr.Len.Cmp(big.NewInt(0)) == 0 || ipr.Len.Cmp(big.NewInt(maxHostLen)) == 1 {
		return nil
	}
	return ipr.Extract(int(ipr.Len.Int64()))
}

func (ipr *ipRange) GetRandomX(num int) (IPList []net.IP) {
	// or it's don't have any IPS to extract, return nil
	if ipr.Extracted || ipr.Len.Cmp(big.NewInt(0)) == 0 {
		return nil
	}
	// we extract all while ipr don't have enogth ips for extracted
	if big.NewInt(int64(num)).Cmp(ipr.Len) >= 0 {
		m := ipr.ExtractAll()
		if m == nil {
			return nil
		}
		for i := 0; i < len(m); i++ {
			IPList = append(IPList, m[i])
		}
		// suffle
		myRand.Shuffle(len(IPList), func(i, j int) {
			IPList[i], IPList[j] = IPList[j], IPList[i]
		})
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

func printOneRow(x, y int, style tcell.Style, str string) {
	for _, c := range str {
		var comb []rune
		w := runewidth.RuneWidth(c)
		if w == 0 {
			comb = []rune{c}
			c = ' '
			w = 1
		}
		(*termAll).SetContent(x, y, c, comb, style)
		x += w
	}
	tx, _ := (*termAll).Size()
	if tx > len(str) {
		c := ' '
		for i := 0; i < tx-len(str); i++ {
			(*termAll).SetContent(x, y, c, nil, tcell.StyleDefault)
			x += 1
		}
	}
}

func initScreen() {
	defer func() { (*termAll).Sync() }()
	(*termAll).Clear()
	initTitleStr()
	printRuntimeWithoutSync()
	printCancelWithoutSync()
	printTitlePreWithoutSync()
	printTitleResultWithoutSync()
	updateScreen()
}

func printRuntimeWithoutSync() {
	printOneRow(0, titleRuntimeRow, titleStyle, *titleRuntime)
}

func printCancelWithoutSync() {
	printOneRow(0, titleCancelRow, titleStyleCancel, titleCancel)
}

func printCancel() {
	printCancelWithoutSync()
	(*termAll).Show()
}

func printCancelComfirm() {
	printOneRow(0, titleCancelRow, titleStyleCancel, titleCancelComfirm)
	(*termAll).Show()
}

func printQuitWaiting() {
	printOneRow(0, titleCancelRow, titleStyleCancel, titleWaitQuit)
	(*termAll).Show()
}

func printExitHint() {
	printOneRow(0, titleCancelRow, titleStyleCancel, titleExitHint)
	(*termAll).Show()
}

func printTitlePreWithoutSync() {
	printOneRow(0, titlePreRow, contentStyle, *titlePre)
}

func printTitleResultWithoutSync() {
	printOneRow(0, titleResultHintRow, titleStyle, titleResultHint)
	printOneRow(0, titleResultRow, titleStyle, *titleResult)
}

func printTasksStatWithoutSync() {
	printOneRow(0, titleTasksStatRow, contentStyle, *titleTasksStat)
}

func printResultListWithoutSync() {
	defer func() { (*termAll).Show() }()
	if *resultStrSlice == nil || len(*resultStrSlice) == 0 {
		return
	}
	t_len := len(*resultStrSlice)
	t_lowest := 0
	if t_len > maxResultsDisplay {
		t_lowest = t_len - maxResultsDisplay
	}
	for i := t_len - 1; i >= t_lowest; i-- {
		printOneRow(0, titleResultRow+t_len-i, contentStyle, (*resultStrSlice)[i])
	}
}

func printDebugListWithoutSync() {
	defer func() { (*termAll).Show() }()
	if debug {
		if *debugStrSlice == nil || len(*debugStrSlice) == 0 {
			return
		}
		printOneRow(0, titleDebugHintRow, contentStyle, titleDebugHint)
		printOneRow(0, titleDebugRow, contentStyle, *titleDebug)
		t_len := len(*debugStrSlice)
		t_low := 0
		if t_len > maxDebugDisplay {
			t_low = t_len - maxDebugDisplay
		}
		for i := t_len - 1; i >= t_low; i-- {
			printOneRow(0, titleDebugRow+t_len-i, contentStyle, (*debugStrSlice)[i])
		}
	}
}

func updateScreen() {
	defer func() { (*termAll).Show() }()
	printTasksStatWithoutSync()
	printResultListWithoutSync()
	printDebugListWithoutSync()
}

func initTitleStr() {
	var tIntroMSG string
	if dtOnly {
		tIntroMSG = fmt.Sprintf("Start Delay(%v) Test -- DelayMax:%v DTPassedRateMin:%v ResultMin:%v DTWorker:%v\n",
			dtSource, delayMax, dtPassedRateMin, resultMin, dtWorkerThread)
	} else if dltOnly {
		tIntroMSG = fmt.Sprintf("Start Speed Test -- SpeedMin(kB/s):%v ResultMin:%v DLTWorker:%v\n",
			speedMinimal, resultMin, dltWorkerThread)
	} else {
		tIntroMSG = fmt.Sprintf("Start Delay(%v) and Speed Test -- SpeedMin(kB/s):%v DelayMax:%v DTPassedRateMin:%v Result:%v DTWorker:%v DLTWorker:%v\n",
			dtSource, speedMinimal, delayMax, dtPassedRateMin, resultMin, dtWorkerThread, dltWorkerThread)
	}
	titlePre = &tIntroMSG
	tIntroMSG2 := fmt.Sprintf("%v - %v\n", runTIME, version)
	titleRuntime = &tIntroMSG2

	tIntroMSG3 := ""
	if !haveIPv6() {
		tIntroMSG3 = fmt.Sprintf("%-15v%s", "IP", myIndent)
	} else {
		tIntroMSG3 = fmt.Sprintf("%-39v%s", "IP", myIndent)
	}
	if !disableDownload {
		tIntroMSG3 += fmt.Sprintf("%-11v%s", "Speed(KB/s)", myIndent)
	}
	tIntroMSG3 += fmt.Sprintf("%-12v%s", "DelayAvg(ms)", myIndent)
	tIntroMSG3 += fmt.Sprintf("%-16v%s", "Stability(DTPR, %)", myIndent)
	titleResult = &tIntroMSG3
	updateTasksStatStr(0, 0, 0, 0, 0, 0, 0)
	if debug {
		titleDebug = &tIntroMSG3
	}
}

func updateTasksStatStr(allDTTasks, dtTasksDone, allDLTTasks, dltTasksDone, dtCached, dltCached, verifyResults int) {
	updateTasksStaticData(allDTTasks, dtTasksDone, allDLTTasks, dltTasksDone, dtCached, dltCached, verifyResults)
	var t = strings.Builder{}
	t.WriteString(getTimeNowStr())
	t.WriteString(myIndent)
	t.WriteString(fmt.Sprintf("Results:%-5d%s", taskStatistic.verifyResults, myIndent))
	if !dltOnly {
		t.WriteString(fmt.Sprintf("IPsCached(delay):%-5d%s", taskStatistic.dtCached+taskStatistic.allDTTasks-taskStatistic.dtTasksDoned, myIndent))
		t.WriteString(fmt.Sprintf("IPsTested(delay):%-5d%s", taskStatistic.dtTasksDoned, myIndent))
	}
	if !dtOnly {
		t.WriteString(fmt.Sprintf("IPsCached(speed):%-5d%s", taskStatistic.dltCached+taskStatistic.allDLTTasks-taskStatistic.dltTasksDoned, myIndent))
		t.WriteString(fmt.Sprintf("IPsTested(speed):%-5d%s", taskStatistic.dltTasksDoned, myIndent))
	}
	ts := t.String()
	titleTasksStat = &ts
}

//allPingTasks, pingTestedTasks,
//allDownloadTasks, downloadTestedTasks, len(pingCaches),
//len(downloadCaches), len(verifyResultsMap)
func updateTasksStaticData(allDTTasks, dtTasksDone, allDLTTasks, dltTasksDone, dtCached, dltCached, verifyResults int) {
	taskStatistic.allDTTasks = allDTTasks
	taskStatistic.dtTasksDoned = dtTasksDone
	taskStatistic.allDLTTasks = allDLTTasks
	taskStatistic.dltTasksDoned = dltTasksDone
	taskStatistic.dtCached = dtCached
	taskStatistic.dltCached = dltCached
	taskStatistic.verifyResults = verifyResults
}

func updateResultStrList(v VerifyResults) {
	t := *resultStrSlice
	if len(*resultStrSlice) > maxResultsDisplay {
		t = (*resultStrSlice)[(len(*resultStrSlice) - maxResultsDisplay):]
	}
	sb := strings.Builder{}
	if !haveIPv6() {
		sb.WriteString(fmt.Sprintf("%-15v%s", v.ip, myIndent))
	} else {
		sb.WriteString(fmt.Sprintf("%-39v%s", v.ip, myIndent))
	}
	if !disableDownload {
		sb.WriteString(fmt.Sprintf("%-11.2f%s", v.dls, myIndent))
	}
	sb.WriteString(fmt.Sprintf("%-12.0f%s", v.da, myIndent))
	sb.WriteString(fmt.Sprintf("%-16.2f%s", v.dtpr*100, myIndent))
	t = append(t, sb.String())
	resultStrSlice = &t
}

func updateDebugStrList(v VerifyResults) {
	if !debug {
		return
	}
	t := *debugStrSlice
	if len(*debugStrSlice) > maxDebugDisplay {
		t = (*debugStrSlice)[(len(*debugStrSlice) - maxDebugDisplay):]
	}
	sb := strings.Builder{}
	if !haveIPv6() {
		sb.WriteString(fmt.Sprintf("%-15v%s", v.ip, myIndent))
	} else {
		sb.WriteString(fmt.Sprintf("%-39v%s", v.ip, myIndent))
	}
	if !disableDownload {
		sb.WriteString(fmt.Sprintf("%-11.2f%s", v.dls, myIndent))
	}
	sb.WriteString(fmt.Sprintf("%-12.0f%s", v.da, myIndent))
	sb.WriteString(fmt.Sprintf("%-16.2f%s", v.dtpr*100, myIndent))
	t = append(t, sb.String())
	debugStrSlice = &t
}

func updateResult(v VerifyResults, allDTTasks, dtTasksDone, allDLTTasks, dltTasksDone, dtCached, dltCached, verifyResults int) {
	defer (*termAll).Show()
	updateTasksStat(allDTTasks, dtTasksDone, allDLTTasks, dltTasksDone, dtCached, dltCached, verifyResults)
	updateResultStrList(v)
	printResultListWithoutSync()
}

func updateDebug(v VerifyResults, allDTTasks, dtTasksDone, allDLTTasks, dltTasksDone, dtCached, dltCached, verifyResults int) {
	defer (*termAll).Show()
	updateTasksStat(allDTTasks, dtTasksDone, allDLTTasks, dltTasksDone, dtCached, dltCached, verifyResults)
	updateDebugStrList(v)
	printDebugListWithoutSync()
}

func updateTasksStat(allDTTasks, dtTasksDone, allDLTTasks, dltTasksDone, dtCached, dltCached, verifyResults int) {
	defer (*termAll).Show()
	updateTasksStatStr(allDTTasks, dtTasksDone, allDLTTasks, dltTasksDone, dtCached, dltCached, verifyResults)
	printTasksStatWithoutSync()
}

func haveIPv6() bool {
	for _, ipr := range srcIPRs {
		if ipr.IsV6() {
			return true
		}
	}
	return false
}
