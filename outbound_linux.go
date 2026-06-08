//go:build linux

package main

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

func validateOutboundPlatformOptions() error {
	return nil
}

func outboundInterfaceUsesSourceFallback() bool {
	return false
}

func applyOutboundSocketOptions(network, address string, c syscall.RawConn) error {
	var controlErr error
	err := c.Control(func(fd uintptr) {
		if Config.OutboundMarkSet {
			if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_MARK, int(Config.OutboundMark)); err != nil {
				controlErr = fmt.Errorf("set SO_MARK: %w", err)
				return
			}
		}
		if Config.OutboundInterfaceIndex > 0 {
			if err := unix.BindToDevice(int(fd), Config.OutboundInterfaceName); err != nil {
				controlErr = fmt.Errorf("bind socket to interface %q: %w", Config.OutboundInterfaceName, err)
				return
			}
		}
	})
	if err != nil {
		return err
	}
	return controlErr
}
