package main

import (
	"bufio"
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
	logLevelDebug   = 1<<5 - 1
	logLevelInfo    = 1<<4 - 1
	logLevelWarning = 1<<3 - 1
	logLevelError   = 1<<2 - 1
	logLevelFatal   = 1<<1 - 1
	myIndent        = " "
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
		"104.16.0.0/12",
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
	}
	cfURL = "https://speed.cloudflare.com/__down"
)

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

type LogLevel int

type MyLogger struct {
	loggerLevel LogLevel
	indent      string
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

func (myLogger *MyLogger) getLogLevelString(lv LogLevel) string {
	switch lv {
	case logLevelDebug:
		return "DEBUG"
	case logLevelInfo:
		return "INFO"
	case logLevelWarning:
		return "WARNING"
	case logLevelError:
		return "ERROR"
	case logLevelFatal:
		return "FATAL"
	default:
	}
	switch myLogger.loggerLevel {
	case logLevelDebug:
		return "DEBUG"
	case logLevelInfo:
		return "INFO"
	case logLevelWarning:
		return "WARNING"
	case logLevelError:
		return "ERROR"
	case logLevelFatal:
		return "FATAL"
	default:
	}
	return "INFO"
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
	return MyLogger{lv, myIndent}
}

func (myLogger *MyLogger) log_newline(lv LogLevel, newline bool, info ...any) {
	fmt.Print(getTimeNowStr())
	fmt.Print(myLogger.indent)
	t_log_type_str := myLogger.getLogLevelString(lv)
	fmt.Printf("%v", t_log_type_str)
	fmt.Print(myLogger.indent)
	myLogger.print(newline, info...)
}

func (myLogger *MyLogger) log_newlinef(lv LogLevel, format string, info ...any) {
	fmt.Print(getTimeNowStr())
	fmt.Print(myLogger.indent)
	t_log_type_str := myLogger.getLogLevelString(lv)
	fmt.Printf("%v", t_log_type_str)
	fmt.Print(myLogger.indent)
	myLogger.printf(format, info...)
}

func (myLogger *MyLogger) print(newline bool, info ...any) {
	if len(info) >= 1 {
		fmt.Printf("%v", info[0])
		if len(info) > 1 {
			for _, t := range info[1:] {
				fmt.Printf("%s%v", myLogger.indent, t)
			}
		}
	}
	if newline {
		fmt.Println()
	}
}

func (myLogger *MyLogger) printf(format string, info ...any) {
	fmt.Printf(format, info...)
}

func (myLogger *MyLogger) debug(newline bool, info ...any) {
	myLogger.log_newline(logLevelDebug, newline, info...)
}

func (myLogger *MyLogger) debugf(format string, info ...any) {
	myLogger.log_newlinef(logLevelDebug, format, info...)
}

func (myLogger *MyLogger) info(newline bool, info ...any) {
	myLogger.log_newline(logLevelInfo, newline, info...)
}

func (myLogger *MyLogger) infof(format string, info ...any) {
	myLogger.log_newlinef(logLevelInfo, format, info...)
}

func (myLogger *MyLogger) warning(newline bool, info ...any) {
	myLogger.log_newline(logLevelWarning, newline, info...)
}

func (myLogger *MyLogger) warningf(format string, info ...any) {
	myLogger.log_newlinef(logLevelWarning, format, info...)
}

func (myLogger *MyLogger) error(newline bool, info ...any) {
	myLogger.log_newline(logLevelError, newline, info...)
}

func (myLogger *MyLogger) errorf(format string, info ...any) {
	myLogger.log_newlinef(logLevelError, format, info...)
}

func (myLogger *MyLogger) fatal(newline bool, info ...any) {
	myLogger.log_newline(logLevelFatal, newline, info...)
	os.Exit(1)
}

func (myLogger *MyLogger) fatalf(format string, info ...any) {
	myLogger.log_newlinef(logLevelFatal, format, info...)
	os.Exit(1)
}

func (myLogger *MyLogger) log(loglvl LogLevel, newline bool, info ...any) {
	switch loglvl {
	case logLevelDebug:
		myLogger.debug(newline, info...)
	case logLevelInfo:
		myLogger.info(newline, info...)
	case logLevelWarning:
		myLogger.warning(newline, info...)
	case logLevelError:
		myLogger.error(newline, info...)
	case logLevelFatal:
		myLogger.fatal(newline, info...)
	default:
	}
}

func (myLogger *MyLogger) logf(loglvl LogLevel, format string, info ...any) {
	switch loglvl {
	case logLevelDebug:
		myLogger.debugf(format, info...)
	case logLevelInfo:
		myLogger.infof(format, info...)
	case logLevelWarning:
		myLogger.warningf(format, info...)
	case logLevelError:
		myLogger.errorf(format, info...)
	case logLevelFatal:
		myLogger.fatalf(format, info...)
	default:
	}
}

func (myLogger *MyLogger) Debug(info ...any) {
	myLogger.debug(false, info...)
}

func (myLogger *MyLogger) Debugf(format string, info ...any) {
	myLogger.debugf(format, info...)
}

func (myLogger *MyLogger) Debugln(info ...any) {
	myLogger.debug(true, info...)
}

func (myLogger *MyLogger) Info(info ...any) {
	myLogger.info(false, info...)
}

func (myLogger *MyLogger) Infof(format string, info ...any) {
	myLogger.infof(format, info...)
}

func (myLogger *MyLogger) Infoln(info ...any) {
	myLogger.info(true, info...)
}

func (myLogger *MyLogger) Warning(info ...any) {
	myLogger.warning(false, info...)
}

func (myLogger *MyLogger) Warningf(format string, info ...any) {
	myLogger.warningf(format, info...)
}

func (myLogger *MyLogger) Warningln(info ...any) {
	myLogger.warning(true, info...)
}

func (myLogger *MyLogger) Error(info ...any) {
	myLogger.error(false, info...)
}

func (myLogger *MyLogger) Errorf(format string, info ...any) {
	myLogger.errorf(format, info...)
}

func (myLogger *MyLogger) Errorln(info ...any) {
	myLogger.error(true, info...)
}

func (myLogger *MyLogger) Fatal(info ...any) {
	myLogger.fatal(false, info...)
}

func (myLogger *MyLogger) Fatalf(format string, info ...any) {
	myLogger.fatalf(format, info...)
}

func (myLogger *MyLogger) Fatalln(info ...any) {
	myLogger.fatal(true, info...)
}

func (myLogger *MyLogger) Log(loglvl LogLevel, a ...any) {
	myLogger.log(loglvl, false, a...)
}

func (myLogger *MyLogger) Logf(loglvl LogLevel, format string, a ...any) {
	myLogger.logf(loglvl, format, a...)
}

func (myLogger *MyLogger) Logln(loglvl LogLevel, a ...any) {
	myLogger.log(loglvl, true, a...)
}

func (myLogger *MyLogger) Print(info ...any) {
	myLogger.print(false, info...)
}

func (myLogger *MyLogger) Printf(format string, info ...any) {
	myLogger.printf(format, info...)
}

func (myLogger *MyLogger) Println(info ...any) {
	myLogger.print(true, info...)
}

func (myLogger *MyLogger) PrintSingleStat(logLvl LogLevel, v []VerifyResults, ov overAllStat) {
	myLogger.PrintDetails(logLvl, v)
	myLogger.PrintOverAllStat(logLvl, ov)
}

// log when debug or info
func (myLogger *MyLogger) PrintDetails(logLvl LogLevel, v []VerifyResults) {
	// no data for print
	if len(v) == 0 {
		return
	}

	// print only when logLvl is permitted in myLogger
	if myLogger.loggerLevel&logLvl != logLvl {
		return
	}
	// fix indent
	if len(myLogger.indent) == 0 {
		myLogger.indent = myIndent
	}
	lc := v
	for i := 0; i < len(lc); i++ {
		myLogger.Logf(logLvl, "IP:%v%s", *lc[i].ip, myLogger.indent)
		if !dtOnly {
			myLogger.Printf("Speed(KB/s):%.2f%s", lc[i].dls, myLogger.indent)
		}
		myLogger.Printf("Delay(ms):%.0f", lc[i].da)
		if !dltOnly {
			myLogger.Printf("%sStab.(%%):%.2f", myLogger.indent, lc[i].dtpr*100)
		}
	}
	myLogger.Println()
}

// print just IPs
func (myLogger *MyLogger) PrintClearIPs(v []VerifyResults) {
	// no data for print
	if len(v) == 0 {
		return
	}
	lc := v
	for i := 0; i < len(lc); i++ {
		myLogger.Println(*lc[i].ip)
	}
}

// print OverAll statistic
func (myLogger *MyLogger) PrintOverAllStat(logLvl LogLevel, ov overAllStat) {
	// print only when logLvl is permitted in myLogger
	if myLogger.loggerLevel&logLvl != logLvl {
		return
	}
	// fix space
	if len(myLogger.indent) == 0 {
		myLogger.indent = myIndent
	}
	myLogger.Logf(logLvl, "Result:%d%s ", ov.resultCount, myLogger.indent)
	if !dltOnly {
		myLogger.Printf("DT - Tested:%d%s", ov.dtTasksDone, myLogger.indent)
		myLogger.Printf("OnGoing:%d%s", ov.dtOnGoing, myLogger.indent)
		myLogger.Printf("Cached:%d", ov.dtCached)
		// add more intent, while it's in both DT & DLT mode
		if !dtOnly {
			myLogger.Print("  ")
		}
	}
	if !dtOnly {
		myLogger.Printf("DLT - Tested:%d%s", ov.dltTasksDone, myLogger.indent)
		myLogger.Printf("OnGoing:%d%s", ov.dltOnGoing, myLogger.indent)
		myLogger.Printf("Cached:%d", ov.dltCached)
	}
	myLogger.Println()
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
			*tD.ip,
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
		urlStr = defaultDLTUrl
	}
	u, err := url.ParseRequestURI(urlStr)
	if err != nil || u == nil || len(u.Host) == 0 {
		myLogger.Fatal(fmt.Sprintf("url is not valid: %s\n", urlStr))
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

func NewUrl(urlStr, port string) string {
	urlStr = strings.TrimSpace(urlStr)
	if len(urlStr) == 0 {
		urlStr = defaultDLTUrl
	}
	u, err := url.ParseRequestURI(urlStr)
	if err != nil || u == nil || len(u.Host) == 0 {
		myLogger.Fatal(fmt.Sprintf("url is not valid: %s\n", urlStr))
	}
	host, old_port, err := net.SplitHostPort(u.Host)
	newHost := ""
	if err != nil {
		newHost = net.JoinHostPort(u.Host, port)
	} else {
		if old_port == port {
			return urlStr
		}
		newHost = net.JoinHostPort(host, port)
	}
	u.Host = newHost
	return u.String()
}

func initRandSeed() {
	myRand.Seed(time.Now().UnixNano())
}

func getASNAndCityWithIP(ipStr *string) (ASN int, city string) {
	if defaultASN > 0 || len(defaultCity) > 0 {
		ASN = defaultASN
		city = defaultCity
		return
	}
	// try 3 times
	for i := 0; i < 3; i++ {
		tReq, err := http.NewRequest("GET", cfURL, nil)
		if err != nil {
			log.Fatal(err)
		}
		var client = http.Client{
			Transport:     nil,
			CheckRedirect: nil,
			Jar:           nil,
			Timeout:       httpRspTimeoutDuration + 1*time.Second,
		}
		if len(*ipStr) > 0 && isValidIPs(*ipStr) {
			fullAddress := genHostFromIPStrPort(*ipStr, 443)
			client.Transport = &http.Transport{
				DialContext: GetDialContextByAddr(fullAddress),
				//ResponseHeaderTimeout: HttpRspTimeoutDuration,
			}
		}
		response, err := client.Do(tReq)
		// connection is failed(network error), won't continue
		if err != nil || response == nil {
			myLogger.Error(fmt.Sprintf("An error occurred while request ASN and city info from cloudflare: %v\n", err))
			time.Sleep(time.Duration(interval) * time.Millisecond)
			continue
		}
		if values, ok := (*response).Header["Cf-Meta-Asn"]; ok {
			if len(values) > 0 {
				if ASN, err = strconv.Atoi(values[0]); err != nil {
					myLogger.Error(fmt.Sprintf("An error occurred while convert ASN for header: %T, %v", values[0], values[0]))
				}
			}
		}
		if values, ok := (*response).Header["Cf-Meta-City"]; ok {
			if len(values) > 0 {
				city = values[0]
			}
		}
		if len(city) == 0 { // no "Cf-Meta-City" in header, we get "Cf-Meta-Country" instead
			if values, ok := (*response).Header["Cf-Meta-Country"]; ok {
				if len(values) > 0 {
					city = values[0]
				}
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
		tIP := net.ParseIP(*v[i].ip)
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
			fmt.Printf("%-15v%s", *v[i].ip, myLogger.indent)
		} else {
			fmt.Printf("%-39v%s", *v[i].ip, myLogger.indent)
		}
		if !disableDownload {
			fmt.Printf("%-11.2f%s", v[i].dls, myLogger.indent)
		}
		fmt.Printf("%-12.0f%s", v[i].da, myLogger.indent)
		fmt.Printf("%-12.2f%s", v[i].dtpr*100, myLogger.indent)
		// close line, LatestLogLength should be 0
		fmt.Println()
	}
	fmt.Println()
}

func InsertIntoDb(verifyResultsSlice []VerifyResults, dbFile string) {
	if len(verifyResultsSlice) > 0 && storeToDB {
		//get ASN and city
		tRound := MinInt(3, len(verifyResultsSlice))
		dbRecords := make([]cfTestDetail, 0)
		var ASN int
		var city string
		for _, item := range verifyResultsSlice[:tRound] {
			ASN, city = getASNAndCityWithIP(item.ip)
			if ASN > 0 || len(city) > 0 {
				break
			}
		}
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
	var tVerifyResult = VerifyResults{}
	tVerifyResult.testTime = out.testTime
	tIP := out.host
	tVerifyResult.ip = &tIP
	if len(out.resultSlice) == 0 {
		return tVerifyResult
	}
	tVerifyResult.dtc = len(out.resultSlice)
	var tDurationsAll = 0.0
	var tDownloadDurations float64
	var tDownloadSizes int64
	for _, v := range out.resultSlice {
		if v.dTPassed {
			tVerifyResult.dtpc += 1
			tVerifyResult.dltc += 1
			tDuration := float64(v.dTDuration) / float64(time.Millisecond)
			// if pingViaHttps, it should add the http duration
			if dtHttps {
				tDuration = +float64(v.httpReqRspDur) / float64(time.Millisecond)
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

func NewIPRangeFromString(StartIPStr *string, EndIPStr *string) *ipRange {
	StartIP := net.ParseIP(*StartIPStr)
	EndIP := net.ParseIP(*EndIPStr)
	return new(ipRange).init(StartIP, EndIP)
}

func NewIPRangeFromCIDR(cidr *string) *ipRange {
	ip, ipCIDR, err := net.ParseCIDR(*cidr)
	// there an error during parsing process, or its an ip
	if err != nil {
		tIP := net.ParseIP(*cidr)
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
	// beyond the boundary
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
	printRuntimeWithoutSync()
	printTitlePreWithoutSync()
	printCancelWithoutSync()
	updateScreen()
}

func printRuntimeWithoutSync() {
	printOneRow(0, titleRuntimeRow, contentStyle, *titleRuntime)
}

func printTitlePreWithoutSync() {
	var len0, len1, len2 int
	len0 = MaxInt(len(titlePre[0][0]), len(titlePre[1][0]))
	len1 = MaxInt(len(titlePre[0][1]), len(titlePre[1][1]))
	len2 = MaxInt(len(titlePre[0][2]), len(titlePre[1][2]))
	rowStr0 := fmt.Sprintf("%*v%-*v  %*v%v", len0, titlePre[0][0], len1, titlePre[0][1], len2, titlePre[0][2], titlePre[0][3])
	printOneRow(0, titlePreRow, contentStyle, rowStr0)
	if len(titlePre[1][0]) > 0 && len(titlePre[1][1]) > 0 {
		rowStr1 := fmt.Sprintf("%*v%-*v", len0, titlePre[1][0], len1, titlePre[1][1])
		if len(titlePre[1][2]) > 0 && len(titlePre[1][3]) > 0 {
			rowStr1 += fmt.Sprintf("  %*v%v", len2, titlePre[1][2], titlePre[1][3])
		}
		printOneRow(0, titlePreRow+1, contentStyle, rowStr1)
	}
}

func printCancelWithoutSync() {
	printOneRow(0, titleCancelRow, titleStyleCancel, titleCancel)
}

func printCancelConfirmWithoutSync() {
	printOneRow(0, titleCancelRow, titleStyleCancel, titleCancelConfirm)
}

func printQuitWaitingWithoutSync() {
	printOneRow(0, titleCancelRow, titleStyleCancel, titleWaitQuit)
}

func printCancel() {
	printCancelWithoutSync()
	(*termAll).Show()
}

func printCancelConfirm() {
	printCancelConfirmWithoutSync()
	(*termAll).Show()
}

func printQuitWaiting() {
	printQuitWaitingWithoutSync()
	(*termAll).Show()
}

func printQuittingCountDown(sec int) {
	for i := sec; i > 0; i-- {
		printOneRow(0, titleCancelRow, titleStyleCancel, fmt.Sprintf("Exit in %ds...", i))
		(*termAll).Show()
		time.Sleep(time.Second)
	}
}

func printTitleResultHintWithoutSync() {
	printOneRow(0, titleResultHintRow, titleStyle, fmt.Sprintf("%s%v", titleResultHint, len(verifyResultsMap)))
}

func printTitleDebugHintWithoutSync() {
	if !debug {
		return
	}
	printOneRow(0, titleDebugHintRow, contentStyle, titleDebugHint)
}

func printTaskStatWithoutSync() {
	if !dltOnly {
		printOneRow(0, titleTasksStatRow, contentStyle, *titleTasksStat[0])
		if !dtOnly {
			// start from row 23
			printOneRow(resultStatIndent, titleTasksStatRow+1, contentStyle, *titleTasksStat[1])
		}
	} else {
		printOneRow(0, titleTasksStatRow, contentStyle, *titleTasksStat[0])
	}
}

func printDetailsListWithoutSync(details [][]*string, startRow int, maxRowsDisplayed int) {
	if len(details) == 0 {
		return
	}
	t_len := len(details)
	t_lowest := 0
	if t_len > maxRowsDisplayed {
		t_lowest = t_len - maxRowsDisplayed
	}
	// scan for indent
	t_indent_slice := make([]int, 0)
	for _, v := range detailTitleSlice {
		t_indent_slice = append(t_indent_slice, len(v))
	}
	for i := t_lowest; i < t_len; i++ {
		for j := 0; j < len(t_indent_slice); j++ {
			t_indent_slice[j] = MaxInt(t_indent_slice[j], len(*details[i][j]))
		}
	}
	// print title
	t_sb := strings.Builder{}
	for j := 0; j < len(t_indent_slice)-1; j++ {
		t_sb.WriteString(fmt.Sprintf("%-*s%s", t_indent_slice[j], detailTitleSlice[j], myIndent))
	}
	t_sb.WriteString(fmt.Sprintf("%v", detailTitleSlice[len(t_indent_slice)-1]))
	printOneRow(0, startRow, contentStyle, t_sb.String())
	// print list
	for i := t_len - 1; i >= t_lowest; i-- {
		t_sb.Reset()
		for j := 0; j < len(t_indent_slice)-1; j++ {
			t_sb.WriteString(fmt.Sprintf("%-*s%s", t_indent_slice[j], *details[i][j], myIndent))
		}
		t_sb.WriteString(*details[i][len(t_indent_slice)-1])
		printOneRow(0, startRow+t_len-i, contentStyle, t_sb.String())
	}
}

func printResultListWithoutSync() {
	printTitleResultHintWithoutSync()
	printDetailsListWithoutSync(resultStrSlice, titleResultRow, maxResultsDisplay)
}

func printDebugListWithoutSync() {
	if !debug {
		return
	}
	printTitleDebugHintWithoutSync()
	printDetailsListWithoutSync(debugStrSlice, titleDebugRow, maxDebugDisplay)
}

func updateScreen() {
	defer func() { (*termAll).Show() }()
	printTaskStatWithoutSync()
	printResultListWithoutSync()
	printDebugListWithoutSync()
}

func initTitleStr() {
	var tMsgRuntime string
	tMsgRuntime = fmt.Sprintf("%v %v - ", runTime, version)

	if !dltOnly {
		tMsgRuntime += fmt.Sprintf("Start Delay (%v) Test (DT)", dtSource)
		if !dtOnly {
			tMsgRuntime += " and "
		}
	}
	if !dtOnly {
		tMsgRuntime += "Speed Test (DLT)"
	}
	titleRuntime = &tMsgRuntime
	titlePre[0][0] = "Result Exp.:"
	// we just control the display "resultMin" in main.init()
	titlePre[0][1] = " " + strconv.Itoa(resultMin)
	// if !testAll {
	// 	titlePre[0][1] = " " + strconv.Itoa(resultMin)
	// } else {
	// 	titlePre[0][1] = " ~"
	// }
	if dtOnly {
		titlePre[0][2] = "Max Delay:"
		titlePre[0][3] = fmt.Sprintf(" %vms", dtEvaluationDelay)
		titlePre[1][0] = "Min Stab.:"
		titlePre[1][1] = fmt.Sprintf(" %v", dtEvaluationDTPR) + "%"
	} else if dltOnly {
		titlePre[0][2] = "Min Speed:"
		titlePre[0][3] = fmt.Sprintf(" %vKB/s", dltEvaluationSpeed)
	} else {
		titlePre[0][2] = "Min Speed:"
		titlePre[0][3] = fmt.Sprintf(" %vKB/s", dltEvaluationSpeed)
		titlePre[1][0] = "Max Delay:"
		titlePre[1][1] = fmt.Sprintf(" %vms", dtEvaluationDelay)
		titlePre[1][2] = "Min Stab.:"
		titlePre[1][3] = fmt.Sprintf(" %v", dtEvaluationDTPR) + "%"
	}
	detailTitleSlice = append(detailTitleSlice, "IP")
	if !dtOnly {
		detailTitleSlice = append(detailTitleSlice, "Speed(KB/s)")
	}
	detailTitleSlice = append(detailTitleSlice, "Delay(ms)")
	if !dltOnly {
		detailTitleSlice = append(detailTitleSlice, "Stab.(%)")
	}
	updateTaskStatStr(overAllStat{0, 0, 0, 0, 0, 0, 0})
}

func updateTaskStatStr(ov overAllStat) {
	var t = strings.Builder{}
	var t1 = strings.Builder{}
	t.WriteString(getTimeNowStr())
	t.WriteString(myIndent)
	// t.WriteString(fmt.Sprintf("Result:%-*d%s", resultNumLen, resultCount, myIndent))
	t_dtCachedS := ov.dtCached
	t_dltCachedS := ov.dltCached
	t_dtCachedSNumLen := len(strconv.Itoa(t_dtCachedS))
	t_dltCachedSNumLen := len(strconv.Itoa(t_dltCachedS))
	t_dtDoneNumLen := len(strconv.Itoa(ov.dtTasksDone))
	t_dltDoneNumLen := len(strconv.Itoa(ov.dltTasksDone))
	var t_indent = 0
	if !dltOnly {
		t_indent = MaxInt(dtThreadsNumLen, t_dtCachedSNumLen, t_dltCachedSNumLen, t_dtDoneNumLen, t_dltDoneNumLen)
	}
	if !dtOnly {
		t_indent = MaxInt(t_indent, dltThreadsNumLen, t_dtCachedSNumLen, t_dltCachedSNumLen, t_dtDoneNumLen, t_dltDoneNumLen)
	}
	if !dltOnly {
		if dtOnly {
			t.WriteString(fmt.Sprintf("DT - Tested:%-*d%s", t_indent, ov.dtTasksDone, myIndent))

		} else {
			t.WriteString(fmt.Sprintf("DT  - Tested:%-*d%s", t_indent, ov.dtTasksDone, myIndent))
			t1.WriteString(fmt.Sprintf("DLT - Tested:%-*d%s", t_indent, ov.dltTasksDone, myIndent))
			t1.WriteString(fmt.Sprintf("OnGoing:%-*d%s", t_indent, ov.dltOnGoing, myIndent))
			t1.WriteString(fmt.Sprintf("Cached:%-*d%s", t_indent, t_dltCachedS, myIndent))
			ts1 := t1.String()
			titleTasksStat[1] = &ts1
		}
		t.WriteString(fmt.Sprintf("OnGoing:%-*d%s", t_indent, ov.dtOnGoing, myIndent))
		t.WriteString(fmt.Sprintf("Cached:%-*d%s", t_indent, t_dtCachedS, myIndent))
		ts := t.String()
		titleTasksStat[0] = &ts
	} else {
		t.WriteString(fmt.Sprintf("DLT - Tested:%-*d%s", t_indent, ov.dltTasksDone, myIndent))
		t.WriteString(fmt.Sprintf("OnGoing:%-*d%s", t_indent, ov.dltOnGoing, myIndent))
		t.WriteString(fmt.Sprintf("Cached:%-*d%s", t_indent, t_dltCachedS, myIndent))
		ts := t.String()
		titleTasksStat[0] = &ts
	}
}

func updateDetailList(src [][]*string, v []VerifyResults, limit int) (dst [][]*string) {
	dst = src
	for _, tv := range v {
		t_str_list := make([]*string, 0)
		// t_v1 := fmt.Sprintf("%v", *tv.ip)
		// t_str_list = append(t_str_list, &t_v1)
		t_str_list = append(t_str_list, tv.ip)
		// show speed only when it performed DLT
		if !dtOnly {
			t_v2 := fmt.Sprintf("%.2f", tv.dls)
			t_str_list = append(t_str_list, &t_v2)
		}
		t_v3 := fmt.Sprintf("%.0f", tv.da)
		t_str_list = append(t_str_list, &t_v3)
		// show DTPR only when it performed DT
		if !dltOnly {
			t_v4 := fmt.Sprintf("%.2f", tv.dtpr*100)
			t_str_list = append(t_str_list, &t_v4)
		}
		dst = append(dst, t_str_list)
	}
	if len(dst) > limit {
		dst = dst[(len(dst) - limit):]
	}
	return
}

func updateResultStrList(v []VerifyResults) {
	resultStrSlice = updateDetailList(resultStrSlice, v, maxResultsDisplay)
}

func updateDebugStrList(v []VerifyResults) {
	if !debug {
		return
	}
	debugStrSlice = updateDetailList(debugStrSlice, v, maxDebugDisplay)
}

func updateResult(v []VerifyResults) {
	defer (*termAll).Show()
	updateResultStrList(v)
	printResultListWithoutSync()
}

func updateDebug(v []VerifyResults) {
	if !debug {
		return
	}
	defer (*termAll).Show()
	updateDebugStrList(v)
	printDebugListWithoutSync()
}

func updateTaskStat(ov overAllStat) {
	defer (*termAll).Show()
	updateTaskStatStr(ov)
	printTaskStatWithoutSync()
}

func updateTcellDetails(isResult bool, v []VerifyResults) {
	// prevent display debug msg when in not-debug mode
	if !debug {
		return
	}
	if isResult { // result
		updateResult(v)
	} else { // non-debug
		updateDebug(v)
	}
}

// test detail
// loglvl should be logLevelDebug or logLevelInfo
// when: 1. in non-debug mode, just print stats instead of pure qualified IPs.
//  2. in debug mode, we show more as tcell or non-tcell form.
//
// isResult: used for tcell mode, indicate show in result area or debug area.
func displayDetails(isResult bool, v []VerifyResults) {
	if !debug {
		// myLogger.PrintClearIPs(v)
		myLogger.PrintDetails(LogLevel(logLevelInfo), v)
	} else {
		if !tcellMode { // no-tcell
			myLogger.PrintDetails(LogLevel(logLevelDebug), v)
		} else { // tcell
			updateTcellDetails(isResult, v)
		}
	}
}

// task statistic - only work in debug mode both in tcell and non-tcell mode
func displayStat(ov overAllStat) {
	if !tcellMode { // no-tcell
		myLogger.PrintOverAllStat(logLevelDebug, ov)
	} else { // tcell, print always
		updateTaskStat(ov)
	}
}

func HaveIPv6() bool {
	for _, ipr := range srcIPRsRaw {
		if ipr.IsV6() {
			return true
		}
	}
	return false
}

func MaxInt(a, b int, num ...int) (t int) {
	t = a
	if b > t {
		t = b
	}
	for _, i := range num {
		if i > t {
			t = i
		}
	}
	return
}

func MinInt(a, b int, num ...int) (t int) {
	t = a
	if b < t {
		t = b
	}
	for _, i := range num {
		if i < t {
			t = i
		}
	}
	return
}

// we get target IPs based on <amount>. We will get amount of <amount> from every IPR in srcIPR and  from srcIPRsCache
func retrieveIPsFromIPR(amount int) (targetIPs []*string) {
	if amount < 0 || amount < retrieveCount {
		amount = retrieveCount
	}

	t_ips := []net.IP{}
	for _, ipr := range srcIPRsRaw {
		if !testAll {
			t_ips = append(t_ips, ipr.GetRandomX(amount)...)
		} else {
			t_ips = append(t_ips, ipr.Extract(amount)...)
		}
	}
	if len(srcIPRsExtracted) > 0 {
		if len(srcIPRsExtracted) <= amount {
			t_ips = append(t_ips, srcIPRsExtracted...)
			srcIPRsExtracted = []net.IP{}
		} else {
			t_ips = append(t_ips, srcIPRsExtracted[0:amount]...)
			srcIPRsExtracted = srcIPRsExtracted[amount:]
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

func retrieveHosts(amount int) (targetHosts []*string) {
	if amount <= 0 || len(srcHosts) == 0 {
		return
	}
	t_amount := amount
	if len(srcHosts) < amount {
		t_amount = len(srcHosts)
	}
	targetHosts = append(targetHosts, srcHosts[:t_amount]...)
	if len(srcHosts) <= amount {
		srcHosts = []*string{}
	} else {
		srcHosts = srcHosts[t_amount:]
	}
	return
}

func isValidCIDR(ips string) bool {
	_, _, err := net.ParseCIDR(ips)
	return err == nil
}

func isValidIP(ip string) bool {
	tIP := net.ParseIP(ip)
	return tIP != nil
}

func isValidIPs(ips string) bool {
	ips = strings.TrimSpace(ips)
	if isValidCIDR(ips) {
		return true
	} else {
		return isValidIP(ips)
	}
}

func isValidHost(host string) bool {
	ok, _, _ := splitHost(host)
	return ok
}

func splitHost(host string) (bool, string, int) {
	host = strings.TrimSpace(host)
	if len(host) == 0 {
		return false, "", -1
	}
	ip, port, err := net.SplitHostPort(host)
	if err != nil {
		return false, "", -1
	}
	// invalid ip in host
	if !isValidIPs(ip) {
		return false, "", -1
	}
	// invalid port
	t_port, err := strconv.Atoi(port)
	if err != nil {
		return false, "", -1
	}
	if t_port <= 0 || t_port > 65535 {
		return false, "", -1
	}
	return true, ip, t_port
}

// func genHostFromIPPort(ip net.IP, port int) (connStr string) {
// 	connStr = genHostFromIPStrPort(ip.String(), port)
// 	return
// }

func genHostFromIPStrPort(ipStr string, port int) (connStr string) {
	if !isValidIPs(ipStr) {
		return
	}
	if port < 1 || port > 65535 {
		return
	}
	connStr = net.JoinHostPort(ipStr, strconv.Itoa(port))
	return
}

func confirm(s string, tries int) bool {
	reader := bufio.NewReader(os.Stdin)
	for ; tries > 0; tries-- {
		fmt.Printf("%s [y/yes/N/no]: ", s)
		rsp, err := reader.ReadString('\n')
		if err != nil {
			myLogger.Fatal(err)
		}
		rsp = strings.ToLower(strings.TrimSpace(rsp))
		if rsp == "y" || rsp == "yes" {
			return true
		} else if rsp == "n" || rsp == "no" || len(rsp) == 0 {
			return false
		} else {
			continue
		}
	}
	return false
}
