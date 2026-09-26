package httpapi

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

func (s *Server) trusted(address netip.Addr) bool {
	for _, prefix := range s.TrustedProxies {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func (s *Server) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return ""
	}
	peer = peer.Unmap()
	if !s.trusted(peer) {
		return peer.String()
	}
	// Only walk a chain supplied by a configured proxy, from the nearest hop back.
	// A trusted peer without client metadata uses only shared account throttling.
	chain := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	if len(chain) > 16 {
		return ""
	}
	for i := len(chain) - 1; i >= 0; i-- {
		address, err := netip.ParseAddr(strings.TrimSpace(chain[i]))
		if err != nil {
			return ""
		}
		address = address.Unmap()
		if !s.trusted(address) {
			return address.String()
		}
	}
	return ""
}
