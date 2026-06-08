package main

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall"
)

func prepareOutboundOptions(opts *cliOptions) error {
	if err := prepareOutboundMark(opts); err != nil {
		return err
	}
	if len(Config.OutboundInterface) > 0 {
		if err := prepareOutboundInterface(); err != nil {
			return err
		}
	}
	return validateOutboundPlatformOptions()
}

func prepareOutboundMark(opts *cliOptions) error {
	var mark uint32
	markSet := false

	if opts.MarkChanged {
		v, err := parseOutboundMark("--mark", opts.Mark)
		if err != nil {
			return err
		}
		mark = v
		markSet = true
	}
	if opts.XMarkChanged {
		v, err := parseOutboundMark("--xmark", opts.XMark)
		if err != nil {
			return err
		}
		if markSet && mark != v {
			return fmt.Errorf("%q and %q cannot set different mark values", "--mark", "--xmark")
		}
		mark = v
		markSet = true
	}
	if markSet {
		Config.OutboundMark = mark
		Config.OutboundMarkSet = true
	}
	return nil
}

func parseOutboundMark(flagName, raw string) (uint32, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("invalid value for %q: must be a decimal or hex mark in range 0..0xffffffff", flagName)
	}
	parsed, err := strconv.ParseUint(value, 0, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid value for %q: must be a decimal or hex mark in range 0..0xffffffff", flagName)
	}
	return uint32(parsed), nil
}

func prepareOutboundInterface() error {
	raw := strings.TrimSpace(Config.OutboundInterface)
	if raw == "" {
		return nil
	}
	if ip, zone, ok := parseOutboundSourceIP(raw); ok {
		if err := validateLocalSourceIP(ip, zone); err != nil {
			return err
		}
		Config.OutboundSourceIP = ip
		Config.OutboundSourceZone = zone
		return nil
	}

	if index, ok, err := parseOutboundInterfaceIndex(raw); err != nil {
		return err
	} else if ok {
		iface, err := net.InterfaceByIndex(index)
		if err != nil {
			return fmt.Errorf("invalid value for %q: interface index %d was not found", "--interface", index)
		}
		Config.OutboundInterfaceIndex = iface.Index
		Config.OutboundInterfaceName = iface.Name
		return nil
	}

	iface, err := net.InterfaceByName(raw)
	if err != nil {
		return fmt.Errorf("invalid value for %q: interface %q was not found", "--interface", raw)
	}
	Config.OutboundInterfaceIndex = iface.Index
	Config.OutboundInterfaceName = iface.Name
	return nil
}

func parseOutboundSourceIP(raw string) (net.IP, string, bool) {
	candidate := raw
	zone := ""
	if before, after, ok := strings.Cut(raw, "%"); ok {
		candidate = before
		zone = after
	}
	ip := net.ParseIP(candidate)
	if ip == nil {
		return nil, "", false
	}
	return normalizeOutboundIP(ip), zone, true
}

func parseOutboundInterfaceIndex(raw string) (int, bool, error) {
	if raw == "" {
		return 0, false, nil
	}
	parsed, err := strconv.ParseUint(raw, 10, 31)
	if err != nil {
		if allDecimalDigits(raw) {
			return 0, false, fmt.Errorf("invalid value for %q: interface index %q is out of range", "--interface", raw)
		}
		return 0, false, nil
	}
	if parsed == 0 {
		return 0, false, fmt.Errorf("invalid value for %q: interface index must be greater than 0", "--interface")
	}
	return int(parsed), true, nil
}

func allDecimalDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func validateLocalSourceIP(ip net.IP, zone string) error {
	if ip == nil || ip.IsUnspecified() {
		return fmt.Errorf("invalid value for %q: source IP must be assigned to a local interface", "--interface")
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("failed to list local interfaces for %q validation: %w", "--interface", err)
	}
	for _, iface := range ifaces {
		if !interfaceMatchesZone(iface, zone) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			addrIP := ipFromAddr(addr)
			if addrIP != nil && addrIP.Equal(ip) {
				return nil
			}
		}
	}
	if zone != "" {
		return fmt.Errorf("invalid value for %q: source IP %s%%%s is not assigned to a local interface", "--interface", ip, zone)
	}
	return fmt.Errorf("invalid value for %q: source IP %s is not assigned to a local interface", "--interface", ip)
}

func interfaceMatchesZone(iface net.Interface, zone string) bool {
	if zone == "" {
		return true
	}
	return zone == iface.Name || zone == strconv.Itoa(iface.Index)
}

func outboundDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if Config.OutboundInterfaceIndex > 0 && outboundInterfaceUsesSourceFallback() {
		return dialWithInterfaceSourceFallback(ctx, network, address)
	}
	dialer, err := newOutboundDialer(network, nil, "")
	if err != nil {
		return nil, err
	}
	return dialer.DialContext(ctx, network, address)
}

func newOutboundDialer(network string, sourceIP net.IP, sourceZone string) (*net.Dialer, error) {
	dialer := &net.Dialer{}
	if sourceIP == nil {
		sourceIP = Config.OutboundSourceIP
		sourceZone = Config.OutboundSourceZone
	}
	if sourceIP != nil {
		localIP, err := sourceIPForNetwork(network, sourceIP)
		if err != nil {
			return nil, err
		}
		dialer.LocalAddr = &net.TCPAddr{IP: localIP, Port: 0, Zone: sourceZone}
	}
	if needsOutboundSocketControl() {
		dialer.ControlContext = func(ctx context.Context, network, address string, c syscall.RawConn) error {
			return applyOutboundSocketOptions(network, address, c)
		}
	}
	return dialer, nil
}

func needsOutboundSocketControl() bool {
	return Config.OutboundMarkSet || (Config.OutboundInterfaceIndex > 0 && !outboundInterfaceUsesSourceFallback())
}

func sourceIPForNetwork(network string, ip net.IP) (net.IP, error) {
	family := networkAddressFamily(network)
	switch family {
	case 4:
		v4 := ip.To4()
		if v4 == nil {
			return nil, fmt.Errorf("source IP %s is not compatible with IPv4 network %q", ip, network)
		}
		return v4, nil
	case 6:
		if ip.To4() != nil {
			return nil, fmt.Errorf("source IP %s is not compatible with IPv6 network %q", ip, network)
		}
		return ip, nil
	default:
		return ip, nil
	}
}

func networkAddressFamily(network string) int {
	if strings.HasSuffix(network, "4") {
		return 4
	}
	if strings.HasSuffix(network, "6") {
		return 6
	}
	return 0
}

func dialWithInterfaceSourceFallback(ctx context.Context, network, address string) (net.Conn, error) {
	targets, err := resolveOutboundDialTargets(ctx, network, address)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for _, target := range targets {
		sourceIP, zone, err := sourceIPFromInterface(target.network)
		if err != nil {
			lastErr = err
			continue
		}
		dialer, err := newOutboundDialer(target.network, sourceIP, zone)
		if err != nil {
			lastErr = err
			continue
		}
		conn, err := dialer.DialContext(ctx, target.network, target.address)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no usable dial targets for %q", address)
	}
	return nil, lastErr
}

type outboundDialTarget struct {
	network string
	address string
}

func resolveOutboundDialTargets(ctx context.Context, network, address string) ([]outboundDialTarget, error) {
	if networkAddressFamily(network) != 0 {
		return []outboundDialTarget{{network: network, address: address}}, nil
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return []outboundDialTarget{{network: network, address: address}}, nil
	}
	if ip, _, ok := parseOutboundSourceIP(host); ok {
		return []outboundDialTarget{{network: familyNetwork(network, ip), address: address}}, nil
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %q for %q interface fallback: %w", host, "--interface", err)
	}
	targets := make([]outboundDialTarget, 0, len(ips))
	for _, ipAddr := range ips {
		ip := normalizeOutboundIP(ipAddr.IP)
		if ip == nil {
			continue
		}
		targets = append(targets, outboundDialTarget{
			network: familyNetwork(network, ip),
			address: net.JoinHostPort(ip.String(), port),
		})
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("failed to resolve %q to an IPv4 or IPv6 address", host)
	}
	return targets, nil
}

func familyNetwork(network string, ip net.IP) string {
	base := network
	if base == "" || base == "tcp" || base == "tcp4" || base == "tcp6" {
		base = "tcp"
	}
	if ip.To4() != nil {
		return strings.TrimSuffix(strings.TrimSuffix(base, "4"), "6") + "4"
	}
	return strings.TrimSuffix(strings.TrimSuffix(base, "4"), "6") + "6"
}

func sourceIPFromInterface(network string) (net.IP, string, error) {
	family := networkAddressFamily(network)
	if family == 0 {
		return nil, "", fmt.Errorf("cannot choose source address for network %q", network)
	}
	iface, err := net.InterfaceByIndex(Config.OutboundInterfaceIndex)
	if err != nil {
		return nil, "", fmt.Errorf("failed to resolve interface index %d: %w", Config.OutboundInterfaceIndex, err)
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, "", fmt.Errorf("failed to list addresses for interface %q: %w", iface.Name, err)
	}
	for _, addr := range addrs {
		ip := normalizeOutboundIP(ipFromAddr(addr))
		if ip == nil || ip.IsUnspecified() {
			continue
		}
		if family == 4 && ip.To4() != nil {
			return ip.To4(), "", nil
		}
		if family == 6 && ip.To4() == nil {
			zone := ""
			if ip.IsLinkLocalUnicast() {
				zone = iface.Name
			}
			return ip, zone, nil
		}
	}
	return nil, "", fmt.Errorf("interface %q has no IPv%d address for %q fallback", iface.Name, family, "--interface")
}

func ipFromAddr(addr net.Addr) net.IP {
	switch a := addr.(type) {
	case *net.IPNet:
		return a.IP
	case *net.IPAddr:
		return a.IP
	default:
		return nil
	}
}

func normalizeOutboundIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}
	if v4 := ip.To4(); v4 != nil {
		return v4
	}
	return ip
}
