package utils

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"cftestor/internal/logger"
	"github.com/tidwall/gjson"
)

func GetTimeNowStr() string {
	return time.Now().Format("15:04:05")
}

func GetTimeNowStrSuffix() string {
	s := time.Now().Format("20060102150405")
	return strings.ReplaceAll(s, ".", "")
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func ParseUrl(urlStr string, defaultDLTUrl string) (tHostName string, tPort int, err error) {
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

func NewUrl(urlStr, port string, defaultDLTUrl string) (string, error) {
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

func InitRandSeed(myRand *rand.Rand) {
	myRand.Seed(time.Now().UnixNano())
}

// from https://api.incolumitas.com/?q=3.5.140.2
func GetGeoInfoFromIncolumitas(ipStr string, interval int) (ASN int, city, country string) {
	t_url, _ := url.Parse("https://api.incolumitas.com/")
	params := url.Values{}
	if len(ipStr) > 0 && IsValidIPs(ipStr) {
		params.Add("q", ipStr)
	}
	t_url.RawQuery = params.Encode()
	tReq, err := http.NewRequest("GET", t_url.String(), nil)
	if err != nil {
		logger.Log.Errorf("failed to create Incolumitas request: %v\n", err)
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
		logger.Log.Errorf("failed to request ASN and city info from Incolumitas: %v\n", err)
		time.Sleep(time.Duration(interval) * time.Millisecond)
		return
	}
	defer response.Body.Close()
	// read response.Body as string
	body, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Log.Errorf("failed to read Incolumitas response body: %v\n", err)
		return
	}
	// decode body []byte into string
	bodyStr := string(body)
	asn := gjson.Get(bodyStr, "asn.asn")
	if asn.Exists() {
		if ASN, err = strconv.Atoi(asn.String()); err != nil {
			logger.Log.Errorf("failed to parse ASN from Incolumitas response: %T, %v\n", asn.String(), asn.String())
		}
	}
	city = gjson.Get(bodyStr, "location.city").String()
	country = gjson.Get(bodyStr, "location.country_code").String()
	return
}

func EraseLine(n int) {
	if n <= 0 {
		return
	}
	fmt.Printf("%s", strings.Repeat("\b \b", n))
}

func IsValidCIDR(ips string) bool {
	_, _, err := net.ParseCIDR(ips)
	return err == nil
}

func IsValidIP(ip string) bool {
	tIP := net.ParseIP(ip)
	return tIP != nil
}

func IsValidIPs(ips string) bool {
	ips = strings.TrimSpace(ips)
	if IsValidCIDR(ips) {
		return true
	} else {
		return IsValidIP(ips)
	}
}

func IsValidHost(host string) bool {
	ok, _, _ := SplitHost(host)
	return ok
}

func GetHostVer(host string) (ver int8) {
	tHost, _, err := net.SplitHostPort(host)
	if err != nil {
		return 0 // TypeIPErr
	}
	tVer := GetIPVer(tHost)
	if tVer != 0 {
		return tVer
	}
	if IsValidDNSHost(tHost) {
		return 1 | 2 // TypeIPv4 | TypeIPv6
	}
	return 0
}

func GetIPsVer(ips string) (ver int8) {
	tV := GetCIDRVer(ips)
	if tV == 0 {
		return GetIPVer(ips)
	} else {
		return tV
	}
}

func GetCIDRVer(ips string) (ver int8) {
	_, _, err := net.ParseCIDR(ips)
	if err != nil {
		return 0
	}
	if strings.Contains(ips, ":") {
		return 2 // TypeIPv6
	}
	return 1 // TypeIPv4
}

func GetIPVer(ips string) (ver int8) {
	tIP := net.ParseIP(ips)
	if tIP == nil {
		return 0
	} else if tIP.To4() != nil {
		return 1 // TypeIPv4
	} else {
		return 2 // TypeIPv6
	}
}

func SplitHost(host string) (bool, string, int) {
	host = strings.TrimSpace(host)
	if len(host) == 0 {
		return false, "", -1
	}
	tHost, port, err := net.SplitHostPort(host)
	if err != nil {
		return false, "", -1
	}
	// invalid host name or IP in host
	if !IsValidIPs(tHost) && !IsValidDNSHost(tHost) {
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

func IsValidDNSHost(host string) bool {
	host = strings.TrimSpace(strings.TrimSuffix(host, "."))
	if len(host) == 0 || len(host) > 253 {
		return false
	}
	if LooksLikeIPv4Literal(host) {
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

func LooksLikeIPv4Literal(host string) bool {
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

func GenHostFromIPStrPort(ip string, port int) (connStr string) {
	if !IsValidIPs(ip) {
		return ""
	}
	if port < 1 || port > 65535 {
		return ""
	}
	connStr = net.JoinHostPort(ip, strconv.Itoa(port))
	return
}

func Confirm(s string, tries int) bool {
	reader := bufio.NewReader(os.Stdin)
	for ; tries > 0; tries-- {
		fmt.Printf("%s [y/yes/N/no]: ", s)
		rsp, err := reader.ReadString('\n')
		if err != nil {
			logger.Log.Errorf("failed to read confirmation: %v\n", err)
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

func NewRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func WriteStringsToFile(filename string, lines []string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, line := range lines {
		if _, err := w.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return w.Flush()
}

func UniqueIntSlice(strSlice []int) []int {
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

func Mean(v []float64) float64 {
	var res float64 = 0
	var n int = len(v)
	for i := 0; i < n; i++ {
		res += v[i]
	}
	return res / float64(n)
}

func Variance(v []float64) float64 {
	if len(v) <= 1 {
		return 0
	}
	var res float64 = 0
	var m = Mean(v)
	var n int = len(v)
	for i := 0; i < n; i++ {
		res += (v[i] - m) * (v[i] - m)
	}
	return res / float64(n-1)
}

func Std(v []float64) float64 {
	if len(v) <= 1 {
		return 0
	}
	return RoundFloat(math.Sqrt(Variance(v)), 2)
}

func RoundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
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

