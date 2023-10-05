package middleware

import (
	"net"
	"net/http"
)

func CheckCIDR(h http.Handler, CIDR string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if CIDR == "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		realIP := r.Header.Get("X-Real-IP")

		_, subnet, err := net.ParseCIDR(CIDR)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		parsedIP := net.ParseIP(realIP)
		if parsedIP == nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !subnet.Contains(parsedIP) {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		h.ServeHTTP(w, r)
	})
}
