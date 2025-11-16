package dpcore

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveHeaders_HopByHop(t *testing.T) {
	header := make(http.Header)
	header.Set("Proxy-Connection", "keep-alive")
	header.Set("Keep-Alive", "timeout=5")
	header.Set("Proxy-Authenticate", "Basic")
	header.Set("Proxy-Authorization", "Bearer token")
	header.Set("Te", "trailers")
	header.Set("Trailer", "X-Custom")
	header.Set("Transfer-Encoding", "chunked")
	header.Set("Content-Type", "application/json") // Should NOT be removed

	removeHeaders(header, false)

	// Hop-by-hop headers should be removed
	assert.Empty(t, header.Get("Proxy-Connection"))
	assert.Empty(t, header.Get("Keep-Alive"))
	assert.Empty(t, header.Get("Proxy-Authenticate"))
	assert.Empty(t, header.Get("Proxy-Authorization"))
	assert.Empty(t, header.Get("Te"))
	assert.Empty(t, header.Get("Trailer"))
	assert.Empty(t, header.Get("Transfer-Encoding"))

	// Non-hop-by-hop headers should remain
	assert.Equal(t, "application/json", header.Get("Content-Type"))
}

func TestRemoveHeaders_ConnectionHeader(t *testing.T) {
	header := make(http.Header)
	header.Set("Connection", "X-Custom-1, X-Custom-2")
	header.Set("X-Custom-1", "value1")
	header.Set("X-Custom-2", "value2")
	header.Set("X-Custom-3", "value3") // Not in Connection header

	removeHeaders(header, false)

	// Headers listed in Connection should be removed
	assert.Empty(t, header.Get("X-Custom-1"))
	assert.Empty(t, header.Get("X-Custom-2"))

	// Headers not listed in Connection should remain
	assert.Equal(t, "value3", header.Get("X-Custom-3"))
}

func TestRemoveHeaders_NoCache(t *testing.T) {
	header := make(http.Header)
	header.Set("Cache-Control", "max-age=3600")
	header.Set("Content-Type", "text/html")

	removeHeaders(header, true)

	// Cache-Control should be set to no-store
	assert.Equal(t, "no-store", header.Get("Cache-Control"))
	assert.Equal(t, "text/html", header.Get("Content-Type"))
}

func TestRemoveHeaders_NoCacheFalse(t *testing.T) {
	header := make(http.Header)
	header.Set("Cache-Control", "max-age=3600")

	removeHeaders(header, false)

	// Cache-Control should remain unchanged when noCache is false
	assert.Equal(t, "max-age=3600", header.Get("Cache-Control"))
}

func TestRemoveHeaders_UpgradeHeader(t *testing.T) {
	header := make(http.Header)
	header.Set("Zr-Origin-Upgrade", "websocket")
	header.Set("Connection", "Upgrade")

	removeHeaders(header, false)

	// Zr-Origin-Upgrade should be restored to Upgrade
	assert.Equal(t, "websocket", header.Get("Upgrade"))
	assert.Empty(t, header.Get("Zr-Origin-Upgrade"))
}

func TestRemoveHeaders_EmptyConnectionHeader(t *testing.T) {
	header := make(http.Header)
	header.Set("Connection", "")
	header.Set("X-Custom", "value")

	removeHeaders(header, false)

	// Should not panic with empty Connection header
	assert.Equal(t, "value", header.Get("X-Custom"))
}

func TestRemoveHeaders_ConnectionHeaderWithSpaces(t *testing.T) {
	header := make(http.Header)
	header.Set("Connection", "  X-Header-1  ,  X-Header-2  ")
	header.Set("X-Header-1", "value1")
	header.Set("X-Header-2", "value2")

	removeHeaders(header, false)

	// Should handle spaces properly
	assert.Empty(t, header.Get("X-Header-1"))
	assert.Empty(t, header.Get("X-Header-2"))
}

func TestRewriteUserAgent_EmptyUA(t *testing.T) {
	header := make(http.Header)

	rewriteUserAgent(header, "Zoraxy/1.0")

	// Should set User-Agent when empty
	assert.Equal(t, "Zoraxy/1.0", header.Get("User-Agent"))
}

func TestRewriteUserAgent_ExistingUA(t *testing.T) {
	header := make(http.Header)
	header.Set("User-Agent", "Mozilla/5.0")

	rewriteUserAgent(header, "Zoraxy/1.0")

	// Should NOT overwrite existing User-Agent
	assert.Equal(t, "Mozilla/5.0", header.Get("User-Agent"))
}

func TestRewriteUserAgent_MultipleUACalls(t *testing.T) {
	header := make(http.Header)

	rewriteUserAgent(header, "Zoraxy/1.0")
	rewriteUserAgent(header, "Zoraxy/2.0")

	// First call should set it, second should not change it
	assert.Equal(t, "Zoraxy/1.0", header.Get("User-Agent"))
}

func TestAddXForwardedForHeader_NoExistingHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	addXForwardedForHeader(req)

	assert.Equal(t, "192.168.1.100", req.Header.Get("X-Forwarded-For"))
	assert.Equal(t, "192.168.1.100", req.Header.Get("X-Real-Ip"))
	assert.Equal(t, "http", req.Header.Get("X-Forwarded-Proto"))
}

func TestAddXForwardedForHeader_ExistingHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.1")

	addXForwardedForHeader(req)

	// Should append to existing X-Forwarded-For
	assert.Equal(t, "203.0.113.1, 192.168.1.100", req.Header.Get("X-Forwarded-For"))
	// X-Real-Ip should use first IP
	assert.Equal(t, "203.0.113.1", req.Header.Get("X-Real-Ip"))
}

func TestAddXForwardedForHeader_WithTLS(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.com/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.TLS = &tls.ConnectionState{}

	addXForwardedForHeader(req)

	assert.Equal(t, "https", req.Header.Get("X-Forwarded-Proto"))
}

func TestAddXForwardedForHeader_CloudflareIP(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Set("CF-Connecting-IP", "203.0.113.1")

	addXForwardedForHeader(req)

	// Should use CF-Connecting-IP for X-Real-Ip
	assert.Equal(t, "203.0.113.1", req.Header.Get("X-Real-Ip"))
}

func TestAddXForwardedForHeader_FastlyIP(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Set("Fastly-Client-IP", "203.0.113.1")

	addXForwardedForHeader(req)

	// Should use Fastly-Client-IP for X-Real-Ip
	assert.Equal(t, "203.0.113.1", req.Header.Get("X-Real-Ip"))
}

func TestAddXForwardedForHeader_CloudflarePriority(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Set("CF-Connecting-IP", "203.0.113.1")
	req.Header.Set("Fastly-Client-IP", "203.0.113.2")

	addXForwardedForHeader(req)

	// CF-Connecting-IP should take priority
	assert.Equal(t, "203.0.113.1", req.Header.Get("X-Real-Ip"))
}

func TestAddXForwardedForHeader_ExistingRealIP(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Set("X-Real-Ip", "existing.ip")

	addXForwardedForHeader(req)

	// Should not overwrite existing X-Real-Ip
	assert.Equal(t, "existing.ip", req.Header.Get("X-Real-Ip"))
}

func TestAddXForwardedForHeader_InvalidRemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.RemoteAddr = "invalid-addr"

	// Should not panic
	addXForwardedForHeader(req)

	// Headers should not be set when RemoteAddr is invalid
	assert.Empty(t, req.Header.Get("X-Forwarded-For"))
}

func TestAddXForwardedForHeader_MultipleForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Add("X-Forwarded-For", "203.0.113.1")
	req.Header.Add("X-Forwarded-For", "203.0.113.2")

	addXForwardedForHeader(req)

	// Should join multiple headers
	assert.Contains(t, req.Header.Get("X-Forwarded-For"), "203.0.113.1")
	assert.Contains(t, req.Header.Get("X-Forwarded-For"), "203.0.113.2")
	assert.Contains(t, req.Header.Get("X-Forwarded-For"), "192.168.1.100")
}

func TestInjectUserDefinedHeaders_SetHeaders(t *testing.T) {
	header := make(http.Header)
	userHeaders := [][]string{
		{"X-Custom-1", "value1"},
		{"X-Custom-2", "value2"},
	}

	injectUserDefinedHeaders(header, userHeaders)

	assert.Equal(t, "value1", header.Get("X-Custom-1"))
	assert.Equal(t, "value2", header.Get("X-Custom-2"))
}

func TestInjectUserDefinedHeaders_RemoveHeader(t *testing.T) {
	header := make(http.Header)
	header.Set("X-To-Remove", "value")
	userHeaders := [][]string{
		{"X-To-Remove", ""},
	}

	injectUserDefinedHeaders(header, userHeaders)

	// Empty value should remove the header
	assert.Empty(t, header.Get("X-To-Remove"))
}

func TestInjectUserDefinedHeaders_OverwriteExisting(t *testing.T) {
	header := make(http.Header)
	header.Set("X-Custom", "old-value")
	userHeaders := [][]string{
		{"X-Custom", "new-value"},
	}

	injectUserDefinedHeaders(header, userHeaders)

	// Should overwrite existing header
	assert.Equal(t, "new-value", header.Get("X-Custom"))
}

func TestInjectUserDefinedHeaders_EmptySlice(t *testing.T) {
	header := make(http.Header)
	header.Set("X-Existing", "value")
	userHeaders := [][]string{}

	injectUserDefinedHeaders(header, userHeaders)

	// Should not affect existing headers
	assert.Equal(t, "value", header.Get("X-Existing"))
}

func TestInjectUserDefinedHeaders_EmptyKey(t *testing.T) {
	header := make(http.Header)
	header.Set("X-Test", "value")
	userHeaders := [][]string{
		{}, // Empty slice should cause early return
		{"X-Test", "should-not-be-set"},
	}

	injectUserDefinedHeaders(header, userHeaders)

	// Should return early and not process further headers
	assert.Equal(t, "value", header.Get("X-Test"))
}

func TestInjectUserDefinedHeaders_MultipleHeaders(t *testing.T) {
	header := make(http.Header)
	userHeaders := [][]string{
		{"X-Header-1", "value1"},
		{"X-Header-2", "value2"},
		{"X-Header-3", "value3"},
	}

	injectUserDefinedHeaders(header, userHeaders)

	assert.Equal(t, "value1", header.Get("X-Header-1"))
	assert.Equal(t, "value2", header.Get("X-Header-2"))
	assert.Equal(t, "value3", header.Get("X-Header-3"))
}

func TestRemoveHeaders_AllHopByHopHeaders(t *testing.T) {
	header := make(http.Header)

	// Add all hop-by-hop headers
	for _, h := range hopHeaders {
		header.Set(h, "test-value")
	}
	header.Set("Content-Type", "text/html") // Non-hop-by-hop

	removeHeaders(header, false)

	// All hop-by-hop headers should be removed
	for _, h := range hopHeaders {
		assert.Empty(t, header.Get(h), "Header %s should be removed", h)
	}

	// Non-hop-by-hop should remain
	assert.Equal(t, "text/html", header.Get("Content-Type"))
}

func TestAddXForwardedForHeader_IPv6(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.RemoteAddr = "[2001:db8::1]:12345"

	addXForwardedForHeader(req)

	assert.Equal(t, "2001:db8::1", req.Header.Get("X-Forwarded-For"))
	assert.Equal(t, "2001:db8::1", req.Header.Get("X-Real-Ip"))
}

func TestInjectUserDefinedHeaders_CaseSensitivity(t *testing.T) {
	header := make(http.Header)
	header.Set("x-custom", "old-value")
	userHeaders := [][]string{
		{"X-Custom", "new-value"},
	}

	injectUserDefinedHeaders(header, userHeaders)

	// HTTP headers are case-insensitive, should overwrite
	assert.Equal(t, "new-value", header.Get("X-Custom"))
	assert.Equal(t, "new-value", header.Get("x-custom"))
}
