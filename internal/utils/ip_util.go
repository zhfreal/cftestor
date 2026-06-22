package utils

import (
	"fmt"
	"math/big"
	"math/rand"
	"net"
)

type IPRange struct {
	IPStart   net.IP
	IPEnd     net.IP
	Len       *big.Int
	Extracted bool
}

func (ipr *IPRange) isValid() bool {
	if ipr == nil || ipr.IPStart == nil || ipr.IPEnd == nil || ipr.Extracted {
		return false
	}
	if len(ipr.IPStart) != len(ipr.IPEnd) {
		return false
	} else if len(ipr.IPStart) != net.IPv4len && len(ipr.IPStart) != net.IPv6len {
		return false
	} else {
		for i := 0; i < len(ipr.IPStart); i++ {
			if (ipr.IPStart)[i] > (ipr.IPEnd)[i] {
				return false
			}
		}
	}
	return true
}

func (ipr *IPRange) IsValid() bool {
	return ipr.isValid()
}

func (ipr *IPRange) length() *big.Int {
	if !ipr.isValid() {
		return big.NewInt(0)
	}
	var newLenBytes = make([]byte, len(ipr.IPEnd), cap(ipr.IPEnd))
	reduce := 0
	for i := len(ipr.IPStart) - 1; i >= 0; i-- {
		m := (ipr.IPStart)[i]
		n := (ipr.IPEnd)[i]
		newValue := int(n) - int(m) - reduce
		// n < m + reduce, borrow from i - 1
		if newValue < 0 {
			reduce = 1
			newValue += int(1 << 8)
		} else {
			// reset reduce
			reduce = 0
		}
		newLenBytes[i] = byte(newValue)
	}
	newLen := big.NewInt(0).SetBytes(newLenBytes)
	// add 1 more
	newLen = newLen.Add(newLen, big.NewInt(1))
	return newLen
}

func (ipr *IPRange) Length() *big.Int {
	return ipr.length()
}

func (ipr *IPRange) isV4() bool {
	if !ipr.isValid() {
		return false
	}
	return len(ipr.IPStart) == net.IPv4len
}

func (ipr *IPRange) isV6() bool {
	if !ipr.isValid() {
		return false
	}
	return len(ipr.IPStart) == net.IPv6len
}

func (ipr *IPRange) IsV4() bool {
	return ipr.isV4()
}

func (ipr *IPRange) IsV6() bool {
	return ipr.isV6()
}

func (ipr *IPRange) init(StartIP net.IP, EndIP net.IP) *IPRange {
	t_s_startIP := StartIP
	if t_s_startIP.To4() != nil {
		t_s_startIP = t_s_startIP.To4()
	}
	t_s_endIP := EndIP
	if t_s_endIP.To4() != nil {
		t_s_endIP = t_s_endIP.To4()
	}
	ipr.IPStart = t_s_startIP
	ipr.IPEnd = t_s_endIP

	ipr.Extracted = false
	if ipr.isValid() {
		ipr.Len = ipr.length()
		return ipr
	}
	return nil
}

func (ipr *IPRange) String() string {
	if !ipr.isValid() {
		return "null"
	}
	return fmt.Sprintf("Start With: %s; End With: %s; Length: %s; Extracted: %t",
		(ipr.IPStart).String(), (ipr.IPEnd).String(), (ipr.length()).String(), ipr.Extracted)
}

func (ipr *IPRange) Extract(num int) (IPList []net.IP) {
	if !ipr.isValid() || num <= 0 || ipr.Extracted || ipr.Len.Cmp(big.NewInt(0)) == 0 {
		return
	}
	numBig := big.NewInt(int64(num))
	if ipr.Len.Cmp(numBig) == -1 {
		num = int(ipr.Len.Int64())
		numBig = big.NewInt(int64(num))
	}

	for i := 0; i < num; i++ {
		n := big.NewInt(int64(i))
		num_in_bytes := fillBytes(n.Bytes(), len(ipr.IPStart))
		newIP := ipShift(ipr.IPStart, num_in_bytes)
		if newIP != nil {
			IPList = append(IPList, newIP)
		}
	}

	// reset IPStart and Extracted
	if numBig.Cmp(ipr.Len) == 0 {
		ipr.Extracted = true
		ipr.Len = big.NewInt(0)
		ipr.IPStart = ipr.IPEnd
	} else {
		num_in_bytes := fillBytes(numBig.Bytes(), len(ipr.IPStart))
		ipr.IPStart = ipShift(ipr.IPStart, num_in_bytes)
		ipr.Len = ipr.length()
	}
	return
}

func (ipr *IPRange) ExtractReverse(num int) (IPList []net.IP) {
	if !ipr.isValid() || num <= 0 || ipr.Extracted || ipr.Len.Cmp(big.NewInt(0)) == 0 {
		return
	}
	numBig := big.NewInt(int64(num))
	if ipr.Len.Cmp(numBig) == -1 {
		num = int(ipr.Len.Int64())
		numBig = big.NewInt(int64(num))
	}

	for i := 0; i < num; i++ {
		n := big.NewInt(int64(i))
		num_in_bytes := fillBytes(n.Bytes(), len(ipr.IPEnd))
		newIP := ipShiftReverse(ipr.IPEnd, num_in_bytes)
		if newIP != nil {
			IPList = append(IPList, newIP)
		}
	}

	// reset IPEnd and Extracted
	if numBig.Cmp(ipr.Len) == 0 {
		ipr.Extracted = true
		ipr.Len = big.NewInt(0)
		ipr.IPEnd = ipr.IPStart
	} else {
		num_in_bytes := fillBytes(numBig.Bytes(), len(ipr.IPEnd))
		ipr.IPEnd = ipShiftReverse(ipr.IPEnd, num_in_bytes)
		ipr.Len = ipr.length()
	}
	return
}

func (ipr *IPRange) ExtractAll(maxHostLen int) (IPList []net.IP) {
	// we limit the max result length to MaxHostLen (currently, 65536), if it's to big, return nil
	// or it's don't have any IPS to extract, return nil
	if ipr.Extracted || ipr.Len.Cmp(big.NewInt(0)) == 0 || ipr.Len.Cmp(big.NewInt(int64(maxHostLen))) == 1 {
		return
	}
	return ipr.Extract(int(ipr.Len.Int64()))
}

func (ipr *IPRange) GetRandomX(r *rand.Rand, num int) (IPList []net.IP) {
	// or it's don't have any IPS to extract, return nil
	if ipr.Extracted || ipr.Len.Cmp(big.NewInt(0)) == 0 {
		return
	}
	// we extract all while ipr don't have enough ips for extracted
	if big.NewInt(int64(num)).Cmp(ipr.Len) >= 0 {
		m := ipr.Extract(int(ipr.Len.Int64()))
		if m == nil {
			return
		}
		IPList = append(IPList, m...)
		// shuffle
		r.Shuffle(len(IPList), func(i, j int) {
			IPList[i], IPList[j] = IPList[j], IPList[i]
		})
		// we done here
		return
	}
	// get randomly
	i := 0
	for i < num {
		n := big.NewInt(0)
		n = n.Rand(r, ipr.Len)
		num_in_bytes := fillBytes(n.Bytes(), len(ipr.IPStart))
		newIP := ipShift(ipr.IPStart, num_in_bytes)
		if newIP != nil {
			IPList = append(IPList, newIP)
			i++
		}
	}
	return
}

func NewIPRangeFromIP(StartIP net.IP, EndIP net.IP) *IPRange {
	return new(IPRange).init(StartIP, EndIP)
}

func NewIPRangeFromString(StartIPStr *string, EndIPStr *string) *IPRange {
	StartIP := net.ParseIP(*StartIPStr)
	EndIP := net.ParseIP(*EndIPStr)
	return new(IPRange).init(StartIP, EndIP)
}

func NewIPRangeFromCIDR(cidr *string) *IPRange {
	ip, ipCIDR, err := net.ParseCIDR(*cidr)
	if err != nil {
		tIP := net.ParseIP(*cidr)
		if tIP != nil {
			return new(IPRange).init(tIP, tIP)
		}
		return nil
	}
	if !ip.Equal(ipCIDR.IP) {
		return new(IPRange).init(ip, ip)
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
	return new(IPRange).init(StartIP, EndIP)
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
	if num > 0 {
		return nil
	}
	return newBytes
}

func ipShift(ip net.IP, num []byte) net.IP {
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
	if reduce > 0 {
		return nil
	}
	return newIP
}
