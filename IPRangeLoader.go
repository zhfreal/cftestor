package main

import (
	"math"
	"net"
)

func extractCIDRHosts(num int, isRandom bool) (targIPs []string) {
	if num < 0 {
		return
	}
	// find the factor t_len
	t_num := num
	t_IPs := []net.IP{}
	for len(t_IPs) < num {
		t_len := 0
		for i := 0; i < len(srcIPRs); i++ {
			this_ipr := &srcIPRs[i]
			if !this_ipr.Extracted {
				t_len += 1
			} else if len(srcIPRsCache[i]) > 0 {
				t_len += 1
			}
		}
		// don't have more srcIPRs for extracted
		if t_len <= 0 {
			break
		}
		// set expected amount IPs from each blocks
		t_v_num := 0
		// get IPs from srcIPRs
		var t_t_IPs []net.IP
		for i := 0; i < len(srcIPRs) && t_num > 0 && t_len > 0; i++ {
			this_ipr := &srcIPRs[i]
			// reset expected amount IPs from each blocks
			t_v_num = int(math.Ceil(float64(t_num) * srcFactor[i]))
			if !this_ipr.Extracted {
				if isRandom { // random
					t_t_IPs = this_ipr.GetRandomX(t_v_num)
				} else { // sequence
					t_t_IPs = this_ipr.Extract(t_v_num)
				}
			} else if len(srcIPRsCache[i]) > 0 {
				// srcIPRsCache[i] have little IPs, get them all
				if len(srcIPRsCache[i]) <= t_v_num {
					t_t_IPs = srcIPRsCache[i]
					srcIPRsCache[i] = []net.IP{}
				} else {
					// shuffle
					myRand.Shuffle(len(srcIPRsCache[i]), func(m, n int) {
						srcIPRsCache[i][m], srcIPRsCache[i][n] = srcIPRsCache[i][n], srcIPRsCache[i][m]
					})
					t_t_IPs = srcIPRsCache[i][0:t_v_num]
					srcIPRsCache[i] = srcIPRsCache[i][t_v_num:]
				}
			}
			if len(t_t_IPs) > 0 {
				for j := 0; j < len(t_t_IPs); j++ {
					t_IPs = append(t_IPs, t_t_IPs[j])
				}
				t_num -= len(t_t_IPs)
				t_len--
			}
		}
	}
	for i := 0; i < len(t_IPs); i++ {
		targIPs = append(targIPs, t_IPs[i].String())
	}
	return
}

func isValidIPs(ips string) bool {
	_, _, err := net.ParseCIDR(ips)
	if err != nil {
		tIP := net.ParseIP(ips)
		return tIP != nil
	}
	// valid CIDR
	return true
}
