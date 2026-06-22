//go:build windows

package outbound

import (
	"fmt"
	"syscall"

	"cftestor/internal/config"
	"golang.org/x/sys/windows"
)

const (
	windowsIPUnicastIF   = 31
	windowsIPv6UnicastIF = 31
)

func validateOutboundPlatformOptions() error {
	if config.Config.OutboundMarkSet {
		return fmt.Errorf("%q is only supported on Linux", "--mark")
	}
	return nil
}

func outboundInterfaceUsesSourceFallback() bool {
	return false
}

func applyOutboundSocketOptions(network, address string, c syscall.RawConn) error {
	if config.Config.OutboundInterfaceIndex == 0 {
		return nil
	}
	var controlErr error
	err := c.Control(func(fd uintptr) {
		family := networkAddressFamily(network)
		switch family {
		case 4:
			controlErr = windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IP, windowsIPUnicastIF, htonl(config.Config.OutboundInterfaceIndex))
		case 6:
			controlErr = windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IPV6, windowsIPv6UnicastIF, config.Config.OutboundInterfaceIndex)
		default:
			controlErr = fmt.Errorf("cannot determine socket family %q for %q", network, "--interface")
		}
	})
	if err != nil {
		return err
	}
	if controlErr != nil {
		return fmt.Errorf("bind socket to interface index %d: %w", config.Config.OutboundInterfaceIndex, controlErr)
	}
	return nil
}

func htonl(value int) int {
	u := uint32(value)
	return int((u&0xff)<<24 | (u&0xff00)<<8 | (u&0xff0000)>>8 | (u&0xff000000)>>24)
}
