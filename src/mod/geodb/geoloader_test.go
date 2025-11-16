package geodb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCSV(t *testing.T) {
	tests := []struct {
		name        string
		csvContent  []byte
		expectedLen int
		wantErr     bool
	}{
		{
			name: "Valid CSV with single entry",
			csvContent: []byte(`192.168.1.0,192.168.1.255,US
`),
			expectedLen: 1,
			wantErr:     false,
		},
		{
			name: "Valid CSV with multiple entries",
			csvContent: []byte(`192.168.1.0,192.168.1.255,US
10.0.0.0,10.255.255.255,CN
172.16.0.0,172.31.255.255,JP
`),
			expectedLen: 3,
			wantErr:     false,
		},
		{
			name: "CSV with no trailing newline",
			csvContent: []byte(`192.168.1.0,192.168.1.255,US
10.0.0.0,10.255.255.255,CN`),
			expectedLen: 2,
			wantErr:     false,
		},
		{
			name:        "Empty CSV",
			csvContent:  []byte(``),
			expectedLen: 0,
			wantErr:     false,
		},
		{
			name: "CSV with spaces",
			csvContent: []byte(`192.168.1.0, 192.168.1.255, US
`),
			expectedLen: 1,
			wantErr:     false,
		},
		{
			name: "CSV with IPv6 addresses",
			csvContent: []byte(`2001:db8::1,2001:db8::ffff,US
2606:4700:4700::1111,2606:4700:4700::1111,US
`),
			expectedLen: 2,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseCSV(tt.csvContent)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedLen, len(result))
			}
		})
	}
}

func TestParseCSV_FieldCount(t *testing.T) {
	csvContent := []byte(`192.168.1.0,192.168.1.255,US
10.0.0.0,10.255.255.255,CN
172.16.0.0,172.31.255.255,JP
`)

	result, err := parseCSV(csvContent)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(result))

	// Verify each record has 3 fields
	for i, record := range result {
		assert.Equal(t, 3, len(record), "Record %d should have 3 fields", i)
	}
}

func TestParseCSV_Values(t *testing.T) {
	csvContent := []byte(`1.0.16.0,1.0.31.255,JP
8.8.8.0,8.8.8.255,US
`)

	result, err := parseCSV(csvContent)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result))

	// Verify first record
	assert.Equal(t, "1.0.16.0", result[0][0])
	assert.Equal(t, "1.0.31.255", result[0][1])
	assert.Equal(t, "JP", result[0][2])

	// Verify second record
	assert.Equal(t, "8.8.8.0", result[1][0])
	assert.Equal(t, "8.8.8.255", result[1][1])
	assert.Equal(t, "US", result[1][2])
}

func TestConstructTrieTree(t *testing.T) {
	tests := []struct {
		name     string
		data     [][]string
		testIP   string
		expected string
	}{
		{
			name: "Single range",
			data: [][]string{
				{"8.8.8.0", "8.8.8.255", "US"},
			},
			testIP:   "8.8.8.100",
			expected: "US",
		},
		{
			name: "Multiple ranges",
			data: [][]string{
				{"1.0.16.0", "1.0.31.255", "JP"},
				{"1.0.32.0", "1.0.63.255", "CN"},
				{"1.0.64.0", "1.0.127.255", "JP"},
			},
			testIP:   "1.0.50.100",
			expected: "CN",
		},
		{
			name: "IPv6 ranges",
			data: [][]string{
				{"2001:db8::1", "2001:db8::ffff", "US"},
			},
			testIP:   "2001:db8::100",
			expected: "US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trie := constrctTrieTree(tt.data)
			assert.NotNil(t, trie)

			// Test search
			result := trie.search(tt.testIP)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConstructTrieTree_LargeDataset(t *testing.T) {
	// Create a larger dataset
	data := [][]string{
		{"1.0.0.0", "1.0.255.255", "AU"},
		{"1.1.0.0", "1.1.255.255", "CN"},
		{"8.8.8.0", "8.8.8.255", "US"},
		{"13.104.0.0", "13.107.255.255", "US"},
		{"104.16.0.0", "104.31.255.255", "US"},
		{"185.199.108.0", "185.199.111.255", "US"},
	}

	trie := constrctTrieTree(data)
	assert.NotNil(t, trie)

	// Test various IPs in the constructed trie
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
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			result := trie.search(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConstructTrieTree_EmptyData(t *testing.T) {
	data := [][]string{}
	trie := constrctTrieTree(data)
	assert.NotNil(t, trie)

	// Search should return empty for any IP
	result := trie.search("8.8.8.8")
	assert.Equal(t, "", result)
}

func TestStore_Search(t *testing.T) {
	// Create a minimal store for testing
	store := &Store{
		geodb: [][]string{
			{"8.8.8.0", "8.8.8.255", "US"},
			{"1.1.1.0", "1.1.1.255", "CN"},
		},
		geodbIpv6: [][]string{
			{"2001:db8::1", "2001:db8::ffff", "US"},
		},
	}

	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{
			name:     "IPv4 in range",
			ip:       "8.8.8.100",
			expected: "US",
		},
		{
			name:     "IPv4 not in range",
			ip:       "50.50.50.50",
			expected: "",
		},
		{
			name:     "IPv6 in range",
			ip:       "2001:db8::100",
			expected: "US",
		},
		{
			name:     "IPv6 not in range",
			ip:       "2001:4860:4860::8888",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := store.search(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStore_Search_CloudflareFormat(t *testing.T) {
	// Create a minimal store for testing
	store := &Store{
		geodb: [][]string{
			{"8.8.8.0", "8.8.8.255", "US"},
		},
	}

	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{
			name:     "Cloudflare proxied format",
			ip:       "8.8.8.8, 172.71.139.178",
			expected: "US",
		},
		{
			name:     "Cloudflare proxied format with extra spaces",
			ip:       "8.8.8.8,  172.70.100.1",
			expected: "US",
		},
		{
			name:     "Multiple IPs - uses first one",
			ip:       "8.8.8.8, 1.1.1.1, 192.168.1.1",
			expected: "US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := store.search(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStore_Search_WithTrie(t *testing.T) {
	// Create store with trie
	data := [][]string{
		{"1.0.16.0", "1.0.31.255", "JP"},
		{"1.0.32.0", "1.0.63.255", "CN"},
		{"1.0.64.0", "1.0.127.255", "JP"},
	}

	store := &Store{
		geodb:   data,
		geotrie: constrctTrieTree(data),
	}

	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{
			name:     "IP in JP range",
			ip:       "1.0.20.100",
			expected: "JP",
		},
		{
			name:     "IP in CN range",
			ip:       "1.0.50.100",
			expected: "CN",
		},
		{
			name:     "IP in CN range middle",
			ip:       "1.0.40.50",
			expected: "CN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := store.search(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStore_Search_WithoutTrie(t *testing.T) {
	// Create store without trie (slow search)
	data := [][]string{
		{"1.0.16.0", "1.0.31.255", "JP"},
		{"1.0.32.0", "1.0.63.255", "CN"},
	}

	store := &Store{
		geodb:   data,
		geotrie: nil, // No trie, will use slow search
	}

	tests := []struct {
		name     string
		ip       string
		expected string
	}{
		{
			name:     "IP in JP range",
			ip:       "1.0.20.100",
			expected: "JP",
		},
		{
			name:     "IP in CN range",
			ip:       "1.0.50.100",
			expected: "CN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := store.search(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStore_Search_IPv6WithTrie(t *testing.T) {
	// Create store with IPv6 trie
	dataIpv6 := [][]string{
		{"2001:db8::1", "2001:db8::ffff", "US"},
		{"2606:4700:4700::1000", "2606:4700:4700::2000", "US"},
	}

	store := &Store{
		geodbIpv6:   dataIpv6,
		geotrieIpv6: constrctTrieTree(dataIpv6),
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
			name:     "IPv6 in second range",
			ip:       "2606:4700:4700::1500",
			expected: "US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := store.search(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStore_Search_IPv6WithoutTrie(t *testing.T) {
	// Create store without IPv6 trie (slow search)
	dataIpv6 := [][]string{
		{"2001:db8::1", "2001:db8::ffff", "US"},
	}

	store := &Store{
		geodbIpv6:   dataIpv6,
		geotrieIpv6: nil, // No trie, will use slow search
	}

	result := store.search("2001:db8::100")
	assert.Equal(t, "US", result)
}
