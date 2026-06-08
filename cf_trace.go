package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// get loc from https://<cloudflared_url>/cdn-cgi/trace
func getGeoInfoFromCF(ipStr *string) (loc string) {
	baseUrl := getCFCDNCgiTraceUrl()
	t_ip := *ipStr
	t_port := -1
	t_url, t_err := url.Parse(baseUrl)
	if t_err != nil {
		myLogger.Errorf("invalid Cloudflare trace base URL %q: %v\n", baseUrl, t_err)
		return
	}
	if isValidIP(*ipStr) {
		if t_url.Scheme == "http" {
			t_port = 80
		} else if t_url.Scheme == "https" {
			t_port = 443
		} else {
			myLogger.Errorf("invalid Cloudflare trace base URL %q: unsupported scheme %q\n", baseUrl, t_url.Scheme)
			return
		}
	} else if isValidHost(*ipStr) {
		ok := true
		ok, t_ip, t_port = splitHost(*ipStr)
		if !ok {
			myLogger.Errorf("invalid host:port for Cloudflare trace lookup: %q\n", *ipStr)
			return
		}
	} else {
		myLogger.Errorf("invalid IP or host:port for Cloudflare trace lookup: %q\n", *ipStr)
		return
	}
	t_url.Host = net.JoinHostPort(t_url.Hostname(), fmt.Sprint(t_port))
	tReq, err := http.NewRequest("GET", t_url.String(), nil)
	if err != nil {
		myLogger.Errorf("failed to create Cloudflare trace request: %v\n", err)
		return
	}
	fullAddress := net.JoinHostPort(t_ip, fmt.Sprint(t_port))
	var client = http.Client{
		Transport: &http.Transport{
			DialContext: GetDialContextByAddr(fullAddress),
		},
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       Config.HttpRspTimeoutDuration + 5*time.Second,
	}
	response, err := client.Do(tReq)
	// connection is failed(network error), won't continue
	if err != nil || response == nil {
		myLogger.Errorf("failed to request Cloudflare trace location: %v\n", err)
		time.Sleep(time.Duration(Config.Interval) * time.Millisecond)
		return
	}
	defer response.Body.Close()
	// read response.Body as string
	loc, err = get_loc_from_cf_resp(response.Body)
	if err != nil {
		myLogger.Errorf("failed to read Cloudflare trace response body: %v\n", err)
		return
	}
	return
}

func getCFCDNCgiTraceUrl() (baseurl string) {
	t_cf_url, t_err := url.Parse(baseCfCDNCgiTraceUrl)
	if t_err != nil {
		myLogger.Errorf("invalid default Cloudflare trace URL %q: %v\n", baseCfCDNCgiTraceUrl, t_err)
		baseurl = baseCfCDNCgiTraceUrl
		return
	}
	if Config.DTOnly {
		if Config.DTHttps {
			t_url, t_err := url.Parse(Config.DTUrl)
			if t_err != nil {
				myLogger.Warningf("invalid --dt-url for Cloudflare trace lookup %q: %v\n", Config.DTUrl, t_err)
				baseurl = baseCfCDNCgiTraceUrl
				return
			}
			t_cf_url.Host = t_url.Host
		} else {
			t_cf_url.Host = Config.HostName
		}
	} else {
		t_url, t_err := url.Parse(Config.DLTUrl)
		if t_err != nil {
			myLogger.Warningf("invalid --dlt-url for Cloudflare trace lookup %q: %v\n", Config.DLTUrl, t_err)
			baseurl = baseCfCDNCgiTraceUrl
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
		// Apply the filter function
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
	// get country code from iataMap
	t, ok := iataMap[loc]
	if ok {
		return t, nil
	} else {
		return loc, nil
	}
}
