package ipfilter

import (
	"fmt"
	"net"

	"go.uber.org/zap"
)

type IPFilter struct {
	trustedSubnet *net.IPNet
	logger        *zap.Logger
}

func NewIPFilter(trustedSubnet *net.IPNet, logger *zap.Logger) *IPFilter {
	return &IPFilter{
		trustedSubnet: trustedSubnet,
		logger:        logger,
	}
}

func (f *IPFilter) IsEnabled() bool {
	return f.trustedSubnet != nil
}

func (f *IPFilter) ValidateIP(ipStr string) error {
	if !f.IsEnabled() {
		return nil
	}

	if ipStr == "" {
		f.logger.Warn("IP address is empty")
		return fmt.Errorf("no IP address provided")
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		f.logger.Warn("Invalid IP address", zap.String("ip", ipStr))
		return fmt.Errorf("invalid IP address")
	}

	if !f.trustedSubnet.Contains(ip) {
		f.logger.Warn("IP not in trusted subnet",
			zap.String("ip", ipStr),
			zap.String("trusted_subnet", f.trustedSubnet.String()))
		return fmt.Errorf("IP address not in trusted subnet")
	}

	f.logger.Debug("IP check passed", zap.String("ip", ipStr))
	return nil
}
