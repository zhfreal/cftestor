package main

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

var dohClient = &http.Client{
	Timeout: 3 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	},
}

type ripeStatResponse struct {
	Data struct {
		Prefixes []struct {
			Prefix string `json:"prefix"`
		} `json:"prefixes"`
	} `json:"data"`
}

func fetchBGPPrefixes() ([]string, error) {
	myLogger.Infoln("Fetching BGP prefixes from RIPEstat...")
	resp, err := http.Get("https://stat.ripe.net/data/announced-prefixes/data.json?resource=AS13335")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RIPEstat returned status %d", resp.StatusCode)
	}

	var data ripeStatResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var ipv6Prefixes []string
	for _, p := range data.Data.Prefixes {
		if strings.Contains(p.Prefix, ":") { // IPv6
			ipv6Prefixes = append(ipv6Prefixes, p.Prefix)
		}
	}
	myLogger.Infof("Found %d BGP IPv6 prefixes.", len(ipv6Prefixes))
	return ipv6Prefixes, nil
}

func fetchTrancoDomains(limit int) ([]string, error) {
	myLogger.Infoln("Fetching top domains from Tranco...")
	resp, err := http.Get("https://tranco-list.eu/top-1m.csv.zip")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Tranco download returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}
	if len(zipReader.File) == 0 {
		return nil, fmt.Errorf("empty zip file")
	}
	f, err := zipReader.File[0].Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var domains []string
	for _, line := range lines {
		parts := strings.SplitN(line, ",", 2)
		if len(parts) == 2 {
			domain := strings.TrimSpace(parts[1])
			if domain != "" {
				domains = append(domains, domain)
			}
		}
		if len(domains) >= limit {
			break
		}
	}
	return domains, nil
}

// resolveDomain performs a DNS query using various methods (UDP, TCP, DoT, DoH)
func resolveDomain(qname string, qtype uint16, netType, server string) (*dns.Msg, error) {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(qname), qtype)
	m.RecursionDesired = true

	if netType == "https" {
		// DoH
		packed, err := m.Pack()
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequest("POST", server, bytes.NewReader(packed))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/dns-message")
		req.Header.Set("Accept", "application/dns-message")
		resp, err := dohClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("DoH bad status: %d", resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		r := new(dns.Msg)
		err = r.Unpack(body)
		return r, err
	}

	c := new(dns.Client)
	c.Net = netType
	c.Timeout = 3 * time.Second
	if netType == "tcp-tls" {
		c.TLSConfig = &tls.Config{InsecureSkipVerify: false}
	}
	r, _, err := c.Exchange(m, server)
	return r, err
}

func FetchDynamicIPv6(dnsServerStr string) ([]string, error) {
	bgpPrefixes, err := fetchBGPPrefixes()
	if err != nil {
		myLogger.Warningf("Failed to fetch BGP prefixes: %v\n", err)
	}

	domains, err := fetchTrancoDomains(10000)
	if err != nil {
		myLogger.Warningf("Failed to fetch Tranco domains: %v\n", err)
	}

	// parse DNS server
	netType := "udp"
	serverAddr := dnsServerStr
	if serverAddr == "" {
		serverAddr = "1.1.1.1:53"
	}

	if strings.HasPrefix(serverAddr, "udp://") {
		netType = "udp"
		serverAddr = strings.TrimPrefix(serverAddr, "udp://")
		if !strings.Contains(serverAddr, ":") {
			serverAddr += ":53"
		}
	} else if strings.HasPrefix(serverAddr, "tcp://") {
		netType = "tcp"
		serverAddr = strings.TrimPrefix(serverAddr, "tcp://")
		if !strings.Contains(serverAddr, ":") {
			serverAddr += ":53"
		}
	} else if strings.HasPrefix(serverAddr, "tls://") || strings.HasPrefix(serverAddr, "dot://") {
		netType = "tcp-tls"
		serverAddr = strings.TrimPrefix(serverAddr, "tls://")
		serverAddr = strings.TrimPrefix(serverAddr, "dot://")
		if !strings.Contains(serverAddr, ":") {
			serverAddr += ":853"
		}
	} else if strings.HasPrefix(serverAddr, "https://") {
		netType = "https"
	} else {
		// no scheme, default UDP with TCP fallback
		netType = "udp-tcp" // custom string to indicate fallback
		if !strings.Contains(serverAddr, ":") {
			serverAddr += ":53"
		}
	}

	myLogger.Infoln("Resolving domains to map Cloudflare IPv6 allocations...")

	var wg sync.WaitGroup
	sem := make(chan struct{}, 100) // 100 concurrent workers

	var mu sync.Mutex
	mappedCIDRs := make(map[string]bool)

	for _, domain := range domains {
		wg.Add(1)
		sem <- struct{}{}
		go func(d string) {
			defer wg.Done()
			defer func() { <-sem }()

			// Resolve NS
			var rNS *dns.Msg
			var err error

			if netType == "udp-tcp" {
				rNS, err = resolveDomain(d, dns.TypeNS, "udp", serverAddr)
				if err == nil && rNS != nil && rNS.Truncated {
					rNS, err = resolveDomain(d, dns.TypeNS, "tcp", serverAddr)
				}
			} else {
				rNS, err = resolveDomain(d, dns.TypeNS, netType, serverAddr)
			}

			isCF := false
			if err == nil && rNS != nil {
				for _, ans := range rNS.Answer {
					if ns, ok := ans.(*dns.NS); ok {
						if strings.HasSuffix(strings.ToLower(ns.Ns), ".ns.cloudflare.com.") {
							isCF = true
							break
						}
					}
				}
			}

			if !isCF {
				return
			}

			// Resolve AAAA
			var rAAAA *dns.Msg
			if netType == "udp-tcp" {
				rAAAA, err = resolveDomain(d, dns.TypeAAAA, "udp", serverAddr)
				if err == nil && rAAAA != nil && rAAAA.Truncated {
					rAAAA, err = resolveDomain(d, dns.TypeAAAA, "tcp", serverAddr)
				}
			} else {
				rAAAA, err = resolveDomain(d, dns.TypeAAAA, netType, serverAddr)
			}

			if err == nil && rAAAA != nil {
				for _, ans := range rAAAA.Answer {
					if aaaa, ok := ans.(*dns.AAAA); ok {
						ip := aaaa.AAAA
						ip48 := ip.Mask(net.CIDRMask(48, 128))
						ip112 := ip.Mask(net.CIDRMask(112, 128))

						mu.Lock()
						mappedCIDRs[fmt.Sprintf("%s/48", ip48.String())] = true
						mappedCIDRs[fmt.Sprintf("%s/112", ip112.String())] = true
						mu.Unlock()
					}
				}
			}
		}(domain)
	}

	wg.Wait()

	finalMap := make(map[string]bool)
	for _, p := range bgpPrefixes {
		finalMap[p] = true
	}
	for p := range mappedCIDRs {
		finalMap[p] = true
	}

	var results []string
	for p := range finalMap {
		results = append(results, p)
	}
	
	myLogger.Infof("Dynamically fetched %d valid Cloudflare IPv6 CIDRs.", len(results))

	return results, nil
}
