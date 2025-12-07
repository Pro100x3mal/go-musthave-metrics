package middlewares

import (
	"net/http"

	"github.com/Pro100x3mal/go-musthave-metrics/internal/server/infrastructure/ipfilter"
)

type IPFilterHandler struct {
	filter *ipfilter.IPFilter
}

func NewIPFilterHandler(filter *ipfilter.IPFilter) *IPFilterHandler {
	return &IPFilterHandler{
		filter: filter,
	}
}

func (ipf *IPFilterHandler) Middleware(next http.Handler) http.Handler {
	if !ipf.filter.IsEnabled() {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		realIP := r.Header.Get("X-Real-IP")
		if err := ipf.filter.ValidateIP(realIP); err != nil {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
