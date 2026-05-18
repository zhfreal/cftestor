package main

import (
	"net"
)

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
