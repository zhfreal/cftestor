package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"text/tabwriter"

	"github.com/tidwall/gjson"
)

func getTimeNowStr() string {
	return time.Now().Format("15:04:05")
}

func getTimeNowStrSuffix() string {
	//s := time.Now().Format("20060102150405.999")
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

func writeCSVResult(data []DBRecord, filePath string) {
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
	// "DLSpeed(DLS,KB/s)",
	// "DelayAvg(DA,ms)",
	// "DelaySource(DS)",
	// "DTPassedRate(DTPR,%)",
	// "DTCount(DTC)",
	// "DTPassedCount(DTPC)",
	// "DelayMin(DMI,ms)",
	// "DelayMax(DMX,ms)",
	// "DLTCount(DLTC)",
	// "DLTPassedCount(DLTPC)",
	// "DLTPassedRate(DLPR,%)",
	// "City(Src)",
	// "ASN(Src)",
	// "Location(CF)",
	for _, tD := range data {
		asn_str, city := "", ""
		if tD.Asn > 0 {
			asn_str = fmt.Sprintf("AS%v", tD.Asn)
			city = tD.City
		}
		err = w.Write([]string{
			tD.TestTimeStr,
			tD.IP,
			fmt.Sprintf("%.2f", tD.DLS),
			fmt.Sprintf("%.0f", tD.DA),
			tD.DS,
			fmt.Sprintf("%.2f", tD.DTPR*100),
			fmt.Sprintf("%d", tD.DTC),
			fmt.Sprintf("%d", tD.DTPC),
			fmt.Sprintf("%.0f", tD.DMI),
			fmt.Sprintf("%.0f", tD.DMX),
			fmt.Sprintf("%d", tD.DLTC),
			fmt.Sprintf("%d", tD.DLTPC),
			fmt.Sprintf("%.2f", tD.DLTPR*100),
			city,
			asn_str,
			tD.Loc,
		})
		if err != nil {
			log.Fatalf("Write csv File %v failed with: %v", filePath, err)
		}
	}
	w.Flush()
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

// func getASNAndCityWithIP(ipStr *string) (ASN int, city string) {
// 	if defaultASN > 0 || len(defaultCity) > 0 {
// 		ASN = defaultASN
// 		city = defaultCity
// 		return
// 	}
// 	// try 3 times
// 	for i := 0; i < 3; i++ {
// 		tReq, err := http.NewRequest("GET", cfURL, nil)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		var client = http.Client{
// 			Transport:     nil,
// 			CheckRedirect: nil,
// 			Jar:           nil,
// 			Timeout:       httpRspTimeoutDuration + 1*time.Second,
// 		}
// 		if len(*ipStr) > 0 && isValidIPs(*ipStr) {
// 			fullAddress := genHostFromIPStrPort(*ipStr, 443)
// 			client.Transport = &http.Transport{
// 				DialContext: GetDialContextByAddr(fullAddress),
// 				//ResponseHeaderTimeout: HttpRspTimeoutDuration,
// 			}
// 		}
// 		response, err := client.Do(tReq)
// 		// connection is failed(network error), won't continue
// 		if err != nil || response == nil {
// 			myLogger.Error(fmt.Sprintf("An error occurred while request ASN and city info from cloudflare: %v\n", err))
// 			time.Sleep(time.Duration(interval) * time.Millisecond)
// 			continue
// 		}
// 		if values, ok := (*response).Header["Cf-Meta-Asn"]; ok {
// 			if len(values) > 0 {
// 				if ASN, err = strconv.Atoi(values[0]); err != nil {
// 					myLogger.Error(fmt.Sprintf("An error occurred while convert ASN for header: %T, %v", values[0], values[0]))
// 				}
// 			}
// 		}
// 		if values, ok := (*response).Header["Cf-Meta-City"]; ok {
// 			if len(values) > 0 {
// 				city = values[0]
// 			}
// 		}
// 		if len(city) == 0 { // no "Cf-Meta-City" in header, we get "Cf-Meta-Country" instead
// 			if values, ok := (*response).Header["Cf-Meta-Country"]; ok {
// 				if len(values) > 0 {
// 					city = values[0]
// 				}
// 			}
// 		}
// 		defaultASN = ASN
// 		defaultCity = city
// 		break
// 	}
// 	return
// }

// from https://api.incolumitas.com/?q=3.5.140.2
func getGeoInfoFromIncolumitas(querIP string) (ASN int, city, country string) {
	// if defaultASN > 0 || len(defaultCity) > 0 {
	// 	ASN = defaultASN
	// 	city = defaultCity
	// 	return
	// }
	t_url, _ := url.Parse("https://api.incolumitas.com/")
	params := url.Values{}
	if len(querIP) > 0 && isValidIPs(querIP) {
		params.Add("q", querIP)
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
		time.Sleep(time.Duration(interval) * time.Millisecond)
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

// get loc from https://<cloudflared_url>/cdn-cgi/trace
func getGeoInfoFromCF(ipStr *string) (loc string) {
	baseUrl := getCFCDNCgiTraceUrl()
	t_ip := *ipStr
	t_port := -1
	t_url, t_err := url.Parse(baseUrl)
	if t_err != nil {
		myLogger.Errorln("<getGeoInfoFromCF> invalid base url ", baseUrl)
		return
	}
	if isValidIP(*ipStr) {
		if t_url.Scheme == "http" {
			t_port = 80
		} else if t_url.Scheme == "https" {
			t_port = 443
		} else {
			myLogger.Errorln("<getGeoInfoFromCF> invalid base url ", baseUrl)
			return
		}
	} else if isValidHost(*ipStr) {
		ok := true
		ok, t_ip, t_port = splitHost(*ipStr)
		if !ok {
			myLogger.Errorln("<getGeoInfoFromCF> invalid host ", *ipStr)
			return
		}
	} else {
		myLogger.Errorln("<getGeoInfoFromCF> invalid ip ", *ipStr)
		return
	}
	t_url.Host = net.JoinHostPort(t_url.Hostname(), fmt.Sprint(t_port))
	tReq, err := http.NewRequest("GET", t_url.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
	fullAddress := genHostFromIPStrPort(t_ip, t_port)
	var client = http.Client{
		Transport: &http.Transport{
			DialContext: GetDialContextByAddr(fullAddress),
			//ResponseHeaderTimeout: HttpRspTimeoutDuration,
		},
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       httpRspTimeoutDuration + 5*time.Second,
	}
	response, err := client.Do(tReq)
	// connection is failed(network error), won't continue
	if err != nil || response == nil {
		myLogger.Error(fmt.Sprintf("An error occurred while request ASN and city info from cloudflare: %v\n", err))
		time.Sleep(time.Duration(interval) * time.Millisecond)
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
	t_str_slice := strings.Split(bodyStr, "\n")
	for _, t_str := range t_str_slice {
		if strings.HasPrefix(t_str, "loc=") {
			loc = strings.TrimPrefix(t_str, "loc=")
			break
		}
	}
	return
}

func getCFCDNCgiTraceUrl() (baseurl string) {
	t_cf_url, t_err := url.Parse(baseCfCDNCgiTraceUrl)
	if t_err != nil {
		myLogger.Errorln("<getCFCgiCDNTraceUrl> invalid base url ", baseCfCDNCgiTraceUrl)
		baseurl = "https://ww1.zhfreal.top/cdn-cgi/trace"
		return
	}
	if dtOnly {
		if dtHttps {
			t_url, t_err := url.Parse(dtUrl)
			if t_err != nil {
				myLogger.Warningln("<getCFCgiCDNTraceUrl> invalid dt url ", dtUrl)
				baseurl = baseCfCDNCgiTraceUrl
				return
			}
			t_cf_url.Host = t_url.Host
		} else {
			t_cf_url.Host = hostName
		}
	} else {
		t_url, t_err := url.Parse(dltUrl)
		if t_err != nil {
			myLogger.Warningln("<getCFCgiCDNTraceUrl> invalid dt url ", dtUrl)
			baseurl = baseCfCDNCgiTraceUrl
			return
		}
		t_cf_url.Host = t_url.Host
	}
	baseurl = t_cf_url.String()
	return
}

func genDBRecords(verifyResultsSlice []VerifyResults, getLocalAsnAndCity bool) (dbRecords []DBRecord) {
	if len(verifyResultsSlice) > 0 {
		dbRecords = make([]DBRecord, 0)
		ASN, city := 0, ""
		if getLocalAsnAndCity {
			ASN, city, _ = getGeoInfoFromIncolumitas("")
		}
		for _, v := range verifyResultsSlice {
			record := DBRecord{}
			record.Asn = ASN
			record.City = city
			record.Label = suffixLabel
			record.DS = dtSource
			record.TestTimeStr = v.testTime.Format("2006-01-02 15:04:05")
			record.IP = *v.ip
			if len(*v.loc) == 0 {
				record.Loc = getGeoInfoFromCF(v.ip)
			}
			record.DTC = v.dtc
			record.DTPC = v.dtpc
			record.DTPR = v.dtpr
			record.DA = v.da
			record.DMI = v.dmi
			record.DMX = v.dmx
			record.DLTC = v.dltc
			record.DLTPC = v.dltpc
			record.DLTPR = v.dltpr
			record.DLS = v.dls
			record.DLDS = v.dlds
			record.DLTD = v.dltd
			dbRecords = append(dbRecords, record)
		}
	}
	return
}

func printFinalStat(v []VerifyResults, dtOnly bool) {
	// no data for print
	if len(v) == 0 {
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight|tabwriter.Debug)
	if !dtOnly {
		fmt.Fprintln(w, "Time\tIP\tSpeed(KB/s)\tDelayAvg(ms)\tStability(%)")
	} else {
		fmt.Fprintln(w, "Time\tIP\tDelayAvg(ms)\tStability(%)")
	}
	// close line, LatestLogLength should be 0
	fmt.Println()
	for i := 0; i < len(v); i++ {
		t_ip := *v[i].ip
		if len(*v[i].loc) > 0 {
			t_ip = fmt.Sprintf("%s#%s", t_ip, *v[i].loc)
		}
		if !dtOnly {
			fmt.Fprintf(w, "%s\t%s\t%.0f\t%.0f\t%.2f\n", v[i].testTime.Format("15:04:05"), t_ip, v[i].dls, v[i].da, v[i].dtpr*100)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%.0f\t%.2f\n", v[i].testTime.Format("15:04:05"), t_ip, v[i].da, v[i].dtpr*100)
		}
	}
	fmt.Println()
	w.Flush()
}

func saveDBRecords(dbRecords []DBRecord, dbFile string) {
	if len(dbRecords) > 0 {
		db, err := OpenSqlite(dbFile)
		if err != nil {
			myLogger.Errorln("<saveDBRecords> open sqlite error ", err)
			return
		}
		err = AddCFDTRecords(db, dbRecords)
		if err != nil {
			myLogger.Errorln("<saveDBRecords> add CFDT records error ", err)
		}
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
	tVerifyResult.loc = &out.loc
	if len(out.resultSlice) == 0 {
		return tVerifyResult
	}
	tVerifyResult.dtc = len(out.resultSlice)
	var tDurationsAll = 0.0
	var tDownloadDurations float64
	var tDownloadSizes int64
	var t_delays_slice = []float64{}
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
			t_delays_slice = append(t_delays_slice, tDuration)
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
		// just calculate variance and standard deviation when we ev-dt enabled
		if enableStdEv {
			tVerifyResult.daVar = variance(t_delays_slice)
			tVerifyResult.daStd = std(t_delays_slice)
		}
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

// test detail
// loglvl should be logLevelDebug or logLevelInfo
// when: 1. in non-debug mode, just print stats instead of pure qualified IPs.
//  2. in debug mode, we show more as tcell or non-tcell form.
//
// isResult: used for tcell mode, indicate show in result area or debug area.
func displayDetails(isResult, isSilence bool, v []VerifyResults) {
	if !debug {
		// myLogger.PrintClearIPs(v)
		if isSilence {
			for _, t_v := range v {
				tStr := *t_v.ip
				if len(*t_v.loc) > 0 {
					tStr = fmt.Sprintf("%s#%s", tStr, *t_v.loc)
				}
				myLogger.Println(tStr)
			}
		} else {
			myLogger.PrintDetails(LogLevel(logLevelInfo), v)
		}
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

func retrieveSome(amount int) (targetIPs []*string) {
	var t_target = []*string{}
	targetIPs = append(targetIPs, retrieveHosts(amount)...)
	t_target = append(t_target, retrieveIPsFromIPR(amount)...)
	for _, ipStr := range t_target {
		for _, port := range ports {
			host := genHostFromIPStrPort(*ipStr, port)
			if len(host) > 0 {
				targetIPs = append(targetIPs, &host)
			}
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
		return ""
	}
	if port < 1 || port > 65535 {
		return ""
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

func uniqueIntSlice(strSlice []int) []int {
	keys := make(map[int]bool)
	list := []int{}
	for _, entry := range strSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
func mean(v []float64) float64 {
	var res float64 = 0
	var n int = len(v)
	for i := 0; i < n; i++ {
		res += v[i]
	}
	return res / float64(n)
}

func variance(v []float64) float64 {
	var res float64 = 0
	var m = mean(v)
	var n int = len(v)
	for i := 0; i < n; i++ {
		res += (v[i] - m) * (v[i] - m)
	}
	return res / float64(n-1)
}

func std(v []float64) float64 {
	return roundFloat(math.Sqrt(variance(v)), 2)
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
