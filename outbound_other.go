//go:build !linux && !windows && !darwin && !solaris && !freebsd && !openbsd && !netbsd && !dragonfly

package main

import (
	"fmt"
	"runtime"
	"syscall"
)

func validateOutboundPlatformOptions() error {
	if Config.OutboundMarkSet {
		return fmt.Errorf("%q is only supported on Linux", "--mark")
	}
	if Config.OutboundInterfaceIndex > 0 {
		return fmt.Errorf("%q with interface name or index is not supported on %s; use a local source IP instead", "--interface", runtime.GOOS)
	}
	return nil
}

func outboundInterfaceUsesSourceFallback() bool {
	return false
}

func applyOutboundSocketOptions(network, address string, c syscall.RawConn) error {
	return nil
}
