package geodb

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIpToBytes(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected int // length of bytes
	}{
		{
			name:     "IPv4 address",
			ip:       "192.168.1.1",
			expected: 4,
		},
		{
			name:     "IPv4 localhost",
			ip:       "127.0.0.1",
			expected: 4,
		},
		{
			name:     "IPv4 zero",
			ip:       "0.0.0.0",
			expected: 4,
		},
		{
			name:     "IPv4 max",
			ip:       "255.255.255.255",
			expected: 4,
		},
		{
			name:     "IPv6 address",
			ip:       "2001:db8::1",
			expected: 16,
		},
		{
			name:     "IPv6 loopback",
			ip:       "::1",
			expected: 16,
		},
		{
			name:     "IPv6 full format",
			ip:       "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			expected: 16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ipToBytes(tt.ip)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expected, len(result))
		})
	}
}

func TestIpToBytes_Values(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected []byte
	}{
		{
			name:     "Simple IPv4",
			ip:       "1.2.3.4",
			expected: []byte{1, 2, 3, 4},
		},
		{
			name:     "IPv4 localhost",
			ip:       "127.0.0.1",
			expected: []byte{127, 0, 0, 1},
		},
		{
			name:     "IPv4 zero",
			ip:       "0.0.0.0",
			expected: []byte{0, 0, 0, 0},
		},
		{
			name:     "IPv4 max",
			ip:       "255.255.255.255",
			expected: []byte{255, 255, 255, 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ipToBytes(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewTrie(t *testing.T) {
	tr := newTrie()
	assert.NotNil(t, tr)
	assert.NotNil(t, tr.root)
	assert.Equal(t, "", tr.root.cc)
}

func TestTrie_Insert(t *testing.T) {
	tr := newTrie()

	tests := []struct {
		name string
		ip   string
		cc   string
	}{
		{
			name: "Insert IPv4 US",
			ip:   "8.8.8.8",
			cc:   "US",
		},
		{
			name: "Insert IPv4 CN",
			ip:   "1.2.3.4",
			cc:   "CN",
		},
		{
			name: "Insert IPv4 JP",
			ip:   "192.168.1.1",
			cc:   "JP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			assert.NotPanics(t, func() {
				tr.insert(tt.ip, tt.cc)
			})
		})
	}
}

func TestTrie_InsertAndSearch(t *testing.T) {
	tests := []struct {
		name       string
		insertData []struct {
			startIP string
			endIP   string
			cc      string
		}
		searchIP   string
		expectedCC string
	}{
		{
			name: "Single range middle IP",
			insertData: []struct {
				startIP string
				endIP   string
				cc      string
			}{
				{"1.0.16.0", "1.0.31.255", "JP"},
			},
			searchIP:   "1.0.20.100",
			expectedCC: "JP",
		},
		{
			name: "Multiple ranges",
			insertData: []struct {
				startIP string
				endIP   string
				cc      string
			}{
				{"1.0.16.0", "1.0.31.255", "JP"},
				{"1.0.32.0", "1.0.63.255", "CN"},
				{"1.0.64.0", "1.0.127.255", "JP"},
			},
			searchIP:   "1.0.50.100",
			expectedCC: "CN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := newTrie()
			for _, data := range tt.insertData {
				tr.insert(data.startIP, data.cc)
				tr.insert(data.endIP, data.cc)
			}
			result := tr.search(tt.searchIP)
			assert.Equal(t, tt.expectedCC, result)
		})
	}
}

func TestTrie_Search(t *testing.T) {
	// Build a trie with known data
	tr := newTrie()
	testData := [][]string{
		{"1.0.16.0", "1.0.31.255", "JP"},
		{"1.0.32.0", "1.0.63.255", "CN"},
		{"1.0.64.0", "1.0.127.255", "JP"},
		{"1.0.128.0", "1.0.255.255", "TH"},
		{"1.1.0.0", "1.1.0.255", "CN"},
		{"1.1.1.0", "1.1.1.255", "AU"},
		{"1.1.2.0", "1.1.63.255", "CN"},
	}

	for _, entry := range testData {
		tr.insert(entry[0], entry[2])
		tr.insert(entry[1], entry[2])
	}

	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{
			name:     "IP in JP range",
			ip:       "1.0.16.20",
			expected: "JP",
		},
		{
			name:     "IP in CN range",
			ip:       "1.0.50.100",
			expected: "CN",
		},
		{
			name:     "IP in TH range",
			ip:       "1.0.200.100",
			expected: "TH",
		},
		{
			name:     "IP in AU range",
			ip:       "1.1.1.100",
			expected: "AU",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tr.search(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrie_SearchReservedIPs(t *testing.T) {
	tr := newTrie()
	tr.insert("127.0.0.0", "US")
	tr.insert("127.255.255.255", "US")

	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{
			name:     "Loopback IP",
			ip:       "127.0.0.1",
			expected: "",
		},
		{
			name:     "Private IP 192.168.x.x",
			ip:       "192.168.1.1",
			expected: "",
		},
		{
			name:     "Private IP 10.x.x.x",
			ip:       "10.0.0.1",
			expected: "",
		},
		{
			name:     "Private IP 172.16.x.x",
			ip:       "172.16.0.1",
			expected: "",
		},
		{
			name:     "Link local",
			ip:       "169.254.1.1",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tr.search(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrie_SearchIPv6(t *testing.T) {
	tr := newTrie()

	// Insert IPv6 ranges
	ipv6Data := [][]string{
		{"2001:200::", "2001:200:ffff:ffff:ffff:ffff:ffff:ffff", "JP"},
	}

	for _, entry := range ipv6Data {
		tr.insert(entry[0], entry[2])
		tr.insert(entry[1], entry[2])
	}

	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{
			name:     "IPv6 in JP range",
			ip:       "2001:200::1234",
			expected: "JP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tr.search(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsReservedIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{
			name:     "Loopback IPv4",
			ip:       "127.0.0.1",
			expected: true,
		},
		{
			name:     "Loopback IPv4 other",
			ip:       "127.255.255.255",
			expected: true,
		},
		{
			name:     "Private IPv4 - 192.168.x.x",
			ip:       "192.168.1.1",
			expected: true,
		},
		{
			name:     "Private IPv4 - 10.x.x.x",
			ip:       "10.0.0.1",
			expected: true,
		},
		{
			name:     "Private IPv4 - 172.16.x.x",
			ip:       "172.16.0.1",
			expected: true,
		},
		{
			name:     "Private IPv4 - 172.31.x.x",
			ip:       "172.31.255.254",
			expected: true,
		},
		{
			name:     "Link local IPv4",
			ip:       "169.254.1.1",
			expected: true,
		},
		{
			name:     "Public IPv4 - Google DNS",
			ip:       "8.8.8.8",
			expected: false,
		},
		{
			name:     "Public IPv4 - Cloudflare DNS",
			ip:       "1.1.1.1",
			expected: false,
		},
		{
			name:     "Public IPv4 - example",
			ip:       "93.184.216.34",
			expected: false,
		},
		{
			name:     "Loopback IPv6",
			ip:       "::1",
			expected: true,
		},
		{
			name:     "Link local IPv6",
			ip:       "fe80::1",
			expected: true,
		},
		{
			name:     "Unique local IPv6",
			ip:       "fd00::1",
			expected: true,
		},
		{
			name:     "Public IPv6",
			ip:       "2001:4860:4860::8888",
			expected: false,
		},
		{
			name:     "Public IPv6 Cloudflare",
			ip:       "2606:4700:4700::1111",
			expected: false,
		},
		{
			name:     "Invalid IP",
			ip:       "invalid",
			expected: false,
		},
		{
			name:     "Empty string",
			ip:       "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isReservedIP(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to parse IP for testing
func parseIP(ip string) net.IP {
	return net.ParseIP(ip)
}

func TestTrie_LargeDataset(t *testing.T) {
	tr := newTrie()

	// Insert a large number of IP ranges
	ranges := [][]string{
		{"1.0.0.0", "1.0.255.255", "AU"},
		{"1.1.0.0", "1.1.255.255", "CN"},
		{"8.8.8.0", "8.8.8.255", "US"},
		{"13.104.0.0", "13.107.255.255", "US"},
		{"104.16.0.0", "104.31.255.255", "US"},
		{"185.199.108.0", "185.199.111.255", "US"},
	}

	for _, r := range ranges {
		tr.insert(r[0], r[2])
		tr.insert(r[1], r[2])
	}

	// Test various IPs
	tests := []struct {
		ip       string
		expected string
	}{
		{"1.0.128.100", "AU"},
		{"1.1.100.200", "CN"},
		{"8.8.8.8", "US"},
		{"13.105.100.100", "US"},
		{"104.20.50.100", "US"},
		{"185.199.109.100", "US"},
		{"192.168.1.1", ""}, // Private IP
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			result := tr.search(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTrie_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		setup  func() *trie
		testIP string
		expect string
	}{
		{
			name: "Empty trie",
			setup: func() *trie {
				return newTrie()
			},
			testIP: "8.8.8.8",
			expect: "",
		},
		{
			name: "Entire IPv4 space",
			setup: func() *trie {
				tr := newTrie()
				tr.insert("0.0.0.0", "XX")
				tr.insert("255.255.255.255", "XX")
				return tr
			},
			testIP: "123.45.67.89",
			expect: "XX",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := tt.setup()
			result := tr.search(tt.testIP)
			assert.Equal(t, tt.expect, result)
		})
	}
}
