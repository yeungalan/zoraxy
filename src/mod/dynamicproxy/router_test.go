package dynamicproxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"imuslab.com/zoraxy/mod/dynamicproxy/loadbalance"
)

func TestRouter_PrepareProxyRoute(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.RootOrMatchingDomain = "test.example.com"

	// Add an upstream
	upstream := &loadbalance.Upstream{
		OriginIpOrDomain:      "backend.local",
		RequireTLS:            false,
		SkipCertValidations:   false,
		SkipWebSocketOriginCheck: false,
	}
	endpoint.ActiveOrigins = []*loadbalance.Upstream{upstream}

	result, err := router.PrepareProxyRoute(&endpoint)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, router, result.parent)
}

func TestRouter_PrepareProxyRoute_WithVirtualDirectory(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.RootOrMatchingDomain = "test.example.com"

	// Add virtual directory
	vdir := &VirtualDirectoryEndpoint{
		MatchingPath:        "/api/",
		Domain:              "api.backend.local",
		RequireTLS:          false,
		SkipCertValidations: false,
	}
	endpoint.VirtualDirectories = []*VirtualDirectoryEndpoint{vdir}

	result, err := router.PrepareProxyRoute(&endpoint)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.VirtualDirectories, 1)
	assert.NotNil(t, result.VirtualDirectories[0].proxy)
	assert.Equal(t, &endpoint, result.VirtualDirectories[0].parent)
}

func TestRouter_PrepareProxyRoute_VirtualDirectoryWithTLS(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()

	vdir := &VirtualDirectoryEndpoint{
		MatchingPath:        "/secure/",
		Domain:              "secure.backend.local",
		RequireTLS:          true,
		SkipCertValidations: true,
	}
	endpoint.VirtualDirectories = []*VirtualDirectoryEndpoint{vdir}

	result, err := router.PrepareProxyRoute(&endpoint)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.VirtualDirectories[0].proxy)
}

func TestRouter_PrepareProxyRoute_VirtualDirectoryWithTrailingSlash(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()

	// Domain with trailing slash should be handled
	vdir := &VirtualDirectoryEndpoint{
		MatchingPath: "/path/",
		Domain:       "backend.local/",
		RequireTLS:   false,
	}
	endpoint.VirtualDirectories = []*VirtualDirectoryEndpoint{vdir}

	result, err := router.PrepareProxyRoute(&endpoint)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRouter_PrepareProxyRoute_VirtualDirectoryEmptyDomain(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()

	// Empty domain should be skipped
	vdir := &VirtualDirectoryEndpoint{
		MatchingPath: "/path/",
		Domain:       "",
		RequireTLS:   false,
	}
	endpoint.VirtualDirectories = []*VirtualDirectoryEndpoint{vdir}

	result, err := router.PrepareProxyRoute(&endpoint)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// Proxy should not be set for empty domain
	assert.Nil(t, result.VirtualDirectories[0].proxy)
}

func TestRouter_AddProxyRouteToRuntime(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.RootOrMatchingDomain = "test.example.com"

	// Prepare first
	prepared, err := router.PrepareProxyRoute(&endpoint)
	require.NoError(t, err)

	// Add to runtime
	err = router.AddProxyRouteToRuntime(prepared)
	assert.NoError(t, err)

	// Verify it's in the map
	result, ok := router.ProxyEndpoints.Load("test.example.com")
	assert.True(t, ok)
	assert.NotNil(t, result)
}

func TestRouter_AddProxyRouteToRuntime_CaseInsensitive(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.RootOrMatchingDomain = "TEST.EXAMPLE.COM"

	prepared, err := router.PrepareProxyRoute(&endpoint)
	require.NoError(t, err)

	err = router.AddProxyRouteToRuntime(prepared)
	assert.NoError(t, err)

	// Should be stored in lowercase
	result, ok := router.ProxyEndpoints.Load("test.example.com")
	assert.True(t, ok)
	assert.NotNil(t, result)
}

func TestRouter_AddProxyRouteToRuntime_NoActiveOrigins(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.RootOrMatchingDomain = "test.example.com"
	endpoint.ActiveOrigins = []*loadbalance.Upstream{} // Empty

	prepared, err := router.PrepareProxyRoute(&endpoint)
	require.NoError(t, err)

	// Should succeed even without active origins
	err = router.AddProxyRouteToRuntime(prepared)
	assert.NoError(t, err)
}

func TestRouter_AddProxyRouteToRuntime_NotPrepared(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.RootOrMatchingDomain = "test.example.com"

	// Add upstream but don't prepare
	upstream := &loadbalance.Upstream{
		OriginIpOrDomain: "backend.local",
	}
	endpoint.ActiveOrigins = []*loadbalance.Upstream{upstream}
	endpoint.parent = router

	// Try to add without preparing
	err = router.AddProxyRouteToRuntime(&endpoint)

	// Should error because endpoint is not prepared
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not ready")
}

func TestRouter_SetProxyRouteAsRoot(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.ProxyType = ProxyTypeRoot

	prepared, err := router.PrepareProxyRoute(&endpoint)
	require.NoError(t, err)

	err = router.SetProxyRouteAsRoot(prepared)
	assert.NoError(t, err)
	assert.Equal(t, prepared, router.Root)
}

func TestRouter_SetProxyRouteAsRoot_NotPrepared(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.ProxyType = ProxyTypeRoot

	// Add upstream but don't prepare
	upstream := &loadbalance.Upstream{
		OriginIpOrDomain: "backend.local",
	}
	endpoint.ActiveOrigins = []*loadbalance.Upstream{upstream}
	endpoint.parent = router

	err = router.SetProxyRouteAsRoot(&endpoint)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not ready")
}

func TestRouter_RemoveProxyEndpointByRootname(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.RootOrMatchingDomain = "test.example.com"
	endpoint.parent = router

	// Add endpoint
	router.ProxyEndpoints.Store("test.example.com", &endpoint)

	// Remove it
	err = router.RemoveProxyEndpointByRootname("test.example.com")
	assert.NoError(t, err)

	// Verify it's removed
	_, ok := router.ProxyEndpoints.Load("test.example.com")
	assert.False(t, ok)
}

func TestRouter_RemoveProxyEndpointByRootname_NotFound(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	err = router.RemoveProxyEndpointByRootname("nonexistent.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRouter_GetProxyEndpointById(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.RootOrMatchingDomain = "test.example.com"
	endpoint.MatchingDomainAlias = []string{"alias1.com", "alias2.com"}
	endpoint.parent = router

	router.ProxyEndpoints.Store("test.example.com", &endpoint)

	// Test finding by root domain
	result, err := router.GetProxyEndpointById("test.example.com", false)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test.example.com", result.RootOrMatchingDomain)

	// Test finding by alias with includeAlias=true
	result, err = router.GetProxyEndpointById("alias1.com", true)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test.example.com", result.RootOrMatchingDomain)

	// Test finding by alias with includeAlias=false
	result, err = router.GetProxyEndpointById("alias1.com", false)
	assert.Error(t, err)
	assert.Nil(t, result)

	// Test not found
	result, err = router.GetProxyEndpointById("nonexistent.com", true)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestRouter_GetProxyEndpointByAlias(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.RootOrMatchingDomain = "main.example.com"
	endpoint.MatchingDomainAlias = []string{"alias.example.com"}
	endpoint.parent = router

	router.ProxyEndpoints.Store("main.example.com", &endpoint)

	// Test exact alias match
	result, err := router.GetProxyEndpointByAlias("alias.example.com")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "main.example.com", result.RootOrMatchingDomain)

	// Test not found
	result, err = router.GetProxyEndpointByAlias("nonexistent.com")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestRouter_GetProxyEndpointByAlias_Wildcard(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()
	endpoint.RootOrMatchingDomain = "main.example.com"
	endpoint.MatchingDomainAlias = []string{"*.wildcard.com"}
	endpoint.parent = router

	router.ProxyEndpoints.Store("main.example.com", &endpoint)

	// Test wildcard alias match
	result, err := router.GetProxyEndpointByAlias("sub.wildcard.com")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "main.example.com", result.RootOrMatchingDomain)

	// Test wildcard no match
	result, err = router.GetProxyEndpointByAlias("sub.other.com")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestRouter_PrepareProxyRoute_UpstreamError(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()

	// Add an upstream with invalid configuration (will fail to start)
	upstream := &loadbalance.Upstream{
		OriginIpOrDomain: "", // Invalid empty origin
	}
	endpoint.ActiveOrigins = []*loadbalance.Upstream{upstream}

	// Should still succeed but log the error
	result, err := router.PrepareProxyRoute(&endpoint)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestRouter_PrepareProxyRoute_VirtualDirectoryWithHTTPPrefix(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()

	// Domain already has http:// prefix
	vdir := &VirtualDirectoryEndpoint{
		MatchingPath: "/path/",
		Domain:       "http://backend.local",
		RequireTLS:   false,
	}
	endpoint.VirtualDirectories = []*VirtualDirectoryEndpoint{vdir}

	result, err := router.PrepareProxyRoute(&endpoint)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.VirtualDirectories[0].proxy)
}

func TestRouter_PrepareProxyRoute_VirtualDirectoryWithHTTPSPrefix(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()

	// Domain already has https:// prefix but requireTLS is false
	vdir := &VirtualDirectoryEndpoint{
		MatchingPath: "/path/",
		Domain:       "https://backend.local",
		RequireTLS:   false, // Should be overridden by prefix
	}
	endpoint.VirtualDirectories = []*VirtualDirectoryEndpoint{vdir}

	result, err := router.PrepareProxyRoute(&endpoint)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.VirtualDirectories[0].proxy)
}

func TestRouter_GetProxyEndpointByAlias_MultipleEndpoints(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Add multiple endpoints
	endpoint1 := GetDefaultProxyEndpoint()
	endpoint1.RootOrMatchingDomain = "main1.example.com"
	endpoint1.MatchingDomainAlias = []string{"alias1.com"}
	router.ProxyEndpoints.Store("main1.example.com", &endpoint1)

	endpoint2 := GetDefaultProxyEndpoint()
	endpoint2.RootOrMatchingDomain = "main2.example.com"
	endpoint2.MatchingDomainAlias = []string{"alias2.com"}
	router.ProxyEndpoints.Store("main2.example.com", &endpoint2)

	// Should find the correct one
	result, err := router.GetProxyEndpointByAlias("alias2.com")
	assert.NoError(t, err)
	assert.Equal(t, "main2.example.com", result.RootOrMatchingDomain)
}

func TestRouter_PrepareProxyRoute_MultipleVirtualDirectories(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	endpoint := GetDefaultProxyEndpoint()

	// Add multiple vdirs
	vdir1 := &VirtualDirectoryEndpoint{
		MatchingPath: "/api/",
		Domain:       "api.backend.local",
		RequireTLS:   false,
	}
	vdir2 := &VirtualDirectoryEndpoint{
		MatchingPath: "/admin/",
		Domain:       "admin.backend.local",
		RequireTLS:   true,
	}
	endpoint.VirtualDirectories = []*VirtualDirectoryEndpoint{vdir1, vdir2}

	result, err := router.PrepareProxyRoute(&endpoint)

	assert.NoError(t, err)
	assert.Len(t, result.VirtualDirectories, 2)
	assert.NotNil(t, result.VirtualDirectories[0].proxy)
	assert.NotNil(t, result.VirtualDirectories[1].proxy)
}
