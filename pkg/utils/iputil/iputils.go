package iputil

import (
	"fmt"
	"net"
	"net/http"
)

const (
	XForwardedFor = "X-Forwarded-For"
	XRealIP       = "X-Real-IP"
	XClientIP     = "x-client-ip"
)

func RemoteIp(req *http.Request) string {
	remoteAddr := req.RemoteAddr
	if ip := req.Header.Get(XClientIP); ip != "" {
		remoteAddr = ip
	} else if ip := req.Header.Get(XRealIP); ip != "" {
		remoteAddr = ip
	} else if ip = req.Header.Get(XForwardedFor); ip != "" {
		remoteAddr = ip
	} else {
		remoteAddr, _, _ = net.SplitHostPort(remoteAddr)
	}

	if remoteAddr == "::1" {
		remoteAddr = "127.0.0.1"
	}

	return remoteAddr
}

// ParseIP parse an ip address and return whether it is a v4 or v6 ip address
func ParseIP(s string) (net.IP, int, error) {
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, 0, fmt.Errorf("%s is not a valid ip address", s)
	}
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.':
			return ip, 4, nil
		case ':':
			return ip, 6, nil
		}
	}
	return nil, 0, fmt.Errorf("%s is not a valid ip address", s)
}
