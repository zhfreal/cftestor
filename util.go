package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

func getTimeNowStr() string {
	return time.Now().Format("15:04:05")
}

func getTimeNowStrSuffix() string {
	s := time.Now().Format("20060102150405")
	return strings.ReplaceAll(s, ".", "")
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func parseUrl(urlStr string) (tHostName string, tPort int) {
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

func newUrl(urlStr, port string) string {
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

// from https://api.incolumitas.com/?q=3.5.140.2
func getGeoInfoFromIncolumitas(ipStr string) (ASN int, city, country string) {
	t_url, _ := url.Parse("https://api.incolumitas.com/")
	params := url.Values{}
	if len(ipStr) > 0 && isValidIPs(ipStr) {
		params.Add("q", ipStr)
	}
	t_url.RawQuery = params.Encode()
	tReq, err := http.NewRequest("GET", t_url.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
	var client = http.Client{
		Transport:     nil,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       5 * time.Second,
	}

	response, err := client.Do(tReq)
	// connection is failed(network error), won't continue
	if err != nil || response == nil {
		myLogger.Error(fmt.Sprintf("An error occurred while request ASN and city info from cloudflare: %v\n", err))
		time.Sleep(time.Duration(Config.Interval) * time.Millisecond)
		return
	}
	// read response.Body as string
	body, err := io.ReadAll(response.Body)
	if err != nil {
		myLogger.Error(fmt.Sprintf("An error occurred while read response.Body: %v\n", err))
		return
	}
	// decode body []byte into string
	bodyStr := string(body)
	asn := gjson.Get(bodyStr, "asn.asn")
	if asn.Exists() {
		if ASN, err = strconv.Atoi(asn.String()); err != nil {
			myLogger.Error(fmt.Sprintf("An error occurred while convert ASN for header: %T, %v", asn.String(), asn.String()))
		}
	}
	city = gjson.Get(bodyStr, "location.city").String()
	country = gjson.Get(bodyStr, "location.country_code").String()
	return
}

func GetDialContextByAddr(addrPort string) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		c, e := (&net.Dialer{}).DialContext(ctx, network, addrPort)
		return c, e
	}
}

// calcResult will calculate the statistic results of DT and DLT for a IP
func calcResult(out singleVerifyResult, statDownload bool) VerifyResults {
	// initialize VerifyResults
	var tVerifyResult = VerifyResults{}
	tVerifyResult.dtDList = make([]float64, 0)
	tVerifyResult.testTime = out.testTime
	tIP := out.host
	tVerifyResult.ip = &tIP
	tVerifyResult.loc = &out.loc
	if len(out.resultSlice) == 0 {
		return tVerifyResult
	}
	tVerifyResult.dtc = len(out.resultSlice)
	var tDurationsAll = 0.0
	for _, v := range out.resultSlice {
		if v.dTPassed {
			tVerifyResult.dtpc += 1
			tDuration := float64(v.dTDuration) / float64(time.Millisecond)
			// if pingViaHttps, it should add the http duration
			if Config.DTHttps {
				tDuration += float64(v.httpReqRspDur) / float64(time.Millisecond)
			}
			tVerifyResult.dtDList = append(tVerifyResult.dtDList, tDuration)
			tDurationsAll += tDuration
			if tDuration > tVerifyResult.dmx {
				tVerifyResult.dmx = tDuration
			}
			if tVerifyResult.dmi <= 0.0 || tDuration < tVerifyResult.dmi {
				tVerifyResult.dmi = tDuration
			}
			if statDownload {
				tVerifyResult.dltc += 1
				if v.dLTWasDone && v.dLTPassed {
					tVerifyResult.dltpc += 1
					tVerifyResult.dltd += float64(v.dLTDuration) / float64(time.Second)
					tVerifyResult.dlds += v.dLTDataSize
				}
			}
		}
	}
	if tVerifyResult.dtpc > 0 {
		tVerifyResult.da = tDurationsAll / float64(tVerifyResult.dtpc)
		tVerifyResult.dtpr = float64(tVerifyResult.dtpc) / float64(tVerifyResult.dtc)
		// just calculate variance and standard deviation when we ev-dt enabled
		if Config.EnableStdEv {
			tVerifyResult.daVar = variance(tVerifyResult.dtDList)
			tVerifyResult.daStd = std(tVerifyResult.dtDList)
		}
	}
	// we statistic download speed while the downloaded file size is greater than DownloadSizeMin
	if statDownload {
		if tVerifyResult.dltpc > 0 && tVerifyResult.dlds > downloadSizeMin {
			tVerifyResult.dltpr = float64(tVerifyResult.dltpc) / float64(tVerifyResult.dltc)
			tVerifyResult.dls = float64(tVerifyResult.dlds) / tVerifyResult.dltd / 1000
		}
	}
	return tVerifyResult
}

func EraseLine(n int) {
	if n <= 0 {
		return
	}
	fmt.Printf("%s", strings.Repeat("\b \b", n))
}

// displayDetailsNew displays the details of the given VerifyResults slice.
// It will print the details to the console in Config.Debug mode, and only print the
// IPs in non-Config.Debug mode. The showSpeed parameter determines whether the speed
// information should be shown or not.
func displayDetails(showSpeed, loopEnabled bool, v []VerifyResults) {
	// if in Config.Debug mode, print the details
	if Config.Debug {
		myLogger.PrintDetails(LogLevel(logLevelDebug), v, showSpeed)
	} else {
		// if in non-Config.Debug mode, only print the IPs
		if Config.SilenceMode {
			if !loopEnabled {
				for _, t_v := range v {
					tStr := *t_v.ip
					if t_v.loc != nil && len(*t_v.loc) > 0 {
						tStr = fmt.Sprintf("%s#%s", tStr, *t_v.loc)
					}
					myLogger.Println(tStr)
				}
			}
		} else {
			// print the details in non-Config.Debug mode
			myLogger.PrintDetails(LogLevel(logLevelInfo), v, showSpeed)
		}
	}
}

// task statistic - only work in Config.Debug mode
func displayStat(ov overAllStat) {
	myLogger.PrintOverAllStat(logLevelDebug, ov)
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

func getHostVer(host string) (ver int8) {
	tIP, _, err := net.SplitHostPort(host)
	if err != nil {
		return TypeIPErr
	}
	return getIPVer(tIP)
}

func getIPsVer(ips string) (ver int8) {
	tV := getCIDRVer(ips)
	if tV == TypeIPErr {
		return getIPVer(ips)
	} else {
		return tV
	}
}

func getCIDRVer(ips string) (ver int8) {
	_, _, err := net.ParseCIDR(ips)
	if err != nil {
		return TypeIPErr
	}
	if strings.Contains(ips, ":") {
		return TypeIPv6
	}
	return TypeIPv4
}

func getIPVer(ips string) (ver int8) {
	tIP := net.ParseIP(ips)
	if tIP == nil {
		return TypeIPErr
	} else if tIP.To4() != nil {
		return TypeIPv4
	} else {
		return TypeIPv6
	}
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

func genHostFromIPStrPort(ip string, port int) (connStr string) {
	if !isValidIPs(ip) {
		return ""
	}
	if port < 1 || port > 65535 {
		return ""
	}
	connStr = net.JoinHostPort(ip, strconv.Itoa(port))
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


func newRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}
