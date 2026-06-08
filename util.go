package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
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

func parseUrl(urlStr string) (tHostName string, tPort int, err error) {
	urlStr = strings.TrimSpace(urlStr)
	if len(urlStr) == 0 {
		urlStr = defaultDLTUrl
	}
	u, err := url.ParseRequestURI(urlStr)
	if err != nil || u == nil || len(u.Host) == 0 {
		return "", 0, fmt.Errorf("invalid URL %q", urlStr)
	}
	tHostName = u.Hostname()
	if len(tHostName) == 0 {
		return "", 0, fmt.Errorf("invalid URL %q: missing host", urlStr)
	}
	if port := u.Port(); len(port) > 0 {
		tPort, err = strconv.Atoi(port)
		if err != nil || tPort <= 0 || tPort > 65535 {
			return "", 0, fmt.Errorf("invalid URL %q: invalid port %q", urlStr, port)
		}
		return tHostName, tPort, nil
	}
	switch u.Scheme {
	case "http":
		tPort = 80
	case "https":
		tPort = 443
	default:
		return "", 0, fmt.Errorf("invalid URL %q: scheme must be http or https when no port is provided", urlStr)
	}
	return tHostName, tPort, nil
}

func newUrl(urlStr, port string) (string, error) {
	urlStr = strings.TrimSpace(urlStr)
	if len(urlStr) == 0 {
		urlStr = defaultDLTUrl
	}
	newPort, err := strconv.Atoi(port)
	if err != nil || newPort <= 0 || newPort > 65535 {
		return "", fmt.Errorf("invalid target port %q", port)
	}
	u, err := url.ParseRequestURI(urlStr)
	if err != nil || u == nil || len(u.Host) == 0 {
		return "", fmt.Errorf("invalid URL %q", urlStr)
	}
	host := u.Hostname()
	if len(host) == 0 {
		return "", fmt.Errorf("invalid URL %q: missing host", urlStr)
	}
	if u.Port() == port {
		return u.String(), nil
	}
	u.Host = net.JoinHostPort(host, port)
	return u.String(), nil
}

func shouldApplyNoCache(sourceURL string) bool {
	return Config.NoCache && !isDefaultTestURL(sourceURL)
}

func isDefaultTestURL(sourceURL string) bool {
	return equivalentURL(sourceURL, defaultDTUrl) || equivalentURL(sourceURL, defaultDLTUrl)
}

func equivalentURL(a, b string) bool {
	aURL, err := url.Parse(strings.TrimSpace(a))
	if err != nil || aURL == nil {
		return false
	}
	bURL, err := url.Parse(strings.TrimSpace(b))
	if err != nil || bURL == nil {
		return false
	}
	return strings.EqualFold(aURL.Scheme, bURL.Scheme) &&
		strings.EqualFold(strings.TrimSuffix(aURL.Hostname(), "."), strings.TrimSuffix(bURL.Hostname(), ".")) &&
		effectiveURLPort(aURL) == effectiveURLPort(bURL) &&
		normalizedURLPath(aURL) == normalizedURLPath(bURL) &&
		aURL.Query().Encode() == bURL.Query().Encode()
}

func effectiveURLPort(u *url.URL) string {
	if port := u.Port(); len(port) > 0 {
		return port
	}
	switch strings.ToLower(u.Scheme) {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}

func normalizedURLPath(u *url.URL) string {
	if len(u.EscapedPath()) == 0 {
		return "/"
	}
	return u.EscapedPath()
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
		myLogger.Errorf("failed to create Incolumitas request: %v\n", err)
		return
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
		myLogger.Errorf("failed to request ASN and city info from Incolumitas: %v\n", err)
		time.Sleep(time.Duration(Config.Interval) * time.Millisecond)
		return
	}
	defer response.Body.Close()
	// read response.Body as string
	body, err := io.ReadAll(response.Body)
	if err != nil {
		myLogger.Errorf("failed to read Incolumitas response body: %v\n", err)
		return
	}
	// decode body []byte into string
	bodyStr := string(body)
	asn := gjson.Get(bodyStr, "asn.asn")
	if asn.Exists() {
		if ASN, err = strconv.Atoi(asn.String()); err != nil {
			myLogger.Errorf("failed to parse ASN from Incolumitas response: %T, %v\n", asn.String(), asn.String())
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

// calcResult calculates the DT and DLT statistics for one candidate.
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
	tHost, _, err := net.SplitHostPort(host)
	if err != nil {
		return TypeIPErr
	}
	tVer := getIPVer(tHost)
	if tVer != TypeIPErr {
		return tVer
	}
	if isValidDNSHost(tHost) {
		return TypeIPv4 | TypeIPv6
	}
	return TypeIPErr
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
	tHost, port, err := net.SplitHostPort(host)
	if err != nil {
		return false, "", -1
	}
	// invalid host name or IP in host
	if !isValidIPs(tHost) && !isValidDNSHost(tHost) {
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
	return true, tHost, t_port
}

func isValidDNSHost(host string) bool {
	host = strings.TrimSpace(strings.TrimSuffix(host, "."))
	if len(host) == 0 || len(host) > 253 {
		return false
	}
	if looksLikeIPv4Literal(host) {
		return false
	}
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
				continue
			}
			return false
		}
	}
	return true
}

func looksLikeIPv4Literal(host string) bool {
	parts := strings.Split(host, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if len(part) == 0 {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
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
			myLogger.Errorf("failed to read confirmation: %v\n", err)
			return false
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
