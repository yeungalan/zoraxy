package dynamicproxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyHandler_ServeHTTP_RootRouting(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Set up root endpoint
	rootEndpoint := GetDefaultProxyEndpoint()
	rootEndpoint.ProxyType = ProxyTypeRoot
	rootEndpoint.DefaultSiteOption = DefaultSite_NotFoundPage
	rootEndpoint.parent = router

	preparedRoot, err := router.PrepareProxyRoute(&rootEndpoint)
	require.NoError(t, err)
	router.Root = preparedRoot

	handler := &ProxyHandler{Parent: router}

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return 404 for root with NotFoundPage option
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProxyHandler_ServeHTTP_SubdomainRouting(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Set up root endpoint
	rootEndpoint := GetDefaultProxyEndpoint()
	rootEndpoint.ProxyType = ProxyTypeRoot
	rootEndpoint.DefaultSiteOption = DefaultSite_NotFoundPage
	rootEndpoint.parent = router
	preparedRoot, err := router.PrepareProxyRoute(&rootEndpoint)
	require.NoError(t, err)
	router.Root = preparedRoot

	// Create a subdomain endpoint with disabled state
	subEndpoint := GetDefaultProxyEndpoint()
	subEndpoint.ProxyType = ProxyTypeHost
	subEndpoint.RootOrMatchingDomain = "test.example.com"
	subEndpoint.Disabled = true // Disabled endpoint
	subEndpoint.parent = router

	preparedSub, err := router.PrepareProxyRoute(&subEndpoint)
	require.NoError(t, err)
	router.AddProxyRouteToRuntime(preparedSub)

	handler := &ProxyHandler{Parent: router}

	req := httptest.NewRequest("GET", "http://test.example.com/", nil)
	req.Host = "test.example.com"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should fallback to root since subdomain is disabled
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProxyHandler_handleRootRouting_NotFoundPage(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	rootEndpoint := GetDefaultProxyEndpoint()
	rootEndpoint.ProxyType = ProxyTypeRoot
	rootEndpoint.DefaultSiteOption = DefaultSite_NotFoundPage
	rootEndpoint.parent = router
	router.Root = &rootEndpoint

	handler := &ProxyHandler{Parent: router}

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	handler.handleRootRouting(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
}

func TestProxyHandler_handleRootRouting_Redirect(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	rootEndpoint := GetDefaultProxyEndpoint()
	rootEndpoint.ProxyType = ProxyTypeRoot
	rootEndpoint.DefaultSiteOption = DefaultSite_Redirect
	rootEndpoint.DefaultSiteValue = "https://redirect.example.com"
	rootEndpoint.parent = router
	router.Root = &rootEndpoint

	handler := &ProxyHandler{Parent: router}

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	handler.handleRootRouting(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "https://redirect.example.com", w.Header().Get("Location"))
}

func TestProxyHandler_handleRootRouting_Redirect_NoProtocol(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	rootEndpoint := GetDefaultProxyEndpoint()
	rootEndpoint.ProxyType = ProxyTypeRoot
	rootEndpoint.DefaultSiteOption = DefaultSite_Redirect
	rootEndpoint.DefaultSiteValue = "redirect.example.com"
	rootEndpoint.parent = router
	router.Root = &rootEndpoint

	handler := &ProxyHandler{Parent: router}

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	handler.handleRootRouting(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	// Should add http:// prefix
	assert.Equal(t, "http://redirect.example.com", w.Header().Get("Location"))
}

func TestProxyHandler_handleRootRouting_Redirect_Empty(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	rootEndpoint := GetDefaultProxyEndpoint()
	rootEndpoint.ProxyType = ProxyTypeRoot
	rootEndpoint.DefaultSiteOption = DefaultSite_Redirect
	rootEndpoint.DefaultSiteValue = ""
	rootEndpoint.parent = router
	router.Root = &rootEndpoint

	handler := &ProxyHandler{Parent: router}

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	handler.handleRootRouting(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "about:blank", w.Header().Get("Location"))
}

func TestProxyHandler_handleRootRouting_Redirect_Loopback(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	rootEndpoint := GetDefaultProxyEndpoint()
	rootEndpoint.ProxyType = ProxyTypeRoot
	rootEndpoint.DefaultSiteOption = DefaultSite_Redirect
	rootEndpoint.DefaultSiteValue = "http://example.com/redirect"
	rootEndpoint.parent = router
	router.Root = &rootEndpoint

	handler := &ProxyHandler{Parent: router}

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	handler.handleRootRouting(w, req)

	// Should detect loopback and return error
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Loopback redirects")
}

func TestProxyHandler_handleRootRouting_NoResponse(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	rootEndpoint := GetDefaultProxyEndpoint()
	rootEndpoint.ProxyType = ProxyTypeRoot
	rootEndpoint.DefaultSiteOption = DefaultSite_NoResponse
	rootEndpoint.parent = router
	router.Root = &rootEndpoint

	handler := &ProxyHandler{Parent: router}

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	handler.handleRootRouting(w, req)

	// For NoResponse, connection should be set to close
	// In httptest.ResponseRecorder, we can't hijack, so it falls back to WriteHeader
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestProxyHandler_handleRootRouting_Teapot(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	rootEndpoint := GetDefaultProxyEndpoint()
	rootEndpoint.ProxyType = ProxyTypeRoot
	rootEndpoint.DefaultSiteOption = DefaultSite_TeaPot
	rootEndpoint.parent = router
	router.Root = &rootEndpoint

	handler := &ProxyHandler{Parent: router}

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	handler.handleRootRouting(w, req)

	assert.Equal(t, http.StatusTeapot, w.Code)
	assert.Contains(t, w.Body.String(), "teapot")
}

func TestProxyHandler_handleRootRouting_UnknownOption(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	rootEndpoint := GetDefaultProxyEndpoint()
	rootEndpoint.ProxyType = ProxyTypeRoot
	rootEndpoint.DefaultSiteOption = 999 // Invalid option
	rootEndpoint.parent = router
	router.Root = &rootEndpoint

	handler := &ProxyHandler{Parent: router}

	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	w := httptest.NewRecorder()

	handler.handleRootRouting(w, req)

	assert.Equal(t, 544, w.Code)
	assert.Contains(t, w.Body.String(), "No Route Defined")
}

func TestProxyHandler_serve404PageWithTemplate(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	handler := &ProxyHandler{Parent: router}

	req := httptest.NewRequest("GET", "http://example.com/notfound", nil)
	w := httptest.NewRecorder()

	handler.serve404PageWithTemplate(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	// Should use embedded template since file doesn't exist
	assert.NotEmpty(t, w.Body.Bytes())
}

func TestProxyHandler_ServeHTTP_WithHostPort(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Set up root
	rootEndpoint := GetDefaultProxyEndpoint()
	rootEndpoint.ProxyType = ProxyTypeRoot
	rootEndpoint.DefaultSiteOption = DefaultSite_NotFoundPage
	rootEndpoint.parent = router
	preparedRoot, err := router.PrepareProxyRoute(&rootEndpoint)
	require.NoError(t, err)
	router.Root = preparedRoot

	// Create subdomain endpoint
	subEndpoint := GetDefaultProxyEndpoint()
	subEndpoint.ProxyType = ProxyTypeHost
	subEndpoint.RootOrMatchingDomain = "test.example.com"
	subEndpoint.Disabled = true
	subEndpoint.parent = router
	preparedSub, err := router.PrepareProxyRoute(&subEndpoint)
	require.NoError(t, err)
	router.AddProxyRouteToRuntime(preparedSub)

	handler := &ProxyHandler{Parent: router}

	// Test with port in hostname
	req := httptest.NewRequest("GET", "http://test.example.com:8080/", nil)
	req.Host = "test.example.com:8080"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should still match the subdomain (port is stripped for matching)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestProxyHandler_ServeHTTP_TrailingSlashRedirect(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	// Set up root
	rootEndpoint := GetDefaultProxyEndpoint()
	rootEndpoint.ProxyType = ProxyTypeRoot
	rootEndpoint.DefaultSiteOption = DefaultSite_NotFoundPage
	rootEndpoint.parent = router

	// Add virtual directory
	vdir := &VirtualDirectoryEndpoint{
		MatchingPath: "/api/",
		Domain:       "api.local",
		RequireTLS:   false,
		parent:       &rootEndpoint,
	}
	rootEndpoint.VirtualDirectories = []*VirtualDirectoryEndpoint{vdir}

	preparedRoot, err := router.PrepareProxyRoute(&rootEndpoint)
	require.NoError(t, err)
	router.Root = preparedRoot

	handler := &ProxyHandler{Parent: router}

	// Request without trailing slash
	req := httptest.NewRequest("GET", "http://example.com/api", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should redirect to add trailing slash
	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "/api/", w.Header().Get("Location"))
}

func TestProxyHandler_handleRootRouting_WithVirtualDirectory(t *testing.T) {
	option := createTestRouterOption(t)
	router, err := NewDynamicProxy(option)
	require.NoError(t, err)

	rootEndpoint := GetDefaultProxyEndpoint()
	rootEndpoint.ProxyType = ProxyTypeRoot
	rootEndpoint.DefaultSiteOption = DefaultSite_ReverseProxy
	rootEndpoint.parent = router

	// Add a vdir that will be disabled
	vdir := &VirtualDirectoryEndpoint{
		MatchingPath: "/api/",
		Domain:       "api.local",
		RequireTLS:   false,
		Disabled:     true,
		parent:       &rootEndpoint,
	}
	rootEndpoint.VirtualDirectories = []*VirtualDirectoryEndpoint{vdir}

	preparedRoot, err := router.PrepareProxyRoute(&rootEndpoint)
	require.NoError(t, err)
	router.Root = preparedRoot

	handler := &ProxyHandler{Parent: router}

	// Request to disabled vdir
	req := httptest.NewRequest("GET", "http://example.com/api/test", nil)
	w := httptest.NewRecorder()

	handler.handleRootRouting(w, req)

	// Should fail since vdir is disabled and there's no backend
	// The actual behavior depends on the proxy setup
	assert.NotEqual(t, http.StatusOK, w.Code)
}
