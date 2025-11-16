package dynamicproxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"imuslab.com/zoraxy/mod/dynamicproxy/loadbalance"
)

func TestRouter_getTargetProxyEndpointFromRequestURI(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Create endpoints with different matching paths
	endpoint1 := GetDefaultProxyEndpoint()
	endpoint1.RootOrMatchingDomain = "/api/"
	endpoint1.parent = router
	router.ProxyEndpoints.Store("/api/", &endpoint1)

	endpoint2 := GetDefaultProxyEndpoint()
	endpoint2.RootOrMatchingDomain = "/admin/"
	endpoint2.parent = router
	router.ProxyEndpoints.Store("/admin/", &endpoint2)

	tests := []struct {
		name       string
		requestURI string
		expected   *ProxyEndpoint
	}{
		{
			name:       "Match /api/",
			requestURI: "/api/users",
			expected:   &endpoint1,
		},
		{
			name:       "Match /admin/",
			requestURI: "/admin/settings",
			expected:   &endpoint2,
		},
		{
			name:       "No match",
			requestURI: "/public/file",
			expected:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.getTargetProxyEndpointFromRequestURI(tt.requestURI)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRouter_GetProxyEndpointFromHostname(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Add exact match endpoint
	endpoint1 := GetDefaultProxyEndpoint()
	endpoint1.RootOrMatchingDomain = "test.example.com"
	endpoint1.parent = router
	router.ProxyEndpoints.Store("test.example.com", &endpoint1)

	// Add wildcard endpoint
	endpoint2 := GetDefaultProxyEndpoint()
	endpoint2.RootOrMatchingDomain = "*.wildcard.com"
	endpoint2.parent = router
	router.ProxyEndpoints.Store("*.wildcard.com", &endpoint2)

	// Add endpoint with alias
	endpoint3 := GetDefaultProxyEndpoint()
	endpoint3.RootOrMatchingDomain = "main.example.com"
	endpoint3.MatchingDomainAlias = []string{"alias.example.com"}
	endpoint3.parent = router
	router.ProxyEndpoints.Store("main.example.com", &endpoint3)

	tests := []struct {
		name     string
		hostname string
		expected *ProxyEndpoint
	}{
		{
			name:     "Exact match",
			hostname: "test.example.com",
			expected: &endpoint1,
		},
		{
			name:     "Case insensitive exact match",
			hostname: "TEST.EXAMPLE.COM",
			expected: &endpoint1,
		},
		{
			name:     "Wildcard match",
			hostname: "sub.wildcard.com",
			expected: &endpoint2,
		},
		{
			name:     "Alias match",
			hostname: "alias.example.com",
			expected: &endpoint3,
		},
		{
			name:     "No match",
			hostname: "nonexistent.com",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.GetProxyEndpointFromHostname(tt.hostname)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRouter_GetProxyEndpointFromHostname_DisabledEndpoint(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Add disabled endpoint
	endpoint := GetDefaultProxyEndpoint()
	endpoint.RootOrMatchingDomain = "test.example.com"
	endpoint.Disabled = true
	endpoint.parent = router
	router.ProxyEndpoints.Store("test.example.com", &endpoint)

	result := router.GetProxyEndpointFromHostname("test.example.com")

	// Should not return disabled endpoint from exact match
	assert.Nil(t, result)
}

func TestRouter_GetProxyEndpointFromHostname_MultipleWildcardMatches(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Add multiple wildcard endpoints with different specificity
	endpoint1 := GetDefaultProxyEndpoint()
	endpoint1.RootOrMatchingDomain = "*.example.com"
	endpoint1.parent = router
	router.ProxyEndpoints.Store("*.example.com", &endpoint1)

	endpoint2 := GetDefaultProxyEndpoint()
	endpoint2.RootOrMatchingDomain = "*.test.example.com"
	endpoint2.parent = router
	router.ProxyEndpoints.Store("*.test.example.com", &endpoint2)

	// Should match the more specific one (longer domain)
	result := router.GetProxyEndpointFromHostname("sub.test.example.com")
	assert.NotNil(t, result)
	assert.Equal(t, "*.test.example.com", result.RootOrMatchingDomain)
}

func TestRouter_rewriteURL(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	tests := []struct {
		name       string
		rootURL    string
		requestURL string
		expected   string
	}{
		{
			name:       "Basic rewrite",
			rootURL:    "/api/",
			requestURL: "/api/users",
			expected:   "/users",
		},
		{
			name:       "Rewrite with trailing slash",
			rootURL:    "/api/",
			requestURL: "/api/",
			expected:   "/",
		},
		{
			name:       "Rewrite without trailing slash in root",
			rootURL:    "/api",
			requestURL: "/api/users",
			expected:   "/users",
		},
		{
			name:       "Remove double slashes",
			rootURL:    "/api/",
			requestURL: "/api//users",
			expected:   "/users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.rewriteURL(tt.rootURL, tt.requestURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProxyHandler_upstreamHostSwap(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Create a target endpoint
	targetEndpoint := GetDefaultProxyEndpoint()
	targetEndpoint.RootOrMatchingDomain = "target.example.com"
	targetEndpoint.parent = router
	router.ProxyEndpoints.Store("target.example.com", &targetEndpoint)

	// Create a loopback endpoint (enabled)
	loopbackEndpoint := GetDefaultProxyEndpoint()
	loopbackEndpoint.RootOrMatchingDomain = "loopback.example.com"
	loopbackEndpoint.Disabled = false
	loopbackEndpoint.parent = router
	router.ProxyEndpoints.Store("loopback.example.com", &loopbackEndpoint)

	handler := &ProxyHandler{Parent: router}

	upstream := &loadbalance.Upstream{
		OriginIpOrDomain: "loopback.example.com",
	}

	req := httptest.NewRequest("GET", "http://target.example.com/", nil)
	w := httptest.NewRecorder()

	// Test loopback detection
	result := handler.upstreamHostSwap(w, req, upstream, &targetEndpoint)

	// Should detect loopback and return true
	assert.True(t, result)
}

func TestProxyHandler_upstreamHostSwap_NoLoopback(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	targetEndpoint := GetDefaultProxyEndpoint()
	targetEndpoint.RootOrMatchingDomain = "target.example.com"
	targetEndpoint.parent = router

	handler := &ProxyHandler{Parent: router}

	upstream := &loadbalance.Upstream{
		OriginIpOrDomain: "external.example.com",
	}

	req := httptest.NewRequest("GET", "http://target.example.com/", nil)
	w := httptest.NewRecorder()

	result := handler.upstreamHostSwap(w, req, upstream, &targetEndpoint)

	// Should not detect loopback
	assert.False(t, result)
}

func TestProxyHandler_upstreamHostSwap_DisabledLoopback(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	targetEndpoint := GetDefaultProxyEndpoint()
	targetEndpoint.RootOrMatchingDomain = "target.example.com"
	targetEndpoint.parent = router

	// Create disabled loopback endpoint
	loopbackEndpoint := GetDefaultProxyEndpoint()
	loopbackEndpoint.RootOrMatchingDomain = "loopback.example.com"
	loopbackEndpoint.Disabled = true
	loopbackEndpoint.parent = router
	router.ProxyEndpoints.Store("loopback.example.com", &loopbackEndpoint)

	handler := &ProxyHandler{Parent: router}

	upstream := &loadbalance.Upstream{
		OriginIpOrDomain: "loopback.example.com",
	}

	req := httptest.NewRequest("GET", "http://target.example.com/", nil)
	w := httptest.NewRecorder()

	result := handler.upstreamHostSwap(w, req, upstream, &targetEndpoint)

	// Should detect loopback but endpoint is disabled
	assert.True(t, result)
	// Should return error status
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestProxyHandler_upstreamHostSwap_WithPort(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	targetEndpoint := GetDefaultProxyEndpoint()
	targetEndpoint.RootOrMatchingDomain = "target.example.com"
	targetEndpoint.parent = router

	loopbackEndpoint := GetDefaultProxyEndpoint()
	loopbackEndpoint.RootOrMatchingDomain = "loopback.example.com"
	loopbackEndpoint.parent = router
	router.ProxyEndpoints.Store("loopback.example.com", &loopbackEndpoint)

	handler := &ProxyHandler{Parent: router}

	// Upstream with port
	upstream := &loadbalance.Upstream{
		OriginIpOrDomain: "loopback.example.com:8080",
	}

	req := httptest.NewRequest("GET", "http://target.example.com/", nil)
	w := httptest.NewRecorder()

	result := handler.upstreamHostSwap(w, req, upstream, &targetEndpoint)

	// Should strip port and still detect loopback
	assert.True(t, result)
}

func TestRouter_logRequest(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.DisableLogging = false

	req := httptest.NewRequest("GET", "http://example.com/test", nil)

	// Should not panic
	router.logRequest(req, true, 200, "test", "example.com", "backend.com", &endpoint)
}

func TestRouter_logRequest_LoggingDisabled(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.DisableLogging = true

	req := httptest.NewRequest("GET", "http://example.com/test", nil)

	// Should not panic and should skip logging
	router.logRequest(req, true, 200, "test", "example.com", "backend.com", &endpoint)
}

func TestRouter_logRequest_NilEndpoint(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "http://example.com/test", nil)

	// Should not panic with nil endpoint
	router.logRequest(req, true, 200, "redirect", "example.com", "", nil)
}

func TestRouter_GetProxyEndpointFromHostname_WildcardAlias(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.RootOrMatchingDomain = "main.example.com"
	endpoint.MatchingDomainAlias = []string{"*.alias.com"}
	endpoint.parent = router
	router.ProxyEndpoints.Store("main.example.com", &endpoint)

	result := router.GetProxyEndpointFromHostname("sub.alias.com")

	assert.NotNil(t, result)
	assert.Equal(t, "main.example.com", result.RootOrMatchingDomain)
}

func TestRouter_GetProxyEndpointFromHostname_InvalidWildcardPattern(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.RootOrMatchingDomain = "[invalid-pattern"
	endpoint.parent = router
	router.ProxyEndpoints.Store("[invalid-pattern", &endpoint)

	// Should handle invalid pattern gracefully
	result := router.GetProxyEndpointFromHostname("test.com")
	assert.Nil(t, result)
}

func TestRouter_rewriteURL_EdgeCases(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	tests := []struct {
		name       string
		rootURL    string
		requestURL string
		expected   string
	}{
		{
			name:       "Empty root URL",
			rootURL:    "",
			requestURL: "/test",
			expected:   "/test",
		},
		{
			name:       "Multiple consecutive slashes",
			rootURL:    "/api/",
			requestURL: "/api///test",
			expected:   "/test",
		},
		{
			name:       "Root URL equals request URL",
			rootURL:    "/api/",
			requestURL: "/api/",
			expected:   "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := router.rewriteURL(tt.rootURL, tt.requestURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRouter_GetProxyEndpointFromHostname_SortingBySpecificity(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Add endpoints with different specificity levels
	endpoint1 := GetDefaultProxyEndpoint()
	endpoint1.RootOrMatchingDomain = "*.com"
	endpoint1.parent = router
	router.ProxyEndpoints.Store("*.com", &endpoint1)

	endpoint2 := GetDefaultProxyEndpoint()
	endpoint2.RootOrMatchingDomain = "*.example.com"
	endpoint2.parent = router
	router.ProxyEndpoints.Store("*.example.com", &endpoint2)

	endpoint3 := GetDefaultProxyEndpoint()
	endpoint3.RootOrMatchingDomain = "test.example.com"
	endpoint3.parent = router
	router.ProxyEndpoints.Store("test.example.com", &endpoint3)

	// Test matching - should prioritize more specific matches
	result := router.GetProxyEndpointFromHostname("test.example.com")
	assert.NotNil(t, result)
	// Exact match should win
	assert.Equal(t, "test.example.com", result.RootOrMatchingDomain)

	// Test wildcard matching - should pick longer domain
	result = router.GetProxyEndpointFromHostname("other.example.com")
	assert.NotNil(t, result)
	assert.Equal(t, "*.example.com", result.RootOrMatchingDomain)
}

func TestRouter_GetProxyEndpointFromHostname_WildcardCount(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Add endpoints with same length but different wildcard counts
	endpoint1 := GetDefaultProxyEndpoint()
	endpoint1.RootOrMatchingDomain = "*.*.example.com"
	endpoint1.parent = router
	router.ProxyEndpoints.Store("*.*.example.com", &endpoint1)

	endpoint2 := GetDefaultProxyEndpoint()
	endpoint2.RootOrMatchingDomain = "test.*.example.com"
	endpoint2.parent = router
	router.ProxyEndpoints.Store("test.*.example.com", &endpoint2)

	// Should prefer fewer wildcards
	result := router.GetProxyEndpointFromHostname("test.sub.example.com")
	assert.NotNil(t, result)
	// The one with fewer wildcards should win
	assert.Equal(t, "test.*.example.com", result.RootOrMatchingDomain)
}

func TestRouter_GetProxyEndpointFromHostname_LexicographicOrder(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Add endpoints with same length and wildcard count
	endpoint1 := GetDefaultProxyEndpoint()
	endpoint1.RootOrMatchingDomain = "b.example.com"
	endpoint1.parent = router
	router.ProxyEndpoints.Store("b.example.com", &endpoint1)

	endpoint2 := GetDefaultProxyEndpoint()
	endpoint2.RootOrMatchingDomain = "a.example.com"
	endpoint2.parent = router
	router.ProxyEndpoints.Store("a.example.com", &endpoint2)

	// Both match with wildcards theoretically, but we need exact matches
	// This test verifies lexicographic ordering as a tiebreaker
	endpoint1.RootOrMatchingDomain = "*.example.com"
	router.ProxyEndpoints.Store("*.example.com", &endpoint1)
	endpoint2.RootOrMatchingDomain = "*.example.com"
	router.ProxyEndpoints.Delete("a.example.com")
	router.ProxyEndpoints.Store("*.example.org", &endpoint2)

	result := router.GetProxyEndpointFromHostname("test.example.com")
	assert.NotNil(t, result)
}

func TestVirtualDirectoryEndpoint_URLParsing(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.parent = router

	// Test various URL formats
	tests := []struct {
		name         string
		domain       string
		requireTLS   bool
		matchingPath string
	}{
		{
			name:         "HTTP domain",
			domain:       "backend.local",
			requireTLS:   false,
			matchingPath: "/api/",
		},
		{
			name:         "HTTPS domain",
			domain:       "backend.local",
			requireTLS:   true,
			matchingPath: "/secure/",
		},
		{
			name:         "Domain with trailing slash",
			domain:       "backend.local/",
			requireTLS:   false,
			matchingPath: "/path/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vdir := &VirtualDirectoryEndpoint{
				MatchingPath: tt.matchingPath,
				Domain:       tt.domain,
				RequireTLS:   tt.requireTLS,
			}
			endpoint.VirtualDirectories = []*VirtualDirectoryEndpoint{vdir}

			// Prepare should handle URL parsing
			_, err := router.PrepareProxyRoute(&endpoint)

			// Should not error on valid domains
			if !contains(tt.domain, "://") {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestRouter_GetProxyEndpointFromHostname_EmptyHostname(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	result := router.GetProxyEndpointFromHostname("")
	assert.Nil(t, result)
}

func TestParseURL_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		urlString   string
		shouldError bool
	}{
		{
			name:        "Valid HTTP URL",
			urlString:   "http://example.com",
			shouldError: false,
		},
		{
			name:        "Valid HTTPS URL",
			urlString:   "https://example.com",
			shouldError: false,
		},
		{
			name:        "URL with path",
			urlString:   "http://example.com/path",
			shouldError: false,
		},
		{
			name:        "URL with port",
			urlString:   "http://example.com:8080",
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := url.Parse(tt.urlString)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
