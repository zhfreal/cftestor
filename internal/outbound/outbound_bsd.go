//go:build freebsd || openbsd || netbsd || dragonfly

package outbound

import (
	"fmt"
	"syscall"

	"cftestor/internal/config"
)

func validateOutboundPlatformOptions() error {
	if config.Config.OutboundMarkSet {
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
