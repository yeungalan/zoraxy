package dynamicproxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"imuslab.com/zoraxy/mod/dynamicproxy/loadbalance"
	"imuslab.com/zoraxy/mod/dynamicproxy/rewrite"
)

func TestProxyEndpoint_UserDefinedHeaderExists(t *testing.T) {
	endpoint := GetDefaultProxyEndpoint()

	// Add a header
	header := &rewrite.UserDefinedHeader{
		Key:       "X-Test-Header",
		Value:     "test-value",
		Direction: rewrite.HeaderDirection_RequestHeader,
	}
	err := endpoint.AddUserDefinedHeader(header)
	require.NoError(t, err)

	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "Existing header - exact case",
			key:      "X-Test-Header",
			expected: true,
		},
		{
			name:     "Existing header - different case",
			key:      "x-test-header",
			expected: true,
		},
		{
			name:     "Existing header - uppercase",
			key:      "X-TEST-HEADER",
			expected: true,
		},
		{
			name:     "Non-existing header",
			key:      "X-Nonexistent",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := endpoint.UserDefinedHeaderExists(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProxyEndpoint_RemoveUserDefinedHeader(t *testing.T) {
	endpoint := GetDefaultProxyEndpoint()

	// Add multiple headers
	header1 := &rewrite.UserDefinedHeader{
		Key:       "X-Header-1",
		Value:     "value1",
		Direction: rewrite.HeaderDirection_RequestHeader,
	}
	header2 := &rewrite.UserDefinedHeader{
		Key:       "X-Header-2",
		Value:     "value2",
		Direction: rewrite.HeaderDirection_RequestHeader,
	}

	endpoint.AddUserDefinedHeader(header1)
	endpoint.AddUserDefinedHeader(header2)

	// Remove one header
	err := endpoint.RemoveUserDefinedHeader("X-Header-1")
	assert.NoError(t, err)

	// Verify removal
	assert.False(t, endpoint.UserDefinedHeaderExists("X-Header-1"))
	assert.True(t, endpoint.UserDefinedHeaderExists("X-Header-2"))

	// Remove non-existing header should not error
	err = endpoint.RemoveUserDefinedHeader("X-Nonexistent")
	assert.NoError(t, err)
}

func TestProxyEndpoint_RemoveUserDefinedHeader_CaseInsensitive(t *testing.T) {
	endpoint := GetDefaultProxyEndpoint()

	header := &rewrite.UserDefinedHeader{
		Key:       "X-Test-Header",
		Value:     "value",
		Direction: rewrite.HeaderDirection_RequestHeader,
	}
	endpoint.AddUserDefinedHeader(header)

	// Remove using different case
	err := endpoint.RemoveUserDefinedHeader("x-test-header")
	assert.NoError(t, err)
	assert.False(t, endpoint.UserDefinedHeaderExists("X-Test-Header"))
}

func TestProxyEndpoint_AddUserDefinedHeader(t *testing.T) {
	endpoint := GetDefaultProxyEndpoint()

	header := &rewrite.UserDefinedHeader{
		Key:       "X-Custom-Header",
		Value:     "custom-value",
		Direction: rewrite.HeaderDirection_RequestHeader,
	}

	err := endpoint.AddUserDefinedHeader(header)

	assert.NoError(t, err)
	assert.True(t, endpoint.UserDefinedHeaderExists("X-Custom-Header"))
	assert.NotNil(t, endpoint.HeaderRewriteRules)
	assert.Len(t, endpoint.HeaderRewriteRules.UserDefinedHeaders, 1)
}

func TestProxyEndpoint_AddUserDefinedHeader_Duplicate(t *testing.T) {
	endpoint := GetDefaultProxyEndpoint()

	header1 := &rewrite.UserDefinedHeader{
		Key:       "X-Test",
		Value:     "value1",
		Direction: rewrite.HeaderDirection_RequestHeader,
	}
	header2 := &rewrite.UserDefinedHeader{
		Key:       "X-Test",
		Value:     "value2",
		Direction: rewrite.HeaderDirection_RequestHeader,
	}

	// Add first header
	err := endpoint.AddUserDefinedHeader(header1)
	assert.NoError(t, err)

	// Add duplicate - should replace
	err = endpoint.AddUserDefinedHeader(header2)
	assert.NoError(t, err)

	// Should still have only one header
	assert.Len(t, endpoint.HeaderRewriteRules.UserDefinedHeaders, 1)
	assert.Equal(t, "value2", endpoint.HeaderRewriteRules.UserDefinedHeaders[0].Value)
}

func TestProxyEndpoint_GetVirtualDirectoryHandlerFromRequestURI(t *testing.T) {
	endpoint := GetDefaultProxyEndpoint()

	// Add virtual directories
	vdir1 := &VirtualDirectoryEndpoint{
		MatchingPath: "/api/v1/",
		Domain:       "backend1.local",
		RequireTLS:   false,
	}
	vdir2 := &VirtualDirectoryEndpoint{
		MatchingPath: "/api/v2/",
		Domain:       "backend2.local",
		RequireTLS:   true,
	}
	vdir3 := &VirtualDirectoryEndpoint{
		MatchingPath: "/admin/",
		Domain:       "admin.local",
		RequireTLS:   true,
	}

	endpoint.VirtualDirectories = []*VirtualDirectoryEndpoint{vdir1, vdir2, vdir3}

	tests := []struct {
		name       string
		requestURI string
		expected   *VirtualDirectoryEndpoint
	}{
		{
			name:       "Match /api/v1/",
			requestURI: "/api/v1/users",
			expected:   vdir1,
		},
		{
			name:       "Match /api/v2/",
			requestURI: "/api/v2/posts",
			expected:   vdir2,
		},
		{
			name:       "Match /admin/",
			requestURI: "/admin/settings",
			expected:   vdir3,
		},
		{
			name:       "No match",
			requestURI: "/public/file",
			expected:   nil,
		},
		{
			name:       "Exact match",
			requestURI: "/api/v1/",
			expected:   vdir1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := endpoint.GetVirtualDirectoryHandlerFromRequestURI(tt.requestURI)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProxyEndpoint_GetVirtualDirectoryRuleByMatchingPath(t *testing.T) {
	endpoint := GetDefaultProxyEndpoint()

	vdir := &VirtualDirectoryEndpoint{
		MatchingPath: "/api/",
		Domain:       "api.local",
	}
	endpoint.VirtualDirectories = []*VirtualDirectoryEndpoint{vdir}

	// Test exact match
	result := endpoint.GetVirtualDirectoryRuleByMatchingPath("/api/")
	assert.NotNil(t, result)
	assert.Equal(t, "/api/", result.MatchingPath)

	// Test no match
	result = endpoint.GetVirtualDirectoryRuleByMatchingPath("/api/v1/")
	assert.Nil(t, result)
}

func TestProxyEndpoint_RemoveVirtualDirectoryRuleByMatchingPath(t *testing.T) {
	endpoint := GetDefaultProxyEndpoint()

	vdir1 := &VirtualDirectoryEndpoint{
		MatchingPath: "/api/",
		Domain:       "api.local",
	}
	vdir2 := &VirtualDirectoryEndpoint{
		MatchingPath: "/admin/",
		Domain:       "admin.local",
	}

	endpoint.VirtualDirectories = []*VirtualDirectoryEndpoint{vdir1, vdir2}

	// Remove existing vdir
	err := endpoint.RemoveVirtualDirectoryRuleByMatchingPath("/api/")
	assert.NoError(t, err)
	assert.Len(t, endpoint.VirtualDirectories, 1)
	assert.Equal(t, "/admin/", endpoint.VirtualDirectories[0].MatchingPath)

	// Try to remove non-existing vdir
	err = endpoint.RemoveVirtualDirectoryRuleByMatchingPath("/nonexistent/")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProxyEndpoint_AddVirtualDirectoryRule(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.ProxyType = ProxyTypeHost
	endpoint.RootOrMatchingDomain = "test.example.com"
	endpoint.parent = router

	// Prepare the endpoint
	preparedEndpoint, err := router.PrepareProxyRoute(&endpoint)
	require.NoError(t, err)

	// Add to runtime
	router.AddProxyRouteToRuntime(preparedEndpoint)

	// Create a new vdir
	vdir := &VirtualDirectoryEndpoint{
		MatchingPath: "/api/",
		Domain:       "api.local",
		RequireTLS:   false,
	}

	// Add vdir
	result, err := preparedEndpoint.AddVirtualDirectoryRule(vdir)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.VirtualDirectories, 1)
}

func TestProxyEndpoint_AddVirtualDirectoryRule_Duplicate(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.ProxyType = ProxyTypeHost
	endpoint.parent = router

	vdir1 := &VirtualDirectoryEndpoint{
		MatchingPath: "/api/",
		Domain:       "api.local",
	}
	endpoint.VirtualDirectories = []*VirtualDirectoryEndpoint{vdir1}

	preparedEndpoint, err := router.PrepareProxyRoute(&endpoint)
	require.NoError(t, err)

	// Try to add duplicate
	vdir2 := &VirtualDirectoryEndpoint{
		MatchingPath: "/api/",
		Domain:       "api2.local",
	}

	result, err := preparedEndpoint.AddVirtualDirectoryRule(vdir2)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "already exists")
}

func TestProxyEndpoint_UpstreamOriginExists(t *testing.T) {
	endpoint := GetDefaultProxyEndpoint()

	upstream1 := &loadbalance.Upstream{
		OriginIpOrDomain: "backend1.local",
	}
	upstream2 := &loadbalance.Upstream{
		OriginIpOrDomain: "backend2.local",
	}

	endpoint.ActiveOrigins = []*loadbalance.Upstream{upstream1}
	endpoint.InactiveOrigins = []*loadbalance.Upstream{upstream2}

	tests := []struct {
		name     string
		origin   string
		expected bool
	}{
		{
			name:     "Active origin exists",
			origin:   "backend1.local",
			expected: true,
		},
		{
			name:     "Inactive origin exists",
			origin:   "backend2.local",
			expected: true,
		},
		{
			name:     "Origin does not exist",
			origin:   "backend3.local",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := endpoint.UpstreamOriginExists(tt.origin)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProxyEndpoint_GetUpstreamOriginByMatchingIP(t *testing.T) {
	endpoint := GetDefaultProxyEndpoint()

	upstream1 := &loadbalance.Upstream{
		OriginIpOrDomain: "192.168.1.10",
	}
	upstream2 := &loadbalance.Upstream{
		OriginIpOrDomain: "192.168.1.20",
	}

	endpoint.ActiveOrigins = []*loadbalance.Upstream{upstream1}
	endpoint.InactiveOrigins = []*loadbalance.Upstream{upstream2}

	// Test finding active upstream
	result, err := endpoint.GetUpstreamOriginByMatchingIP("192.168.1.10")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "192.168.1.10", result.OriginIpOrDomain)

	// Test finding inactive upstream
	result, err = endpoint.GetUpstreamOriginByMatchingIP("192.168.1.20")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "192.168.1.20", result.OriginIpOrDomain)

	// Test not found
	result, err = endpoint.GetUpstreamOriginByMatchingIP("192.168.1.30")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

func TestProxyEndpoint_RemoveUpstreamOrigin(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.parent = router

	upstream1 := &loadbalance.Upstream{
		OriginIpOrDomain: "backend1.local",
	}
	upstream2 := &loadbalance.Upstream{
		OriginIpOrDomain: "backend2.local",
	}

	endpoint.ActiveOrigins = []*loadbalance.Upstream{upstream1, upstream2}

	// Remove an upstream
	err = endpoint.RemoveUpstreamOrigin("backend1.local")
	assert.NoError(t, err)
	assert.Len(t, endpoint.ActiveOrigins, 1)
	assert.Equal(t, "backend2.local", endpoint.ActiveOrigins[0].OriginIpOrDomain)

	// Remove non-existing upstream (should not error)
	err = endpoint.RemoveUpstreamOrigin("nonexistent.local")
	assert.NoError(t, err)
}

func TestProxyEndpoint_RemoveUpstreamOrigin_WithSpaces(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.parent = router

	upstream := &loadbalance.Upstream{
		OriginIpOrDomain: "backend.local",
	}
	endpoint.ActiveOrigins = []*loadbalance.Upstream{upstream}

	// Remove with leading/trailing spaces
	err = endpoint.RemoveUpstreamOrigin("  backend.local  ")
	assert.NoError(t, err)
	assert.Len(t, endpoint.ActiveOrigins, 0)
}

func TestProxyEndpoint_ContainsWildcardName(t *testing.T) {
	tests := []struct {
		name            string
		hostname        string
		aliases         []string
		skipAliasCheck  bool
		expectedResult  bool
	}{
		{
			name:           "Wildcard in hostname",
			hostname:       "*.example.com",
			aliases:        []string{},
			skipAliasCheck: false,
			expectedResult: true,
		},
		{
			name:           "No wildcard in hostname",
			hostname:       "test.example.com",
			aliases:        []string{},
			skipAliasCheck: false,
			expectedResult: false,
		},
		{
			name:           "Wildcard in alias",
			hostname:       "test.example.com",
			aliases:        []string{"*.alias.com"},
			skipAliasCheck: false,
			expectedResult: true,
		},
		{
			name:           "Wildcard in alias but skipped",
			hostname:       "test.example.com",
			aliases:        []string{"*.alias.com"},
			skipAliasCheck: true,
			expectedResult: false,
		},
		{
			name:           "Empty hostname",
			hostname:       "",
			aliases:        []string{"test.com"},
			skipAliasCheck: false,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint := GetDefaultProxyEndpoint()
			endpoint.RootOrMatchingDomain = tt.hostname
			endpoint.MatchingDomainAlias = tt.aliases

			result := endpoint.ContainsWildcardName(tt.skipAliasCheck)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestProxyEndpoint_Clone(t *testing.T) {
	original := GetDefaultProxyEndpoint()
	original.RootOrMatchingDomain = "test.example.com"
	original.Disabled = true
	original.RequireRateLimit = true
	original.RateLimit = 100
	original.Tags = []string{"tag1", "tag2"}

	cloned := original.Clone()

	assert.NotNil(t, cloned)
	assert.Equal(t, original.RootOrMatchingDomain, cloned.RootOrMatchingDomain)
	assert.Equal(t, original.Disabled, cloned.Disabled)
	assert.Equal(t, original.RequireRateLimit, cloned.RequireRateLimit)
	assert.Equal(t, original.RateLimit, cloned.RateLimit)
	assert.Equal(t, original.Tags, cloned.Tags)

	// Modify clone to ensure it's a deep copy
	cloned.RootOrMatchingDomain = "modified.example.com"
	assert.NotEqual(t, original.RootOrMatchingDomain, cloned.RootOrMatchingDomain)
}

func TestProxyEndpoint_Remove(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.RootOrMatchingDomain = "test.example.com"
	endpoint.parent = router

	// Add endpoint to router
	router.ProxyEndpoints.Store("test.example.com", &endpoint)

	// Verify it exists
	_, ok := router.ProxyEndpoints.Load("test.example.com")
	assert.True(t, ok)

	// Remove it
	err = endpoint.Remove()
	assert.NoError(t, err)

	// Verify it's removed
	_, ok = router.ProxyEndpoints.Load("test.example.com")
	assert.False(t, ok)
}

func TestProxyEndpoint_IsEnabled(t *testing.T) {
	endpoint := GetDefaultProxyEndpoint()

	// Default should be enabled
	assert.True(t, endpoint.IsEnabled())

	// Disable it
	endpoint.Disabled = true
	assert.False(t, endpoint.IsEnabled())
}

func TestProxyEndpoint_UpdateToRuntime(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.RootOrMatchingDomain = "test.example.com"
	endpoint.parent = router

	// Add to runtime
	endpoint.UpdateToRuntime()

	// Verify it's in the map
	result, ok := router.ProxyEndpoints.Load("test.example.com")
	assert.True(t, ok)
	assert.NotNil(t, result)

	// Modify and update
	endpoint.Disabled = true
	endpoint.UpdateToRuntime()

	// Verify update
	result, ok = router.ProxyEndpoints.Load("test.example.com")
	assert.True(t, ok)
	resultEndpoint := result.(*ProxyEndpoint)
	assert.True(t, resultEndpoint.Disabled)
}
