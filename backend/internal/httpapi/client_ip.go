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
		return "invalid-peer"
	}
	peer = peer.Unmap()
	if !s.trusted(peer) {
		return peer.String()
	}
	// Only walk a chain supplied by a configured proxy, from the nearest hop back.
	// Malformed/missing metadata falls back to a coarse peer bucket, never unlimited.
	chain := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	if len(chain) > 16 {
		return peer.String()
	}
	for i := len(chain) - 1; i >= 0; i-- {
		address, err := netip.ParseAddr(strings.TrimSpace(chain[i]))
		if err != nil {
			return peer.String()
		}
		address = address.Unmap()
		if !s.trusted(address) {
			return address.String()
		}
	}
	return peer.String()
}
