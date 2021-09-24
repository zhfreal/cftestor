package main

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"log"
	mRand "math/rand"
	"net"
	"time"
)

func ExtractIPCIDRHosts(netw string, num uint) []string {
	// netw: ip range/CIDR or single ip address
	// num: the number of subnet host to return, num = 0 means get all subnet,
	//  but when source is IPv6 CIDR, when mask is less than 64, the subnet hosts would be to large to return, this will have a panic
	// convert string to IPNet struct: 192.168.1.0/24
	ipAddrIP, ipNet, err := net.ParseCIDR(netw)
	if err != nil {
		tIP := net.ParseIP(netw)
		// pure ipv6 address
		if tIP != nil {
			return []string{tIP.String()}
		} else {
			log.Println("invalid IP CIDR address or IP address: ", netw)
			return []string{}
		}
	}
	maskLength, _ := ipNet.Mask.Size()
	// it's a single ip address,
	if maskLength == 128 || maskLength == 32 {
		return []string{ipAddrIP.String()}
	}
	var hosts []string
	// ipv6
	if ipAddrIP.To4() == nil {
		// the last whole byte of host from left to right, whole byte means don't share with subnet
		hostByteInWhole := maskLength / 8
		// mask length great than 64
		if maskLength >= 64 {
			// if mask length is larger than 64 and the amount is less than we expect, then return all hosts
			// when mask length is lager than 64, the amount of hosts can be represented by a uint64 number
			// subnetBytes := ipNet.Mask[hostByteInWhole:]
			// because mask length is larger than 64, the host size transfer a uint64 var
			// subnetSize := binary.BigEndian.Uint64(subnetBytes)
			// the start of subnet
			var start = uint64(0)
			// the end of subnet
			var finish = uint64(1<<(128-maskLength+1) - 1)
			// loop through addresses as uint32.
			// I used "start + 1" and "finish - 1" to discard the network and broadcast addresses.
			for i := start; i <= finish; i++ {
				// convert back to net.IPs
				// Create IP address of type net.IP. IPv4 is 4 bytes, IPv6 is 16 bytes.
				ip := make(net.IP, 16)
				// copy bytes from host from the last byte to hostByteInWhole + 1
				// the next hostByteInWhole byte, ip[hostByteInWhole] may mix host with subnet bits
				copy(ip, ipNet.IP[:hostByteInWhole])
				for x := 15; x > hostByteInWhole; x-- {
					ip[x] = byte(i >> ((15 - x) * 8))
				}
				// the next hostByteInWhole byte, ip[hostByteInWhole] may mix host with subnet bits
				ip[hostByteInWhole] = ipNet.IP[hostByteInWhole] | byte(i>>uint64((15-hostByteInWhole)*8))
				hosts = append(hosts, ip.String())
			}
		} else {
			if num == 0 { // the subnet hosts is so big, can't return the whole subnet
				panic(fmt.Sprintf("source %s is IPv6 CIDR, can't return all subnet", netw))
			}
			// random ipv6 subnet, when subnet hosts greater than num
			for i := uint(0); i < num; i++ {
				ip := make([]byte, 16)
				cache := make([]byte, 16-hostByteInWhole)
				n, err := rand.Read(cache)
				for err != nil && n != 16-hostByteInWhole {
					n, err = rand.Read(cache)
					time.Sleep(time.Duration(100) * time.Millisecond)
				}
				copy(ip, ipNet.IP[:hostByteInWhole])
				copy(ip[hostByteInWhole+1:], cache[1:])
				ip[hostByteInWhole] = ip[hostByteInWhole] | cache[hostByteInWhole]
				hosts = append(hosts, net.IP(ip).String())
			}
		}
	} else { // ipv4
		// convert IPNet struct mask and address to uint32
		mask := binary.BigEndian.Uint32(ipNet.Mask)
		// find the start IP address
		start := binary.BigEndian.Uint32(ipNet.IP)
		// find the final IP address
		finish := (start & mask) | (mask ^ 0xffffffff)
		for i := start; i <= finish; i++ {
			// convert back to net.IPs
			// Create IP address of type net.IP, an IPv4 address is 4 bytes, while IPv6 is 16 bytes.
			ip := make(net.IP, 4)
			binary.BigEndian.PutUint32(ip, i)
			hosts = append(hosts, ip.String())
		}
	}
	// shuffle hosts
	mRand.Shuffle(len(hosts), func(i, j int) {
		hosts[i], hosts[j] = hosts[j], hosts[i]
	})
	if num == 0 || num >= uint(len(hosts)) { // return all hosts
		return hosts
	} else { // shuffle hosts and just return num of hosts
		return hosts[:num]
	}
}

func GetTestIPs(ipList []string, returnNum int) []string {
	var hosts = make([]string, 0)
	//var tHostsLinker = make([] *[] string, 0)
	if len(ipList) == 0 {
		return hosts
	}
	if returnNum <= 0 {
		returnNum = 0
	}
	for _, netIPS := range ipList {
		tList := ExtractIPCIDRHosts(netIPS, uint(returnNum))
		//tHostsLinker = append(tHostsLinker, &tList)
		// there is a bug here, when returnNum is too big
		hosts = append(hosts, tList...)
	}
	mRand.Shuffle(len(hosts), func(i, j int) {
		hosts[i], hosts[j] = hosts[j], hosts[i]
	})
	//if returnNum > 0 && len(hosts) > returnNum{
	//    return hosts[:returnNum]
	//}
	return hosts
}

func IsValidIPs(ips string) bool {
	_, _, err := net.ParseCIDR(ips)
	if err != nil {
		tIP := net.ParseIP(ips)
		// valid IP
		if tIP != nil {
			return true
		}
		// invalid IP , CIDR
		return false
	}
	// valid CIDR
	return true
}
