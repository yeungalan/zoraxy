package dynamicproxy

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"imuslab.com/zoraxy/mod/access"
	"imuslab.com/zoraxy/mod/dynamicproxy/loadbalance"
	"imuslab.com/zoraxy/mod/dynamicproxy/redirection"
	"imuslab.com/zoraxy/mod/geodb"
	"imuslab.com/zoraxy/mod/info/logger"
	"imuslab.com/zoraxy/mod/statistic"
	"imuslab.com/zoraxy/mod/tlscert"
)

// Helper function to create a minimal RouterOption for testing
func createTestRouterOption(t *testing.T) RouterOption {
	t.Helper()

	// Create mock logger
	testLogger, err := logger.NewLogger("/tmp", "test", false)
	require.NoError(t, err)

	// Create mock TLS manager
	tlsManager, err := tlscert.NewManager("/tmp/test_certs", "", testLogger)
	require.NoError(t, err)

	// Create mock redirection table
	redirectTable, err := redirection.NewRuleTable("/tmp/test_redirect.db")
	require.NoError(t, err)

	// Create mock geodb store
	geoStore, err := geodb.NewGeoDb("/tmp/test_geo.db")
	require.NoError(t, err)

	// Create mock access controller
	accessCtrl, err := access.NewController("/tmp/test_access")
	require.NoError(t, err)

	// Create mock statistic collector
	statCollector, err := statistic.NewCollector("/tmp/test_stats.db")
	require.NoError(t, err)

	// Create mock load balancer
	loadBalancer := loadbalance.NewRouteManager()

	return RouterOption{
		HostUUID:           "test-uuid",
		HostVersion:        "1.0.0-test",
		Port:               0, // Use random port for testing
		UseTls:             false,
		MinTLSVersion:      tls.VersionTLS12,
		NoCache:            false,
		ListenOnPort80:     false,
		ForceHttpsRedirect: false,
		TlsManager:         tlsManager,
		RedirectRuleTable:  redirectTable,
		GeodbStore:         geoStore,
		AccessController:   accessCtrl,
		StatisticCollector: statCollector,
		WebDirectory:       "/tmp/test_web",
		LoadBalancer:       loadBalancer,
		DevelopmentMode:    true,
		Logger:             testLogger,
	}
}

func TestNewDynamicProxy(t *testing.T) {
	option := createTestRouterOption(t)

	router, err := NewDynamicProxy(option)

	assert.NoError(t, err)
	assert.NotNil(t, router)
	assert.NotNil(t, router.ProxyEndpoints)
	assert.NotNil(t, router.mux)
	assert.False(t, router.Running)
	assert.Nil(t, router.server)
	assert.Equal(t, &option, router.Option)
}

func TestRouter_IsProxiedSubdomain(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Create a test endpoint
	endpoint := GetDefaultProxyEndpoint()
	endpoint.RootOrMatchingDomain = "test.example.com"
	endpoint.parent = router

	// Add the endpoint to the router
	router.ProxyEndpoints.Store("test.example.com", &endpoint)

	tests := []struct {
		name     string
		host     string
		expected bool
	}{
		{
			name:     "Matching subdomain",
			host:     "test.example.com",
			expected: true,
		},
		{
			name:     "Matching subdomain with port",
			host:     "test.example.com:8080",
			expected: true,
		},
		{
			name:     "Non-matching subdomain",
			host:     "other.example.com",
			expected: false,
		},
		{
			name:     "Empty host",
			host:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://"+tt.host+"/", nil)
			req.Host = tt.host

			result := router.IsProxiedSubdomain(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRouter_LoadProxy(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Create and add a test endpoint
	endpoint := GetDefaultProxyEndpoint()
	endpoint.RootOrMatchingDomain = "test.example.com"
	endpoint.parent = router
	router.ProxyEndpoints.Store("test.example.com", &endpoint)

	tests := []struct {
		name        string
		domain      string
		expectError bool
	}{
		{
			name:        "Load existing endpoint",
			domain:      "test.example.com",
			expectError: false,
		},
		{
			name:        "Load non-existing endpoint",
			domain:      "nonexistent.example.com",
			expectError: true,
		},
		{
			name:        "Case insensitive load",
			domain:      "TEST.EXAMPLE.COM",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := router.LoadProxy(tt.domain)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, "test.example.com", result.RootOrMatchingDomain)
			}
		})
	}
}

func TestCopyEndpoint(t *testing.T) {
	original := GetDefaultProxyEndpoint()
	original.RootOrMatchingDomain = "test.example.com"
	original.Disabled = true
	original.RequireRateLimit = true
	original.RateLimit = 100

	copied := CopyEndpoint(&original)

	assert.NotNil(t, copied)
	assert.Equal(t, original.RootOrMatchingDomain, copied.RootOrMatchingDomain)
	assert.Equal(t, original.Disabled, copied.Disabled)
	assert.Equal(t, original.RequireRateLimit, copied.RequireRateLimit)
	assert.Equal(t, original.RateLimit, copied.RateLimit)

	// Verify it's a deep copy by modifying the copy
	copied.RootOrMatchingDomain = "modified.example.com"
	assert.NotEqual(t, original.RootOrMatchingDomain, copied.RootOrMatchingDomain)
}

func TestCopyEndpoint_Nil(t *testing.T) {
	// Test with nil-like content to ensure robustness
	endpoint := &ProxyEndpoint{
		RootOrMatchingDomain: "",
		ActiveOrigins:        nil,
		InactiveOrigins:      nil,
	}

	copied := CopyEndpoint(endpoint)
	assert.NotNil(t, copied)
}

func TestRouter_GetProxyEndpointsAsMap(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Add multiple endpoints
	endpoint1 := GetDefaultProxyEndpoint()
	endpoint1.RootOrMatchingDomain = "test1.example.com"
	router.ProxyEndpoints.Store("test1.example.com", &endpoint1)

	endpoint2 := GetDefaultProxyEndpoint()
	endpoint2.RootOrMatchingDomain = "test2.example.com"
	router.ProxyEndpoints.Store("test2.example.com", &endpoint2)

	result := router.GetProxyEndpointsAsMap()

	assert.Equal(t, 2, len(result))
	assert.Contains(t, result, "test1.example.com")
	assert.Contains(t, result, "test2.example.com")
}

func TestRouter_GetProxyEndpointsAsMap_Empty(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	result := router.GetProxyEndpointsAsMap()

	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result))
}

func TestRouter_UpdateTLSSetting(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Initially TLS should be disabled
	assert.False(t, router.Option.UseTls)

	// Update TLS setting
	router.UpdateTLSSetting(true)

	// Verify TLS is enabled
	assert.True(t, router.Option.UseTls)
}

func TestRouter_SetTlsMinVersion(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Set minimum TLS version to 1.3
	router.SetTlsMinVersion(tls.VersionTLS13)

	assert.Equal(t, uint16(tls.VersionTLS13), router.Option.MinTLSVersion)
}

func TestRouter_UpdatePort80ListenerState(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	assert.False(t, router.Option.ListenOnPort80)

	router.UpdatePort80ListenerState(true)

	assert.True(t, router.Option.ListenOnPort80)
}

func TestRouter_UpdateHttpToHttpsRedirectSetting(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	assert.False(t, router.Option.ForceHttpsRedirect)

	router.UpdateHttpToHttpsRedirectSetting(true)

	assert.True(t, router.Option.ForceHttpsRedirect)
}

func TestRouter_StartProxyService_NoRoot(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Try to start without setting root
	err = router.StartProxyService()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "root not set")
}

func TestRouter_StartProxyService_AlreadyRunning(t *testing.T) {
	option := createTestRouterOption(t)
	option.Port = 0 // Random port
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Set up root endpoint
	rootEndpoint := GetDefaultProxyEndpoint()
	rootEndpoint.ProxyType = ProxyTypeRoot
	rootEndpoint.parent = router
	router.Root = &rootEndpoint

	// Create a fake server to simulate running state
	router.server = &http.Server{}

	err = router.StartProxyService()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

func TestRouter_StopProxyService_NotRunning(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	err = router.StopProxyService()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already stopped")
}

func TestRouter_Lifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	option := createTestRouterOption(t)
	option.Port = 0 // Use random available port
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Set up a minimal root endpoint
	rootEndpoint := GetDefaultProxyEndpoint()
	rootEndpoint.ProxyType = ProxyTypeRoot
	rootEndpoint.DefaultSiteOption = DefaultSite_NotFoundPage
	rootEndpoint.parent = router

	// Prepare the endpoint
	preparedEndpoint, err := router.PrepareProxyRoute(&rootEndpoint)
	require.NoError(t, err)

	router.Root = preparedEndpoint

	// Start the proxy service
	err = router.StartProxyService()
	require.NoError(t, err)
	assert.True(t, router.Running)
	assert.NotNil(t, router.server)

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop the proxy service
	err = router.StopProxyService()
	require.NoError(t, err)
	assert.False(t, router.Running)
	assert.Nil(t, router.server)
}

func TestRouter_Restart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	option := createTestRouterOption(t)
	option.Port = 0
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Set up root endpoint
	rootEndpoint := GetDefaultProxyEndpoint()
	rootEndpoint.ProxyType = ProxyTypeRoot
	rootEndpoint.DefaultSiteOption = DefaultSite_NotFoundPage
	rootEndpoint.parent = router

	preparedEndpoint, err := router.PrepareProxyRoute(&rootEndpoint)
	require.NoError(t, err)
	router.Root = preparedEndpoint

	// Start initially
	err = router.StartProxyService()
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// Restart
	err = router.Restart()
	require.NoError(t, err)
	assert.True(t, router.Running)

	// Clean up
	err = router.StopProxyService()
	require.NoError(t, err)
}

func TestRouter_Restart_NotRunning(t *testing.T) {
	option := createTestRouterOption(t)
	option.Port = 0
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Set up root endpoint
	rootEndpoint := GetDefaultProxyEndpoint()
	rootEndpoint.ProxyType = ProxyTypeRoot
	rootEndpoint.parent = router

	preparedEndpoint, err := router.PrepareProxyRoute(&rootEndpoint)
	require.NoError(t, err)
	router.Root = preparedEndpoint

	// Try to restart when not running
	err = router.Restart()
	require.NoError(t, err)
	assert.True(t, router.Running)

	// Clean up
	router.StopProxyService()
}
