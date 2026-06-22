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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

const (
	// DNS settings
	DefaultDNSTimeout    = 3 * time.Second
	DefaultDNSServer     = "1.1.1.1:53"
	DefaultDoHMaxIdle    = 100
	DefaultDoHIdleTimeout = 90 * time.Second
)

var (
	// Resolution target and performance limits
	DNSConcurrencyLimit  = 100
)

var dohClient = &http.Client{
	Timeout: DefaultDNSTimeout,
	Transport: &http.Transport{
		MaxIdleConns:        DefaultDoHMaxIdle,
		MaxIdleConnsPerHost: DefaultDoHMaxIdle,
		IdleConnTimeout:     DefaultDoHIdleTimeout,
	},
}

type ripeStatResponse struct {
	Data struct {
		Prefixes []struct {
			Prefix string `json:"prefix"`
		} `json:"prefixes"`
	} `json:"data"`
}

var seedCFDomains = []string{
	"speed.cloudflare.com",
	"cdnjs.cloudflare.com",
	"cloudflare.com",
	"dash.cloudflare.com",
	"discord.com",
	"canva.com",
	"gitlab.com",
	"zoom.us",
}

type IPNetList []*net.IPNet

func parseCIDRs(cidrs []string) IPNetList {
	var list IPNetList
	for _, cidr := range cidrs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err == nil && ipnet != nil {
			list = append(list, ipnet)
		}
	}
	return list
}

func (list IPNetList) Contains(ip net.IP) bool {
	for _, ipnet := range list {
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}

func fetchRawBGPPrefixes() ([]string, error) {
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

	var prefixes []string
	for _, p := range data.Data.Prefixes {
		prefixes = append(prefixes, p.Prefix)
	}
	return prefixes, nil
}

func fetchBGPPrefixes(ipVersion int) ([]string, error) {
	rawPrefixes, err := fetchRawBGPPrefixes()
	if err != nil {
		return nil, err
	}

	var prefixes []string
	for _, p := range rawPrefixes {
		isV6 := strings.Contains(p, ":")
		if ipVersion == 6 && isV6 {
			prefixes = append(prefixes, p)
		} else if ipVersion == 4 && !isV6 {
			// For IPv4, filter out subnets smaller than /16 (prefix length < 16)
			parts := strings.Split(p, "/")
			if len(parts) == 2 {
				maskSize, err := strconv.Atoi(parts[1])
				if err == nil && maskSize < 16 {
					continue
				}
			}
			prefixes = append(prefixes, p)
		}
	}
	myLogger.Infof("Found %d BGP IPv%d prefixes.", len(prefixes), ipVersion)
	return prefixes, nil
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
	c.Timeout = DefaultDNSTimeout
	if netType == "tcp-tls" {
		c.TLSConfig = &tls.Config{InsecureSkipVerify: false}
	}
	r, _, err := c.Exchange(m, server)
	return r, err
}

func parseDNSServer(dnsServerStr string) (string, string) {
	netType := "udp"
	serverAddr := dnsServerStr
	if serverAddr == "" {
		if dnsConf, err := dns.ClientConfigFromFile("/etc/resolv.conf"); err == nil && len(dnsConf.Servers) > 0 {
			serverAddr = net.JoinHostPort(dnsConf.Servers[0], dnsConf.Port)
		} else {
			serverAddr = DefaultDNSServer
		}
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
	return netType, serverAddr
}

func resolveDomainWithFallback(qname string, qtype uint16, netType, serverAddr string) (*dns.Msg, error) {
	if netType == "udp-tcp" {
		r, err := resolveDomain(qname, qtype, "udp", serverAddr)
		if err == nil && r != nil && r.Truncated {
			r, err = resolveDomain(qname, qtype, "tcp", serverAddr)
		}
		return r, err
	}
	return resolveDomain(qname, qtype, netType, serverAddr)
}

func FetchCloudflareDomains(dnsServerStr string) ([]string, error) {
	bgpPrefixes, err := fetchRawBGPPrefixes()
	if err != nil {
		myLogger.Warningf("Failed to fetch BGP prefixes for domain check: %v\n", err)
	}
	bgpIPNets := parseCIDRs(bgpPrefixes)

	domains, err := fetchTrancoDomains(Config.TrancoLimit)
	if err != nil {
		myLogger.Warningf("Failed to fetch Tranco domains: %v\n", err)
	}

	// Prepend seed domains to guarantee Cloudflare-served domains are resolved
	allDomains := append([]string{}, seedCFDomains...)
	allDomains = append(allDomains, domains...)

	netType, serverAddr := parseDNSServer(dnsServerStr)

	myLogger.Infoln("Checking which top domains are served by Cloudflare CDN...")

	var wg sync.WaitGroup
	sem := make(chan struct{}, DNSConcurrencyLimit)

	var mu sync.Mutex
	verifiedDomains := make(map[string]bool)

	for _, domain := range allDomains {
		wg.Add(1)
		sem <- struct{}{}
		go func(d string) {
			defer wg.Done()
			defer func() { <-sem }()

			isCF := false

			// Check A record (IPv4)
			rA, err := resolveDomainWithFallback(d, dns.TypeA, netType, serverAddr)
			if err == nil && rA != nil {
				for _, ans := range rA.Answer {
					if a, ok := ans.(*dns.A); ok {
						if bgpIPNets.Contains(a.A) {
							isCF = true
							break
						}
					}
				}
			}

			// If not found in A, check AAAA record (IPv6)
			if !isCF {
				rAAAA, err := resolveDomainWithFallback(d, dns.TypeAAAA, netType, serverAddr)
				if err == nil && rAAAA != nil {
					for _, ans := range rAAAA.Answer {
						if aaaa, ok := ans.(*dns.AAAA); ok {
							if bgpIPNets.Contains(aaaa.AAAA) {
								isCF = true
								break
							}
						}
					}
				}
			}

			if isCF {
				mu.Lock()
				verifiedDomains[d] = true
				mu.Unlock()
			}
		}(domain)
	}

	wg.Wait()

	var results []string
	for d := range verifiedDomains {
		results = append(results, d)
	}

	myLogger.Infof("Found %d verified Cloudflare CDN served domains.", len(results))
	return results, nil
}

func FetchDynamicIPv4(dnsServerStr string) ([]string, error) {
	bgpPrefixes, err := fetchBGPPrefixes(4)
	if err != nil {
		myLogger.Warningf("Failed to fetch BGP prefixes: %v\n", err)
	}

	cfDomains, err := FetchCloudflareDomains(dnsServerStr)
	if err != nil {
		return bgpPrefixes, err
	}

	netType, serverAddr := parseDNSServer(dnsServerStr)

	myLogger.Infoln("Resolving domains to map Cloudflare IPv4 allocations...")

	var wg sync.WaitGroup
	sem := make(chan struct{}, DNSConcurrencyLimit)

	var mu sync.Mutex
	mappedCIDRs := make(map[string]bool)

	for _, domain := range cfDomains {
		wg.Add(1)
		sem <- struct{}{}
		go func(d string) {
			defer wg.Done()
			defer func() { <-sem }()

			rA, err := resolveDomainWithFallback(d, dns.TypeA, netType, serverAddr)
			if err == nil && rA != nil {
				for _, ans := range rA.Answer {
					if a, ok := ans.(*dns.A); ok {
						ip := a.A
						ip24 := ip.Mask(net.CIDRMask(24, 32))
						ip32 := ip.Mask(net.CIDRMask(32, 32))

						mu.Lock()
						mappedCIDRs[fmt.Sprintf("%s/24", ip24.String())] = true
						mappedCIDRs[fmt.Sprintf("%s/32", ip32.String())] = true
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
		// Filter out subnets smaller than /16 for IPv4
		parts := strings.Split(p, "/")
		if len(parts) == 2 {
			maskSize, err := strconv.Atoi(parts[1])
			if err == nil && maskSize < 16 {
				continue
			}
		}
		finalMap[p] = true
	}

	var results []string
	for p := range finalMap {
		results = append(results, p)
	}

	myLogger.Infof("Dynamically fetched %d valid Cloudflare IPv4 CIDRs.", len(results))

	return results, nil
}

func FetchDynamicIPv6(dnsServerStr string) ([]string, error) {
	bgpPrefixes, err := fetchBGPPrefixes(6)
	if err != nil {
		myLogger.Warningf("Failed to fetch BGP prefixes: %v\n", err)
	}

	cfDomains, err := FetchCloudflareDomains(dnsServerStr)
	if err != nil {
		return bgpPrefixes, err
	}

	netType, serverAddr := parseDNSServer(dnsServerStr)

	myLogger.Infoln("Resolving domains to map Cloudflare IPv6 allocations...")

	var wg sync.WaitGroup
	sem := make(chan struct{}, DNSConcurrencyLimit)

	var mu sync.Mutex
	mappedCIDRs := make(map[string]bool)

	for _, domain := range cfDomains {
		wg.Add(1)
		sem <- struct{}{}
		go func(d string) {
			defer wg.Done()
			defer func() { <-sem }()

			rAAAA, err := resolveDomainWithFallback(d, dns.TypeAAAA, netType, serverAddr)
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
