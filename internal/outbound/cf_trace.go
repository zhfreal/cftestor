package outbound

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"cftestor/internal/config"
	"cftestor/internal/logger"
	"cftestor/internal/utils"
)

func GetGeoInfoFromCF(ipStr *string) (loc string) {
	if ipStr == nil || len(*ipStr) == 0 {
		return ""
	}
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
	tr := &http.Transport{
		DialContext:       GetDialContextByAddr(fullAddress),
		DisableKeepAlives: true,
	}
	defer tr.CloseIdleConnections()
	var client = http.Client{
		Transport:     tr,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       config.Config.HttpRspTimeoutDuration + 5*time.Second,
	}
	response, err := client.Do(tReq)
	if err != nil || response == nil {
		logger.Log.Errorf("failed to request Cloudflare trace location: %v\n", err)
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ""
	}
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
	scanner := bufio.NewScanner(io.LimitReader(body, 8192))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if strings.EqualFold(k, "colo") {
			loc = strings.ToUpper(v)
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

// ResolveLocationsParallel resolves missing locations for results concurrently up to concurrency limit,
// capped at the quantity of candidates needing resolution.
func ResolveLocationsParallel(results []config.VerifyResults, concurrency int) {
	ResolveLocationsParallelWithResolver(results, concurrency, GetGeoInfoFromCF)
}

// ResolveLocationsParallelWithResolver resolves missing locations using the provided resolver function.
func ResolveLocationsParallelWithResolver(results []config.VerifyResults, concurrency int, resolver func(ip *string) string) {
	if len(results) == 0 {
		return
	}
	if resolver == nil {
		resolver = GetGeoInfoFromCF
	}

	var toResolve []int
	for i := range results {
		if results[i].IP != nil && (results[i].Loc == nil || len(*results[i].Loc) == 0) {
			toResolve = append(toResolve, i)
		}
	}
	if len(toResolve) == 0 {
		return
	}

	if concurrency <= 0 {
		concurrency = 1
	}
	// Cap threads to the quantity of candidates needing resolution
	if concurrency > len(toResolve) {
		concurrency = len(toResolve)
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for _, idx := range toResolve {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			loc := resolver(results[i].IP)
			results[i].Loc = &loc
		}(idx)
	}
	wg.Wait()
}
