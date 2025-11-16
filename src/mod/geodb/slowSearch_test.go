package geodb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIpv4ToUInt32(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected uint32
	}{
		{
			name:     "Minimum IP address",
			ip:       "0.0.0.0",
			expected: 0,
		},
		{
			name:     "Maximum IP address",
			ip:       "255.255.255.255",
			expected: 4294967295,
		},
		{
			name:     "Common IP address",
			ip:       "192.168.1.1",
			expected: 3232235777,
		},
		{
			name:     "Google DNS",
			ip:       "8.8.8.8",
			expected: 134744072,
		},
		{
			name:     "Localhost",
			ip:       "127.0.0.1",
			expected: 2130706433,
		},
		{
			name:     "Example IP 1.2.3.4",
			ip:       "1.2.3.4",
			expected: 16909060,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIP(tt.ip)
			result := ipv4ToUInt32(ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsIPv4InRange(t *testing.T) {
	tests := []struct {
		name        string
		startIP     string
		endIP       string
		testIP      string
		expected    bool
		expectError bool
	}{
		{
			name:        "IP in range - start boundary",
			startIP:     "192.168.1.0",
			endIP:       "192.168.1.255",
			testIP:      "192.168.1.0",
			expected:    true,
			expectError: false,
		},
		{
			name:        "IP in range - end boundary",
			startIP:     "192.168.1.0",
			endIP:       "192.168.1.255",
			testIP:      "192.168.1.255",
			expected:    true,
			expectError: false,
		},
		{
			name:        "IP in range - middle",
			startIP:     "192.168.1.0",
			endIP:       "192.168.1.255",
			testIP:      "192.168.1.128",
			expected:    true,
			expectError: false,
		},
		{
			name:        "IP below range",
			startIP:     "192.168.1.0",
			endIP:       "192.168.1.255",
			testIP:      "192.168.0.255",
			expected:    false,
			expectError: false,
		},
		{
			name:        "IP above range",
			startIP:     "192.168.1.0",
			endIP:       "192.168.1.255",
			testIP:      "192.168.2.0",
			expected:    false,
			expectError: false,
		},
		{
			name:        "Large range - IP in range",
			startIP:     "1.0.0.0",
			endIP:       "1.255.255.255",
			testIP:      "1.128.64.32",
			expected:    true,
			expectError: false,
		},
		{
			name:        "Single IP range",
			startIP:     "8.8.8.8",
			endIP:       "8.8.8.8",
			testIP:      "8.8.8.8",
			expected:    true,
			expectError: false,
		},
		{
			name:        "Invalid start IP",
			startIP:     "invalid",
			endIP:       "192.168.1.255",
			testIP:      "192.168.1.100",
			expected:    false,
			expectError: true,
		},
		{
			name:        "Invalid end IP",
			startIP:     "192.168.1.0",
			endIP:       "invalid",
			testIP:      "192.168.1.100",
			expected:    false,
			expectError: true,
		},
		{
			name:        "Invalid test IP",
			startIP:     "192.168.1.0",
			endIP:       "192.168.1.255",
			testIP:      "invalid",
			expected:    false,
			expectError: true,
		},
		{
			name:        "Empty string IP",
			startIP:     "192.168.1.0",
			endIP:       "192.168.1.255",
			testIP:      "",
			expected:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := isIPv4InRange(tt.startIP, tt.endIP, tt.testIP)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestIsIPv6InRange(t *testing.T) {
	tests := []struct {
		name        string
		startIP     string
		endIP       string
		testIP      string
		expected    bool
		expectError bool
	}{
		{
			name:        "IPv6 in range - start boundary",
			startIP:     "2001:db8::1",
			endIP:       "2001:db8::ffff",
			testIP:      "2001:db8::1",
			expected:    true,
			expectError: false,
		},
		{
			name:        "IPv6 in range - end boundary",
			startIP:     "2001:db8::1",
			endIP:       "2001:db8::ffff",
			testIP:      "2001:db8::ffff",
			expected:    true,
			expectError: false,
		},
		{
			name:        "IPv6 in range - middle",
			startIP:     "2001:db8::1",
			endIP:       "2001:db8::ffff",
			testIP:      "2001:db8::8000",
			expected:    true,
			expectError: false,
		},
		{
			name:        "IPv6 below range",
			startIP:     "2001:db8::1000",
			endIP:       "2001:db8::ffff",
			testIP:      "2001:db8::500",
			expected:    false,
			expectError: false,
		},
		{
			name:        "IPv6 above range",
			startIP:     "2001:db8::1",
			endIP:       "2001:db8::1000",
			testIP:      "2001:db8::2000",
			expected:    false,
			expectError: false,
		},
		{
			name:        "IPv6 single IP range",
			startIP:     "2001:4860:4860::8888",
			endIP:       "2001:4860:4860::8888",
			testIP:      "2001:4860:4860::8888",
			expected:    true,
			expectError: false,
		},
		{
			name:        "IPv6 loopback",
			startIP:     "::1",
			endIP:       "::1",
			testIP:      "::1",
			expected:    true,
			expectError: false,
		},
		{
			name:        "Invalid IPv6 start",
			startIP:     "invalid",
			endIP:       "2001:db8::ffff",
			testIP:      "2001:db8::1",
			expected:    false,
			expectError: true,
		},
		{
			name:        "Invalid IPv6 end",
			startIP:     "2001:db8::1",
			endIP:       "invalid",
			testIP:      "2001:db8::1",
			expected:    false,
			expectError: true,
		},
		{
			name:        "Invalid IPv6 test",
			startIP:     "2001:db8::1",
			endIP:       "2001:db8::ffff",
			testIP:      "invalid",
			expected:    false,
			expectError: true,
		},
		{
			name:        "IPv4-mapped IPv6 address",
			startIP:     "::ffff:192.168.1.0",
			endIP:       "::ffff:192.168.1.255",
			testIP:      "::ffff:192.168.1.100",
			expected:    true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := isIPv6InRange(tt.startIP, tt.endIP, tt.testIP)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestStore_GetSlowSearchCachedIpv4(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Store)
		ip       string
		expected string
	}{
		{
			name: "Cache hit",
			setup: func(s *Store) {
				s.slowLookupCacheIpv4.Store("192.168.1.1", "US")
			},
			ip:       "192.168.1.1",
			expected: "US",
		},
		{
			name:     "Cache miss",
			setup:    func(s *Store) {},
			ip:       "192.168.1.100",
			expected: "",
		},
		{
			name: "Multiple entries in cache",
			setup: func(s *Store) {
				s.slowLookupCacheIpv4.Store("1.1.1.1", "AU")
				s.slowLookupCacheIpv4.Store("8.8.8.8", "US")
				s.slowLookupCacheIpv4.Store("208.67.222.222", "US")
			},
			ip:       "1.1.1.1",
			expected: "AU",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &Store{}
			tt.setup(store)
			result := store.GetSlowSearchCachedIpv4(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStore_GetSlowSearchCachedIpv6(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Store)
		ip       string
		expected string
	}{
		{
			name: "Cache hit",
			setup: func(s *Store) {
				s.slowLookupCacheIpv6.Store("2001:db8::1", "US")
			},
			ip:       "2001:db8::1",
			expected: "US",
		},
		{
			name:     "Cache miss",
			setup:    func(s *Store) {},
			ip:       "2001:db8::ffff",
			expected: "",
		},
		{
			name: "Multiple entries in cache",
			setup: func(s *Store) {
				s.slowLookupCacheIpv6.Store("2001:4860:4860::8888", "US")
				s.slowLookupCacheIpv6.Store("2606:4700:4700::1111", "US")
				s.slowLookupCacheIpv6.Store("2001:db8::1", "AU")
			},
			ip:       "2001:db8::1",
			expected: "AU",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &Store{}
			tt.setup(store)
			result := store.GetSlowSearchCachedIpv6(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStore_SlowSearchIpv4(t *testing.T) {
	// Create a minimal store with test data (using public IPs)
	store := &Store{
		geodb: [][]string{
			{"8.8.8.0", "8.8.8.255", "US"},
			{"1.1.1.0", "1.1.1.255", "AU"},
			{"13.104.0.0", "13.107.255.255", "US"},
			{"104.16.0.0", "104.31.255.255", "US"},
		},
	}

	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{
			name:     "IP in first range",
			ip:       "8.8.8.100",
			expected: "US",
		},
		{
			name:     "IP in second range",
			ip:       "1.1.1.100",
			expected: "AU",
		},
		{
			name:     "IP in third range",
			ip:       "13.105.50.100",
			expected: "US",
		},
		{
			name:     "IP at range boundary",
			ip:       "8.8.8.8",
			expected: "US",
		},
		{
			name:     "IP not in any range",
			ip:       "50.50.50.50",
			expected: "",
		},
		{
			name:     "Reserved IP - localhost",
			ip:       "127.0.0.1",
			expected: "",
		},
		{
			name:     "Reserved IP - link local",
			ip:       "169.254.1.1",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := store.slowSearchIpv4(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStore_SlowSearchIpv4_Caching(t *testing.T) {
	store := &Store{
		geodb: [][]string{
			{"8.8.8.0", "8.8.8.255", "US"},
		},
	}

	// First call should search and cache
	result1 := store.slowSearchIpv4("8.8.8.8")
	assert.Equal(t, "US", result1)

	// Verify it was cached
	cached := store.GetSlowSearchCachedIpv4("8.8.8.8")
	assert.Equal(t, "US", cached)

	// Second call should return from cache
	result2 := store.slowSearchIpv4("8.8.8.8")
	assert.Equal(t, "US", result2)
}

func TestStore_SlowSearchIpv6(t *testing.T) {
	// Create a minimal store with test data
	store := &Store{
		geodbIpv6: [][]string{
			{"2001:db8::1", "2001:db8::ffff", "US"},
			{"2001:4860:4860::8888", "2001:4860:4860::8888", "US"},
			{"2606:4700:4700::1111", "2606:4700:4700::1111", "US"},
		},
	}

	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{
			name:     "IPv6 in range",
			ip:       "2001:db8::100",
			expected: "US",
		},
		{
			name:     "IPv6 single IP match",
			ip:       "2001:4860:4860::8888",
			expected: "US",
		},
		{
			name:     "IPv6 not in any range",
			ip:       "2001:1234::1",
			expected: "",
		},
		{
			name:     "Reserved IPv6 - loopback",
			ip:       "::1",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := store.slowSearchIpv6(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStore_SlowSearchIpv6_Caching(t *testing.T) {
	store := &Store{
		geodbIpv6: [][]string{
			{"2001:db8::1", "2001:db8::ffff", "US"},
		},
	}

	// First call should search and cache
	result1 := store.slowSearchIpv6("2001:db8::100")
	assert.Equal(t, "US", result1)

	// Verify it was cached
	cached := store.GetSlowSearchCachedIpv6("2001:db8::100")
	assert.Equal(t, "US", cached)

	// Second call should return from cache
	result2 := store.slowSearchIpv6("2001:db8::100")
	assert.Equal(t, "US", result2)
}
