package netutils_test

import (
	"net/http/httptest"
	"testing"

	"imuslab.com/zoraxy/mod/netutils"

	"github.com/stretchr/testify/assert"
)

// Test GetRequesterIPUntrusted
func TestGetRequesterIPUntrusted(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		expected   string
	}{
		{"IPv4 with port", "192.168.1.100:8080", "192.168.1.100"},
		{"IPv4 without port", "192.168.1.100", "192.168.1.100"},
		{"IPv6 with port", "[2001:db8::1]:8080", "2001:db8::1"},
		{"IPv6 without port", "2001:db8::1", "2001:db8::1"},
		{"localhost IPv4", "127.0.0.1:9000", "127.0.0.1"},
		{"localhost IPv6", "[::1]:9000", "::1"},
		{"invalid IP", "invalid", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			result := netutils.GetRequesterIPUntrusted(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test GetRequesterIP with various headers
func TestGetRequesterIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expected   string
	}{
		{
			"X-Real-IP header",
			"10.0.0.1:8080",
			map[string]string{"X-Real-Ip": "203.0.113.1"},
			"203.0.113.1",
		},
		{
			"CF-Connecting-IP header",
			"10.0.0.1:8080",
			map[string]string{"CF-Connecting-IP": "198.51.100.1"},
			"198.51.100.1",
		},
		{
			"Fastly-Client-IP header",
			"10.0.0.1:8080",
			map[string]string{"Fastly-Client-IP": "198.51.100.2"},
			"198.51.100.2",
		},
		{
			"X-Forwarded-For single IP",
			"10.0.0.1:8080",
			map[string]string{"X-Forwarded-For": "203.0.113.5"},
			"203.0.113.5",
		},
		{
			"X-Forwarded-For multiple IPs",
			"10.0.0.1:8080",
			map[string]string{"X-Forwarded-For": "203.0.113.10,109.21.249.211,10.0.0.5"},
			"203.0.113.10",
		},
		{
			"X-Forwarded-For IPv6 with port",
			"10.0.0.1:8080",
			map[string]string{"X-Forwarded-For": "[2001:db8::1]:443"},
			"2001:db8::1",
		},
		{
			"X-Forwarded-For IPv6 bracketed",
			"10.0.0.1:8080",
			map[string]string{"X-Forwarded-For": "[2001:db8::1]"},
			"2001:db8::1",
		},
		{
			"No headers - use RemoteAddr",
			"192.168.1.100:8080",
			map[string]string{},
			"192.168.1.100",
		},
		{
			"Priority: CF over Fastly",
			"10.0.0.1:8080",
			map[string]string{
				"CF-Connecting-IP": "203.0.113.20",
				"Fastly-Client-IP": "203.0.113.21",
			},
			"203.0.113.20",
		},
		{
			"RemoteAddr with port",
			"127.0.0.1:61001",
			map[string]string{},
			"127.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			result := netutils.GetRequesterIP(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test MatchIpWildcard
func TestMatchIpWildcard(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		wildcard string
		expected bool
	}{
		{"exact match", "192.168.1.100", "192.168.1.100", true},
		{"wildcard last octet", "192.168.1.100", "192.168.1.*", true},
		{"wildcard middle octets", "192.168.1.100", "192.*.1.100", true},
		{"wildcard all octets", "192.168.1.100", "*.*.*.*", true},
		{"no match first octet", "192.168.1.100", "10.168.1.100", false},
		{"no match last octet", "192.168.1.100", "192.168.1.200", false},
		{"partial wildcard match", "192.168.1.100", "192.168.*.*", true},
		{"invalid IP - too few octets", "192.168.1", "192.168.1.*", false},
		{"invalid wildcard - too few octets", "192.168.1.100", "192.168.1", false},
		{"invalid IP - too many octets", "192.168.1.100.5", "192.168.1.100.*", false},
		{"localhost exact", "127.0.0.1", "127.0.0.1", true},
		{"localhost wildcard", "127.0.0.1", "127.0.0.*", true},
		{"zero IP", "0.0.0.0", "0.0.0.0", true},
		{"zero wildcard", "192.168.1.1", "0.0.0.*", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := netutils.MatchIpWildcard(tt.ip, tt.wildcard)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test MatchIpCIDR
func TestMatchIpCIDR(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		cidr     string
		expected bool
	}{
		// IPv4 tests
		{"IPv4 in range /24", "192.168.1.100", "192.168.1.0/24", true},
		{"IPv4 exact match /32", "192.168.1.100", "192.168.1.100/32", true},
		{"IPv4 not in range", "192.168.2.100", "192.168.1.0/24", false},
		{"IPv4 broader range /16", "192.168.50.100", "192.168.0.0/16", true},
		{"IPv4 narrower range /28", "192.168.1.15", "192.168.1.0/28", true},
		{"IPv4 out of narrow range", "192.168.1.16", "192.168.1.0/28", false},
		{"IPv4 localhost in range", "127.0.0.1", "127.0.0.0/8", true},
		{"IPv4 private range 10/8", "10.50.100.200", "10.0.0.0/8", true},
		{"IPv4 private range 172.16/12", "172.20.10.1", "172.16.0.0/12", true},

		// IPv6 tests
		{"IPv6 in range /64", "2001:db8::1", "2001:db8::/64", true},
		{"IPv6 exact match /128", "2001:db8::1", "2001:db8::1/128", true},
		{"IPv6 not in range", "2001:db9::1", "2001:db8::/64", false},
		{"IPv6 broader range /32", "2001:db8:1234:5678::1", "2001:db8::/32", true},
		{"IPv6 localhost", "::1", "::1/128", true},
		{"IPv6 with scope ID", "fe80::1%eth0", "fe80::/10", true},

		// Invalid CIDR
		{"invalid CIDR format", "192.168.1.1", "192.168.1.0", false},
		{"invalid CIDR range", "192.168.1.1", "192.168.1.0/33", false},
		{"empty CIDR", "192.168.1.1", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := netutils.MatchIpCIDR(tt.ip, tt.cidr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test IsPrivateIP
func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		// IPv4 loopback
		{"IPv4 localhost", "127.0.0.1", true},

		// IPv6 loopback
		{"IPv6 localhost", "::1", true},

		// IPv4 private ranges
		{"IPv4 10.0.0.0/8", "10.0.0.1", true},
		{"IPv4 10.x.x.x", "10.255.255.255", true},
		{"IPv4 172.16.0.0/12", "172.16.0.1", true},
		{"IPv4 172.20.x.x", "172.20.10.1", true},
		{"IPv4 172.31.x.x", "172.31.255.255", true},
		{"IPv4 192.168.0.0/16", "192.168.1.1", true},
		{"IPv4 192.168.x.x", "192.168.255.255", true},

		// IPv4 public IPs
		{"IPv4 public Google DNS", "8.8.8.8", false},
		{"IPv4 public Cloudflare", "1.1.1.1", false},
		{"IPv4 public 203.x.x.x", "203.0.113.1", false},

		// IPv6 private ranges
		{"IPv6 unique local fc00::/7", "fc00::1", true},
		{"IPv6 unique local fd00::/8", "fd00::1", true},
		{"IPv6 link-local fe80::/10", "fe80::1", true},
		{"IPv6 link-local with interface", "fe80::abcd:ef01:2345:6789", true},

		// IPv6 public IPs
		{"IPv6 public 2001::", "2001:db8::1", false},
		{"IPv6 public Google DNS", "2001:4860:4860::8888", false},

		// Invalid IPs
		{"empty string", "", false},
		{"invalid IP", "invalid", false},
		{"malformed IP", "256.256.256.256", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := netutils.IsPrivateIP(tt.ip)
			assert.Equal(t, tt.expected, result, "IP: %s", tt.ip)
		})
	}
}

// Test IsIPv6
func TestIsIPv6(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"valid IPv6", "2001:db8::1", true},
		{"valid IPv6 localhost", "::1", true},
		{"valid IPv6 full", "2001:0db8:0000:0000:0000:0000:0000:0001", true},
		{"valid IPv6 compressed", "2001:db8::8a2e:370:7334", true},
		{"valid IPv6 link-local", "fe80::1", true},
		{"IPv4 address", "192.168.1.1", false},
		{"IPv4 localhost", "127.0.0.1", false},
		{"IPv4-mapped IPv6", "::ffff:192.168.1.1", false}, // This is IPv4-mapped
		{"invalid IP", "invalid", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := netutils.IsIPv6(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test IsIPv4
func TestIsIPv4(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"valid IPv4", "192.168.1.1", true},
		{"valid IPv4 localhost", "127.0.0.1", true},
		{"valid IPv4 zero", "0.0.0.0", true},
		{"valid IPv4 broadcast", "255.255.255.255", true},
		{"valid IPv4 public", "8.8.8.8", true},
		{"IPv6 address", "2001:db8::1", false},
		{"IPv6 localhost", "::1", false},
		{"IPv4-mapped IPv6", "::ffff:192.168.1.1", true}, // ParseIP handles this
		{"invalid IP", "invalid", false},
		{"empty string", "", false},
		{"malformed IPv4", "256.1.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := netutils.IsIPv4(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test GetRequesterIP edge cases
func TestGetRequesterIPEdgeCases(t *testing.T) {
	t.Run("Multiple X-Forwarded-For entries", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.0.0.1:8080"
		req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1, 192.168.1.1")
		result := netutils.GetRequesterIP(req)
		assert.Equal(t, "203.0.113.1", result)
	})

	t.Run("IPv6 with brackets in X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.0.0.1:8080"
		req.Header.Set("X-Forwarded-For", "[2001:db8::1]")
		result := netutils.GetRequesterIP(req)
		assert.Equal(t, "2001:db8::1", result)
	})

	t.Run("IPv6 without brackets in X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "10.0.0.1:8080"
		req.Header.Set("X-Forwarded-For", "2001:db8::1")
		result := netutils.GetRequesterIP(req)
		assert.Equal(t, "2001:db8::1", result)
	})
}

// Test MatchIpCIDR with scope IDs
func TestMatchIpCIDRWithScopeID(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		cidr     string
		expected bool
	}{
		{"IPv6 with scope ID in range", "fe80::1%eth0", "fe80::/10", true},
		{"IPv6 with scope ID out of range", "fe80::1%eth0", "2001::/16", false},
		{"IPv6 with different scope ID", "fe80::abcd%wlan0", "fe80::/10", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := netutils.MatchIpCIDR(tt.ip, tt.cidr)
			assert.Equal(t, tt.expected, result)
		})
	}
}
