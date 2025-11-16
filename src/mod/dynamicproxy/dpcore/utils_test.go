package dpcore

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplaceLocationHost_BasicRedirection(t *testing.T) {
	tests := []struct {
		name           string
		urlString      string
		rrr            *ResponseRewriteRuleSet
		useTLS         bool
		expectedResult string
		expectError    bool
	}{
		{
			name:      "HTTP to HTTPS",
			urlString: "http://backend.local/path",
			rrr: &ResponseRewriteRuleSet{
				ProxyDomain:  "backend.local",
				OriginalHost: "frontend.com",
				UseTLS:       true,
			},
			useTLS:         true,
			expectedResult: "https://frontend.com/path",
			expectError:    false,
		},
		{
			name:      "HTTPS to HTTP",
			urlString: "https://backend.local/path",
			rrr: &ResponseRewriteRuleSet{
				ProxyDomain:  "backend.local",
				OriginalHost: "frontend.com",
				UseTLS:       false,
			},
			useTLS:         false,
			expectedResult: "http://frontend.com/path",
			expectError:    false,
		},
		{
			name:      "Different domain - no rewrite",
			urlString: "http://otherdomain.com/path",
			rrr: &ResponseRewriteRuleSet{
				ProxyDomain:  "backend.local",
				OriginalHost: "frontend.com",
				UseTLS:       false,
			},
			useTLS:         false,
			expectedResult: "http://otherdomain.com/path",
			expectError:    false,
		},
		{
			name:      "With subpath - no host match",
			urlString: "https://backend.local/blog/post",
			rrr: &ResponseRewriteRuleSet{
				ProxyDomain:  "backend.local/blog",
				OriginalHost: "frontend.com",
				UseTLS:       true,
			},
			useTLS:         true,
			// No rewrite because host doesn't match ProxyDomain exactly
			expectedResult: "https://backend.local/blog/post",
			expectError:    false,
		},
		{
			name:      "Port 443 in location - no rewrite due to port",
			urlString: "http://backend.local:443/path",
			rrr: &ResponseRewriteRuleSet{
				ProxyDomain:  "backend.local",
				OriginalHost: "frontend.com",
				UseTLS:       true,
			},
			useTLS:         true,
			expectedResult: "http://backend.local:443/path", // Not rewritten due to port 443
			expectError:    false,
		},
		{
			name:      "Port 80 in location - no rewrite due to port",
			urlString: "http://backend.local:80/path",
			rrr: &ResponseRewriteRuleSet{
				ProxyDomain:  "backend.local",
				OriginalHost: "frontend.com",
				UseTLS:       false,
			},
			useTLS:         false,
			expectedResult: "http://backend.local:80/path", // Not rewritten due to port 80
			expectError:    false,
		},
		{
			name:      "Custom port - no rewrite",
			urlString: "http://backend.local:8080/path",
			rrr: &ResponseRewriteRuleSet{
				ProxyDomain:  "backend.local",
				OriginalHost: "frontend.com",
				UseTLS:       false,
			},
			useTLS:         false,
			expectedResult: "http://backend.local:8080/path",
			expectError:    false,
		},
		{
			name:      "Invalid URL",
			urlString: "://invalid-url",
			rrr: &ResponseRewriteRuleSet{
				ProxyDomain:  "backend.local",
				OriginalHost: "frontend.com",
			},
			useTLS:      false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := replaceLocationHost(tt.urlString, tt.rrr, tt.useTLS)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestIsExternalDomainName(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		expected bool
	}{
		{
			name:     "External domain",
			hostname: "github.com",
			expected: true,
		},
		{
			name:     "External domain with subdomain",
			hostname: "api.example.com",
			expected: true,
		},
		{
			name:     "IP address",
			hostname: "192.168.1.1",
			expected: false,
		},
		{
			name:     "IP with port",
			hostname: "192.168.1.1:8080",
			expected: false,
		},
		{
			name:     ".local domain",
			hostname: "server.local",
			expected: false,
		},
		{
			name:     ".internal domain",
			hostname: "server.internal",
			expected: false,
		},
		{
			name:     ".localhost domain",
			hostname: "test.localhost",
			expected: false,
		},
		{
			name:     ".home.arpa domain",
			hostname: "device.home.arpa",
			expected: false,
		},
		{
			name:     "IPv6 address",
			hostname: "::1",
			expected: false,
		},
		{
			name:     "IPv6 with port",
			hostname: "[::1]:8080",
			expected: false,
		},
		{
			name:     "External domain with port",
			hostname: "example.com:443",
			expected: true,
		},
		{
			name:     ".local uppercase",
			hostname: "SERVER.LOCAL",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isExternalDomainName(tt.hostname)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeepCopyRequest(t *testing.T) {
	// Create original request
	req := httptest.NewRequest("POST", "http://example.com/test?param=value", bytes.NewBufferString("test body"))
	req.Header.Set("X-Custom-Header", "custom-value")
	req.Header.Set("User-Agent", "test-agent")
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})
	req.RemoteAddr = "192.168.1.100:12345"
	req.Host = "example.com"

	// Create deep copy
	reqCopy, err := DeepCopyRequest(req)

	require.NoError(t, err)
	require.NotNil(t, reqCopy)

	// Verify basic fields
	assert.Equal(t, req.Method, reqCopy.Method)
	assert.Equal(t, req.Host, reqCopy.Host)
	assert.Equal(t, req.RemoteAddr, reqCopy.RemoteAddr)
	assert.Equal(t, req.Proto, reqCopy.Proto)
	assert.Equal(t, req.ProtoMajor, reqCopy.ProtoMajor)
	assert.Equal(t, req.ProtoMinor, reqCopy.ProtoMinor)

	// Verify URL is copied
	assert.Equal(t, req.URL.String(), reqCopy.URL.String())
	assert.NotSame(t, req.URL, reqCopy.URL)

	// Verify headers are copied
	assert.Equal(t, req.Header.Get("X-Custom-Header"), reqCopy.Header.Get("X-Custom-Header"))
	assert.Equal(t, req.Header.Get("User-Agent"), reqCopy.Header.Get("User-Agent"))

	// Modify copy headers - should not affect original
	reqCopy.Header.Set("X-Custom-Header", "modified")
	assert.NotEqual(t, req.Header.Get("X-Custom-Header"), reqCopy.Header.Get("X-Custom-Header"))

	// Verify cookies are copied
	assert.Equal(t, len(req.Cookies()), len(reqCopy.Cookies()))
	if len(reqCopy.Cookies()) > 0 {
		assert.Equal(t, req.Cookies()[0].Name, reqCopy.Cookies()[0].Name)
		assert.Equal(t, req.Cookies()[0].Value, reqCopy.Cookies()[0].Value)
	}

	// Verify body is copied
	if req.Body != nil && reqCopy.Body != nil {
		originalBody, _ := io.ReadAll(req.Body)
		copiedBody, _ := io.ReadAll(reqCopy.Body)
		assert.Equal(t, originalBody, copiedBody)
	}
}

func TestDeepCopyRequest_NoBody(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/test", nil)
	req.Header.Set("X-Test", "value")

	reqCopy, err := DeepCopyRequest(req)

	require.NoError(t, err)
	require.NotNil(t, reqCopy)
	assert.Equal(t, req.Method, reqCopy.Method)
	assert.Equal(t, req.Header.Get("X-Test"), reqCopy.Header.Get("X-Test"))
}

func TestDeepCopyRequest_EmptyRequest(t *testing.T) {
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: "/"},
		Header: make(http.Header),
	}

	reqCopy, err := DeepCopyRequest(req)

	require.NoError(t, err)
	require.NotNil(t, reqCopy)
	assert.Equal(t, req.Method, reqCopy.Method)
}

func TestDeepCopyRequest_MultipleHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.Header.Add("X-Multi", "value1")
	req.Header.Add("X-Multi", "value2")
	req.Header.Add("X-Multi", "value3")

	reqCopy, err := DeepCopyRequest(req)

	require.NoError(t, err)
	assert.Equal(t, req.Header["X-Multi"], reqCopy.Header["X-Multi"])
	assert.Equal(t, 3, len(reqCopy.Header["X-Multi"]))
}

func TestDeepCopyRequest_MultipleCookies(t *testing.T) {
	req := httptest.NewRequest("GET", "http://example.com/", nil)
	req.AddCookie(&http.Cookie{Name: "cookie1", Value: "value1"})
	req.AddCookie(&http.Cookie{Name: "cookie2", Value: "value2"})

	reqCopy, err := DeepCopyRequest(req)

	require.NoError(t, err)
	assert.Equal(t, 2, len(reqCopy.Cookies()))
}

func TestReplaceLocationHost_WithQueryString(t *testing.T) {
	urlString := "http://backend.local/path?key=value&foo=bar"
	rrr := &ResponseRewriteRuleSet{
		ProxyDomain:  "backend.local",
		OriginalHost: "frontend.com",
		UseTLS:       true,
	}

	result, err := replaceLocationHost(urlString, rrr, true)

	require.NoError(t, err)
	assert.Contains(t, result, "https://frontend.com/path")
	assert.Contains(t, result, "key=value")
	assert.Contains(t, result, "foo=bar")
}

func TestReplaceLocationHost_WithFragment(t *testing.T) {
	urlString := "http://backend.local/path#section"
	rrr := &ResponseRewriteRuleSet{
		ProxyDomain:  "backend.local",
		OriginalHost: "frontend.com",
		UseTLS:       false,
	}

	result, err := replaceLocationHost(urlString, rrr, false)

	require.NoError(t, err)
	assert.Contains(t, result, "http://frontend.com/path")
	assert.Contains(t, result, "#section")
}

func TestReplaceLocationHost_ApacheStyleRedirect(t *testing.T) {
	// Apache sometimes redirects like: http://example.com -> http://example.com:443
	// Port 80 and 443 are not rewritten according to the logic
	urlString := "http://backend.local:443/path"
	rrr := &ResponseRewriteRuleSet{
		ProxyDomain:  "backend.local",
		OriginalHost: "frontend.com",
		UseTLS:       true,
	}

	result, err := replaceLocationHost(urlString, rrr, true)

	require.NoError(t, err)
	// The function doesn't rewrite when port is 80 or 443
	assert.Equal(t, "http://backend.local:443/path", result)
}

func TestReplaceLocationHost_SubdomainWithPort(t *testing.T) {
	// Subdomain with custom port should not be rewritten
	urlString := "http://sub.backend.local:8080/path"
	rrr := &ResponseRewriteRuleSet{
		ProxyDomain:  "backend.local",
		OriginalHost: "frontend.com",
		UseTLS:       false,
	}

	result, err := replaceLocationHost(urlString, rrr, false)

	require.NoError(t, err)
	// Should not rewrite because of custom port
	assert.Equal(t, urlString, result)
}

func TestIsExternalDomainName_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		expected bool
	}{
		{
			name:     "Empty string",
			hostname: "",
			expected: true, // Empty string is not an IP, so treated as external
		},
		{
			name:     "Single word",
			hostname: "localhost",
			expected: true, // Not ending with .localhost
		},
		{
			name:     "Domain ending with .local",
			hostname: "test.local",
			expected: false,
		},
		{
			name:     "Complex IPv6",
			hostname: "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isExternalDomainName(tt.hostname)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplaceLocationHost_ComplexSubpath(t *testing.T) {
	urlString := "https://backend.local/api/v1/users"
	rrr := &ResponseRewriteRuleSet{
		ProxyDomain:  "backend.local/api",
		OriginalHost: "frontend.com",
		UseTLS:       true,
	}

	result, err := replaceLocationHost(urlString, rrr, true)

	require.NoError(t, err)
	// Host doesn't match ProxyDomain exactly, so no rewrite
	assert.Equal(t, "https://backend.local/api/v1/users", result)
}
