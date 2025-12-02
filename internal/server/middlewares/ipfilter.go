package middlewares

import (
	"net"
	"net/http"

	"go.uber.org/zap"
)

type IPFilterHandler struct {
	logger        *zap.Logger
	trustedSubnet *net.IPNet
}

func NewIPFilterHandler(logger *zap.Logger, trustedSubnet *net.IPNet) *IPFilterHandler {
	return &IPFilterHandler{
		logger:        logger,
		trustedSubnet: trustedSubnet,
	}
}

func (ipf *IPFilterHandler) Middleware(next http.Handler) http.Handler {
	if ipf.trustedSubnet == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		realIP := r.Header.Get("X-Real-IP")
		if realIP == "" {
			ipf.logger.Warn("X-Real-IP header is missing")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		ip := net.ParseIP(realIP)
		if ip == nil {
			ipf.logger.Warn("invalid IP address in X-Real-IP header", zap.String("ip", realIP))
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		if !ipf.trustedSubnet.Contains(ip) {
			ipf.logger.Warn(
				"IP address not in trusted subnet",
				zap.String("ip", realIP),
				zap.String("trusted_subnet", ipf.trustedSubnet.String()),
			)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
