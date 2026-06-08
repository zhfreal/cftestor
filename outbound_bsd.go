//go:build freebsd || openbsd || netbsd || dragonfly

package main

import (
	"fmt"
	"syscall"
)

func validateOutboundPlatformOptions() error {
	if Config.OutboundMarkSet {
		return fmt.Errorf("%q is only supported on Linux", "--mark")
	}
	return nil
}

func outboundInterfaceUsesSourceFallback() bool {
	return true
}

func applyOutboundSocketOptions(network, address string, c syscall.RawConn) error {
	return nil
}
