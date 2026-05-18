package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
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
		},
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       Config.HttpRspTimeoutDuration + 5*time.Second,
	}
	response, err := client.Do(tReq)
	// connection is failed(network error), won't continue
	if err != nil || response == nil {
		myLogger.Error(fmt.Sprintf("An error occurred while request ASN and city info from cloudflare: %v\n", err))
		time.Sleep(time.Duration(Config.Interval) * time.Millisecond)
		return
	}
	// read response.Body as string
	loc, err = get_loc_from_cf_resp(response.Body)
	if err != nil {
		myLogger.Error(fmt.Sprintf("An error occurred while read response.Body: %v\n", err))
		return
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
	if Config.DTOnly {
		if Config.DTHttps {
			t_url, t_err := url.Parse(Config.DTUrl)
			if t_err != nil {
				myLogger.Warningln("<getCFCgiCDNTraceUrl> invalid dt url ", Config.DTUrl)
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
			myLogger.Warningln("<getCFCgiCDNTraceUrl> invalid dt url ", Config.DTUrl)
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
