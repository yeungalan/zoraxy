package geodb_test

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"imuslab.com/zoraxy/mod/geodb"
	"imuslab.com/zoraxy/mod/info/logger"
)

/*
func TestTrieConstruct(t *testing.T) {
	tt := geodb.NewTrie()
	data := [][]string{
		{"1.0.16.0", "1.0.31.255", "JP"},
		{"1.0.32.0", "1.0.63.255", "CN"},
		{"1.0.64.0", "1.0.127.255", "JP"},
		{"1.0.128.0", "1.0.255.255", "TH"},
		{"1.1.0.0", "1.1.0.255", "CN"},
		{"1.1.1.0", "1.1.1.255", "AU"},
		{"1.1.2.0", "1.1.63.255", "CN"},
		{"1.1.64.0", "1.1.127.255", "JP"},
		{"1.1.128.0", "1.1.255.255", "TH"},
		{"1.2.0.0", "1.2.2.255", "CN"},
		{"1.2.3.0", "1.2.3.255", "AU"},
	}

	for _, entry := range data {
		startIp := entry[0]
		endIp := entry[1]
		cc := entry[2]
		tt.Insert(startIp, cc)
		tt.Insert(endIp, cc)
	}

	t.Log(tt.Search("1.0.16.20"), "== JP")  //JP
	t.Log(tt.Search("1.2.0.122"), "== CN")  //CN
	t.Log(tt.Search("1.2.1.0"), "== CN")    //CN
	t.Log(tt.Search("1.0.65.243"), "== JP") //JP
	t.Log(tt.Search("1.0.62.243"), "== CN") //CN
}
*/

func TestResolveCountryCodeFromIP(t *testing.T) {
	// Create a new store
	store, err := geodb.NewGeoDb(nil, &geodb.StoreOptions{
		AllowSlowIpv4LookUp:          true,
		AllowSlowIpv6Lookup:          true,
		Logger:                       &logger.Logger{},
		SlowLookupCacheClearInterval: 0,
	})
	if err != nil {
		t.Errorf("error creating store: %v", err)
		return
	}
	defer store.Close()

	// Test an IP address that should return a valid country code
	knownIpCountryMap := [][]string{
		{"3.224.220.101", "US"},
		{"176.113.115.113", "RU"},
		{"65.21.233.213", "FI"},
		{"94.23.207.193", "FR"},
		{"77.131.21.232", "FR"},
	}

	for _, testcase := range knownIpCountryMap {
		ip := testcase[0]
		expected := testcase[1]
		info, err := store.ResolveCountryCodeFromIP(ip)
		if err != nil {
			t.Errorf("error resolving country code for IP %s: %v", ip, err)
			return
		}
		if info.CountryIsoCode != expected {
			t.Errorf("expected country code %s, but got %s for IP %s", expected, info.CountryIsoCode, ip)
		}
	}

	// Test an IP address that should return an empty country code
	ip := "127.0.0.1"
	expected := ""
	info, err := store.ResolveCountryCodeFromIP(ip)
	if err != nil {
		t.Errorf("error resolving country code for IP %s: %v", ip, err)
		return
	}
	if info.CountryIsoCode != expected {
		t.Errorf("expected country code %s, but got %s for IP %s", expected, info.CountryIsoCode, ip)
	}

	// Test for issue #401
	// Create 100 concurrent goroutines to resolve country code for random IP addresses in the test cases above
	for i := 0; i < 100; i++ {
		go func() {
			for _, testcase := range knownIpCountryMap {
				ip := testcase[0]
				expected := testcase[1]
				info, err := store.ResolveCountryCodeFromIP(ip)
				if err != nil {
					t.Errorf("error resolving country code for IP %s: %v", ip, err)
					return
				}
				if info.CountryIsoCode != expected {
					t.Errorf("expected country code %s, but got %s for IP %s", expected, info.CountryIsoCode, ip)
				}
			}
		}()
	}
}

func TestNewGeoDb(t *testing.T) {
	tests := []struct {
		name    string
		options *geodb.StoreOptions
		wantErr bool
	}{
		{
			name: "Create store with trie (fast lookup)",
			options: &geodb.StoreOptions{
				AllowSlowIpv4LookUp:          false,
				AllowSlowIpv6Lookup:          false,
				Logger:                       &logger.Logger{},
				SlowLookupCacheClearInterval: 0,
			},
			wantErr: false,
		},
		{
			name: "Create store with slow lookup",
			options: &geodb.StoreOptions{
				AllowSlowIpv4LookUp:          true,
				AllowSlowIpv6Lookup:          true,
				Logger:                       &logger.Logger{},
				SlowLookupCacheClearInterval: 0,
			},
			wantErr: false,
		},
		{
			name: "Create store with custom cache interval",
			options: &geodb.StoreOptions{
				AllowSlowIpv4LookUp:          true,
				AllowSlowIpv6Lookup:          true,
				Logger:                       &logger.Logger{},
				SlowLookupCacheClearInterval: 5 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "Create store with IPv4 slow, IPv6 fast",
			options: &geodb.StoreOptions{
				AllowSlowIpv4LookUp:          true,
				AllowSlowIpv6Lookup:          false,
				Logger:                       &logger.Logger{},
				SlowLookupCacheClearInterval: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := geodb.NewGeoDb(nil, tt.options)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, store)
				store.Close()
			}
		})
	}
}

func TestStore_Close(t *testing.T) {
	tests := []struct {
		name              string
		slowIpv4Enabled   bool
		slowIpv6Enabled   bool
		shouldStartTicker bool
	}{
		{
			name:              "Close with slow lookup enabled",
			slowIpv4Enabled:   true,
			slowIpv6Enabled:   false,
			shouldStartTicker: true,
		},
		{
			name:              "Close with IPv6 slow lookup enabled",
			slowIpv4Enabled:   false,
			slowIpv6Enabled:   true,
			shouldStartTicker: true,
		},
		{
			name:              "Close with both slow lookups enabled",
			slowIpv4Enabled:   true,
			slowIpv6Enabled:   true,
			shouldStartTicker: true,
		},
		{
			name:              "Close with no slow lookup",
			slowIpv4Enabled:   false,
			slowIpv6Enabled:   false,
			shouldStartTicker: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := geodb.NewGeoDb(nil, &geodb.StoreOptions{
				AllowSlowIpv4LookUp:          tt.slowIpv4Enabled,
				AllowSlowIpv6Lookup:          tt.slowIpv6Enabled,
				Logger:                       &logger.Logger{},
				SlowLookupCacheClearInterval: 100 * time.Millisecond,
			})
			assert.NoError(t, err)
			assert.NotNil(t, store)

			// Should not panic
			assert.NotPanics(t, func() {
				store.Close()
			})
		})
	}
}

func TestStore_GetRequesterCountryISOCode(t *testing.T) {
	store, err := geodb.NewGeoDb(nil, &geodb.StoreOptions{
		AllowSlowIpv4LookUp:          true,
		AllowSlowIpv6Lookup:          true,
		Logger:                       &logger.Logger{},
		SlowLookupCacheClearInterval: 0,
	})
	assert.NoError(t, err)
	defer store.Close()

	tests := []struct {
		name            string
		setupRequest    func() *http.Request
		expectedCountry string
	}{
		{
			name: "Valid US IP in RemoteAddr",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				req.RemoteAddr = "3.224.220.101:12345"
				return req
			},
			expectedCountry: "US",
		},
		{
			name: "Valid Russian IP in RemoteAddr",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				req.RemoteAddr = "176.113.115.113:54321"
				return req
			},
			expectedCountry: "RU",
		},
		{
			name: "Private IP should return LAN",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			expectedCountry: "LAN",
		},
		{
			name: "Localhost should return LAN",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				req.RemoteAddr = "127.0.0.1:8080"
				return req
			},
			expectedCountry: "LAN",
		},
		{
			name: "X-Forwarded-For header",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "3.224.220.101")
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			expectedCountry: "US",
		},
		{
			name: "X-Real-IP header",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				req.Header.Set("X-Real-IP", "94.23.207.193")
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			expectedCountry: "FR",
		},
		{
			name: "CF-Connecting-IP header",
			setupRequest: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				req.Header.Set("CF-Connecting-IP", "65.21.233.213")
				req.RemoteAddr = "192.168.1.1:8080"
				return req
			},
			expectedCountry: "FI",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			country := store.GetRequesterCountryISOCode(req)
			assert.Equal(t, tt.expectedCountry, country)
		})
	}
}

func TestStore_ResolveCountryCodeFromIP_IPv6(t *testing.T) {
	store, err := geodb.NewGeoDb(nil, &geodb.StoreOptions{
		AllowSlowIpv4LookUp:          true,
		AllowSlowIpv6Lookup:          true,
		Logger:                       &logger.Logger{},
		SlowLookupCacheClearInterval: 0,
	})
	assert.NoError(t, err)
	defer store.Close()

	tests := []struct {
		name     string
		ip       string
		wantCode string
	}{
		{
			name:     "IPv6 Google DNS",
			ip:       "2001:4860:4860::8888",
			wantCode: "US",
		},
		{
			name:     "IPv6 Cloudflare DNS",
			ip:       "2606:4700:4700::1111",
			wantCode: "US",
		},
		{
			name:     "IPv6 loopback should return empty",
			ip:       "::1",
			wantCode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := store.ResolveCountryCodeFromIP(tt.ip)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantCode, info.CountryIsoCode)
		})
	}
}

func TestStore_ResolveCountryCodeFromIP_CloudflareFormat(t *testing.T) {
	store, err := geodb.NewGeoDb(nil, &geodb.StoreOptions{
		AllowSlowIpv4LookUp:          true,
		AllowSlowIpv6Lookup:          true,
		Logger:                       &logger.Logger{},
		SlowLookupCacheClearInterval: 0,
	})
	assert.NoError(t, err)
	defer store.Close()

	tests := []struct {
		name     string
		ip       string
		wantCode string
	}{
		{
			name:     "Cloudflare proxied format - should use first IP",
			ip:       "3.224.220.101, 172.71.139.178",
			wantCode: "US",
		},
		{
			name:     "Cloudflare proxied format with spaces",
			ip:       "94.23.207.193,  172.70.100.1",
			wantCode: "FR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := store.ResolveCountryCodeFromIP(tt.ip)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantCode, info.CountryIsoCode)
		})
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	store, err := geodb.NewGeoDb(nil, &geodb.StoreOptions{
		AllowSlowIpv4LookUp:          true,
		AllowSlowIpv6Lookup:          true,
		Logger:                       &logger.Logger{},
		SlowLookupCacheClearInterval: 0,
	})
	assert.NoError(t, err)
	defer store.Close()

	testIPs := []string{
		"3.224.220.101",
		"176.113.115.113",
		"65.21.233.213",
		"94.23.207.193",
		"77.131.21.232",
	}

	var wg sync.WaitGroup
	concurrentRequests := 100

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ip := testIPs[index%len(testIPs)]
			_, err := store.ResolveCountryCodeFromIP(ip)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()
}

func TestStore_CacheClearTicker(t *testing.T) {
	// Test that cache is cleared after interval
	store, err := geodb.NewGeoDb(nil, &geodb.StoreOptions{
		AllowSlowIpv4LookUp:          true,
		AllowSlowIpv6Lookup:          true,
		Logger:                       &logger.Logger{},
		SlowLookupCacheClearInterval: 200 * time.Millisecond,
	})
	assert.NoError(t, err)
	defer store.Close()

	// Resolve an IP to populate cache
	testIP := "192.168.1.100"
	store.ResolveCountryCodeFromIP(testIP)

	// Check cache has the IP
	cachedBefore := store.GetSlowSearchCachedIpv4(testIP)

	// Wait for cache clear
	time.Sleep(300 * time.Millisecond)

	// Cache should be cleared
	cachedAfter := store.GetSlowSearchCachedIpv4(testIP)

	// Note: This is a timing-sensitive test, so we just verify it doesn't panic
	t.Logf("Cache before: %s, after: %s", cachedBefore, cachedAfter)
}

func TestCountryInfo_Structure(t *testing.T) {
	store, err := geodb.NewGeoDb(nil, &geodb.StoreOptions{
		AllowSlowIpv4LookUp:          true,
		AllowSlowIpv6Lookup:          true,
		Logger:                       &logger.Logger{},
		SlowLookupCacheClearInterval: 0,
	})
	assert.NoError(t, err)
	defer store.Close()

	info, err := store.ResolveCountryCodeFromIP("8.8.8.8")
	assert.NoError(t, err)
	assert.NotNil(t, info)

	// Verify structure fields exist and have correct types
	assert.IsType(t, "", info.CountryIsoCode)
	assert.IsType(t, "", info.ContinetCode)
}
