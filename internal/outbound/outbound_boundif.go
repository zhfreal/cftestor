//go:build darwin || solaris

package outbound

import (
	"fmt"
	"syscall"

	"cftestor/internal/config"
	"golang.org/x/sys/unix"
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
			controlErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_BOUND_IF, config.Config.OutboundInterfaceIndex)
		case 6:
			controlErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IPV6, unix.IPV6_BOUND_IF, config.Config.OutboundInterfaceIndex)
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
