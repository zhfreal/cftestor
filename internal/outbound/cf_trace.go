package outbound

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cftestor/internal/config"
	"cftestor/internal/logger"
	"cftestor/internal/utils"
)

// GetGeoInfoFromCF gets loc from https://<cloudflared_url>/cdn-cgi/trace
func GetGeoInfoFromCF(ipStr *string) (loc string) {
	baseUrl := getCFCDNCgiTraceUrl()
	t_ip := *ipStr
	t_port := -1
	t_url, t_err := url.Parse(baseUrl)
	if t_err != nil {
		logger.Log.Errorf("invalid Cloudflare trace base URL %q: %v\n", baseUrl, t_err)
		return
	}
	if utils.IsValidIP(*ipStr) {
		if t_url.Scheme == "http" {
			t_port = 80
		} else if t_url.Scheme == "https" {
			t_port = 443
		} else {
			logger.Log.Errorf("invalid Cloudflare trace base URL %q: unsupported scheme %q\n", baseUrl, t_url.Scheme)
			return
		}
	} else if utils.IsValidHost(*ipStr) {
		ok := true
		ok, t_ip, t_port = utils.SplitHost(*ipStr)
		if !ok {
			logger.Log.Errorf("invalid host:port for Cloudflare trace lookup: %q\n", *ipStr)
			return
		}
	} else {
		logger.Log.Errorf("invalid IP or host:port for Cloudflare trace lookup: %q\n", *ipStr)
		return
	}
	t_url.Host = net.JoinHostPort(t_url.Hostname(), fmt.Sprint(t_port))
	tReq, err := http.NewRequest("GET", t_url.String(), nil)
	if err != nil {
		logger.Log.Errorf("failed to create Cloudflare trace request: %v\n", err)
		return
	}
	fullAddress := net.JoinHostPort(t_ip, fmt.Sprint(t_port))
	var client = http.Client{
		Transport: &http.Transport{
			DialContext: GetDialContextByAddr(fullAddress),
		},
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       config.Config.HttpRspTimeoutDuration + 5*time.Second,
	}
	response, err := client.Do(tReq)
	if err != nil || response == nil {
		logger.Log.Errorf("failed to request Cloudflare trace location: %v\n", err)
		time.Sleep(time.Duration(config.Config.Interval) * time.Millisecond)
		return
	}
	defer response.Body.Close()
	loc, err = get_loc_from_cf_resp(response.Body)
	if err != nil {
		logger.Log.Errorf("failed to read Cloudflare trace response body: %v\n", err)
		return
	}
	return
}

func getCFCDNCgiTraceUrl() (baseurl string) {
	t_cf_url, t_err := url.Parse(config.BaseCfCDNCgiTraceUrl)
	if t_err != nil {
		logger.Log.Errorf("invalid default Cloudflare trace URL %q: %v\n", config.BaseCfCDNCgiTraceUrl, t_err)
		baseurl = config.BaseCfCDNCgiTraceUrl
		return
	}
	if config.Config.DTOnly {
		if config.Config.DTHttps {
			t_url, t_err := url.Parse(config.Config.DTUrl)
			if t_err != nil {
				logger.Log.Warningf("invalid --dt-url for Cloudflare trace lookup %q: %v\n", config.Config.DTUrl, t_err)
				baseurl = config.BaseCfCDNCgiTraceUrl
				return
			}
			t_cf_url.Host = t_url.Host
		} else {
			t_cf_url.Host = config.Config.HostName
		}
	} else {
		t_url, t_err := url.Parse(config.Config.DLTUrl)
		if t_err != nil {
			logger.Log.Warningf("invalid --dlt-url for Cloudflare trace lookup %q: %v\n", config.Config.DLTUrl, t_err)
			baseurl = config.BaseCfCDNCgiTraceUrl
			return
		}
		t_cf_url.Host = t_url.Host
	}
	baseurl = t_cf_url.String()
	return
}

func get_loc_from_cf_resp(body io.ReadCloser) (string, error) {
	loc := ""
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		t_slice := strings.Split(line, "=")
		if len(t_slice) != 2 {
			continue
		}
		if strings.ToLower(t_slice[0]) == "colo" {
			loc = strings.ToUpper(t_slice[1])
			if len(loc) > 3 {
				loc = loc[:3]
			}
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	t, ok := utils.IataMap[loc]
	if ok {
		return t, nil
	} else {
		return loc, nil
	}
}
