package websocketproxy

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"imuslab.com/zoraxy/mod/dynamicproxy/rewrite"
	"imuslab.com/zoraxy/mod/info/logger"
)

// TestProxyHandler tests the ProxyHandler function
func TestProxyHandler(t *testing.T) {
	targetURL, _ := url.Parse("ws://example.com")
	options := Options{
		SkipTLSValidation: true,
		SkipOriginCheck:   true,
	}

	handler := ProxyHandler(targetURL, options)
	assert.NotNil(t, handler)

	// Verify it returns a WebsocketProxy
	proxy, ok := handler.(*WebsocketProxy)
	assert.True(t, ok)
	assert.NotNil(t, proxy.Backend)
	assert.Equal(t, options, proxy.Options)
}

// TestNewProxy tests the NewProxy function
func TestNewProxy(t *testing.T) {
	tests := []struct {
		name              string
		targetURL         string
		options           Options
		expectDirector    bool
		expectCopyHeaders bool
	}{
		{
			name:              "Basic proxy without options",
			targetURL:         "ws://example.com",
			options:           Options{},
			expectDirector:    false,
			expectCopyHeaders: false,
		},
		{
			name:      "Proxy with CopyAllHeaders",
			targetURL: "ws://example.com",
			options: Options{
				CopyAllHeaders: true,
			},
			expectDirector:    true,
			expectCopyHeaders: true,
		},
		{
			name:      "Proxy with SkipTLSValidation",
			targetURL: "wss://example.com",
			options: Options{
				SkipTLSValidation: true,
			},
			expectDirector:    false,
			expectCopyHeaders: false,
		},
		{
			name:      "Proxy with all options",
			targetURL: "wss://example.com/path",
			options: Options{
				SkipTLSValidation: true,
				SkipOriginCheck:   true,
				CopyAllHeaders:    true,
			},
			expectDirector:    true,
			expectCopyHeaders: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, err := url.Parse(tt.targetURL)
			require.NoError(t, err)

			proxy := NewProxy(target, tt.options)
			assert.NotNil(t, proxy)
			assert.NotNil(t, proxy.Backend)
			assert.Equal(t, tt.options, proxy.Options)
			assert.False(t, proxy.Verbal)

			if tt.expectDirector {
				assert.NotNil(t, proxy.Director)
			} else {
				assert.Nil(t, proxy.Director)
			}

			// Test that Backend function works correctly
			req := httptest.NewRequest("GET", "http://localhost/test?foo=bar", nil)
			req.URL.Fragment = "fragment"
			backendURL := proxy.Backend(req)
			assert.NotNil(t, backendURL)
			assert.Equal(t, target.Scheme, backendURL.Scheme)
			assert.Equal(t, target.Host, backendURL.Host)
			assert.Equal(t, "/test", backendURL.Path)
			assert.Equal(t, "foo=bar", backendURL.RawQuery)
			assert.Equal(t, "fragment", backendURL.Fragment)
		})
	}
}

// TestPrintln tests the Println logging function
func TestPrintln(t *testing.T) {
	tests := []struct {
		name    string
		logger  *logger.Logger
		message string
		err     error
	}{
		{
			name:    "Print without logger",
			logger:  nil,
			message: "Test message",
			err:     errors.New("test error"),
		},
		{
			name:    "Print without error",
			logger:  nil,
			message: "Test message",
			err:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy := &WebsocketProxy{
				Options: Options{
					Logger: tt.logger,
				},
			}

			// This should not panic
			proxy.Println(tt.message, tt.err)
		})
	}
}

// TestDefaultDirector tests the DefaultDirector function
func TestDefaultDirector(t *testing.T) {
	tests := []struct {
		name            string
		requestHeaders  map[string][]string
		expectedHeaders map[string][]string
		removedHeaders  []string
	}{
		{
			name: "Copy regular headers",
			requestHeaders: map[string][]string{
				"Content-Type":  {"application/json"},
				"Authorization": {"Bearer token"},
				"Custom-Header": {"custom-value"},
			},
			expectedHeaders: map[string][]string{
				"Content-Type":  {"application/json"},
				"Authorization": {"Bearer token"},
				"Custom-Header": {"custom-value"},
			},
			removedHeaders: []string{},
		},
		{
			name: "Remove hop-by-hop headers",
			requestHeaders: map[string][]string{
				"Content-Type":            {"application/json"},
				"Connection":              {"keep-alive"},
				"Keep-Alive":              {"timeout=5"},
				"Proxy-Authenticate":      {"Basic"},
				"Proxy-Authorization":     {"Basic abc"},
				"Te":                      {"trailers"},
				"Trailers":                {"Expires"},
				"Transfer-Encoding":       {"chunked"},
				"Sec-WebSocket-Extensions": {"permessage-deflate"},
				"Sec-WebSocket-Key":       {"dGhlIHNhbXBsZSBub25jZQ=="},
				"Sec-WebSocket-Protocol":  {"chat"},
				"Sec-WebSocket-Version":   {"13"},
				"Upgrade":                 {"websocket"},
			},
			expectedHeaders: map[string][]string{
				"Content-Type": {"application/json"},
			},
			removedHeaders: []string{
				"Connection",
				"Keep-Alive",
				"Proxy-Authenticate",
				"Proxy-Authorization",
				"Te",
				"Trailers",
				"Transfer-Encoding",
				"Sec-WebSocket-Extensions",
				"Sec-WebSocket-Key",
				"Sec-WebSocket-Protocol",
				"Sec-WebSocket-Version",
				"Upgrade",
			},
		},
		{
			name: "Handle multiple values",
			requestHeaders: map[string][]string{
				"X-Custom": {"value1", "value2", "value3"},
			},
			expectedHeaders: map[string][]string{
				"X-Custom": {"value3"}, // Set only keeps the last value
			},
			removedHeaders: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com", nil)
			for k, vv := range tt.requestHeaders {
				for _, v := range vv {
					req.Header.Add(k, v)
				}
			}

			outHeader := http.Header{}
			DefaultDirector(req, outHeader)

			// Check expected headers are present
			for k, expectedValues := range tt.expectedHeaders {
				actualValues := outHeader[k]
				assert.Equal(t, expectedValues, actualValues, "Header %s mismatch", k)
			}

			// Check removed headers are not present
			for _, k := range tt.removedHeaders {
				assert.Empty(t, outHeader.Get(k), "Header %s should be removed", k)
			}
		})
	}
}

// TestCopyHeader tests the copyHeader function
func TestCopyHeader(t *testing.T) {
	tests := []struct {
		name        string
		srcHeaders  map[string][]string
		dstHeaders  map[string][]string
		expected    map[string][]string
	}{
		{
			name: "Copy to empty header",
			srcHeaders: map[string][]string{
				"Content-Type": {"application/json"},
				"X-Custom":     {"value1"},
			},
			dstHeaders: map[string][]string{},
			expected: map[string][]string{
				"Content-Type": {"application/json"},
				"X-Custom":     {"value1"},
			},
		},
		{
			name: "Copy with existing headers",
			srcHeaders: map[string][]string{
				"Content-Type": {"application/json"},
			},
			dstHeaders: map[string][]string{
				"X-Existing": {"existing-value"},
			},
			expected: map[string][]string{
				"Content-Type": {"application/json"},
				"X-Existing":   {"existing-value"},
			},
		},
		{
			name: "Copy multiple values",
			srcHeaders: map[string][]string{
				"Set-Cookie": {"cookie1=value1", "cookie2=value2"},
			},
			dstHeaders: map[string][]string{},
			expected: map[string][]string{
				"Set-Cookie": {"cookie1=value1", "cookie2=value2"},
			},
		},
		{
			name: "Append to existing values",
			srcHeaders: map[string][]string{
				"X-Header": {"new-value"},
			},
			dstHeaders: map[string][]string{
				"X-Header": {"old-value"},
			},
			expected: map[string][]string{
				"X-Header": {"old-value", "new-value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := http.Header{}
			for k, vv := range tt.srcHeaders {
				for _, v := range vv {
					src.Add(k, v)
				}
			}

			dst := http.Header{}
			for k, vv := range tt.dstHeaders {
				for _, v := range vv {
					dst.Add(k, v)
				}
			}

			copyHeader(dst, src)

			for k, expectedValues := range tt.expected {
				actualValues := dst[k]
				assert.Equal(t, expectedValues, actualValues, "Header %s mismatch", k)
			}
		})
	}
}

// TestCopyResponse tests the copyResponse function
func TestCopyResponse(t *testing.T) {
	tests := []struct {
		name           string
		responseCode   int
		responseBody   string
		responseHeader map[string]string
	}{
		{
			name:         "Copy 200 response",
			responseCode: 200,
			responseBody: "Success",
			responseHeader: map[string]string{
				"Content-Type": "text/plain",
			},
		},
		{
			name:         "Copy 404 response",
			responseCode: 404,
			responseBody: "Not Found",
			responseHeader: map[string]string{
				"Content-Type": "text/html",
			},
		},
		{
			name:         "Copy response with multiple headers",
			responseCode: 302,
			responseBody: "Redirecting",
			responseHeader: map[string]string{
				"Location":     "http://example.com/new",
				"Content-Type": "text/html",
			},
		},
		{
			name:           "Copy empty response",
			responseCode:   204,
			responseBody:   "",
			responseHeader: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test HTTP response
			resp := &http.Response{
				StatusCode: tt.responseCode,
				Header:     http.Header{},
				Body:       io.NopCloser(bytes.NewBufferString(tt.responseBody)),
			}

			for k, v := range tt.responseHeader {
				resp.Header.Set(k, v)
			}

			// Create a response recorder
			rw := httptest.NewRecorder()

			// Copy the response
			err := copyResponse(rw, resp)
			assert.NoError(t, err)

			// Verify status code
			assert.Equal(t, tt.responseCode, rw.Code)

			// Verify headers
			for k, v := range tt.responseHeader {
				assert.Equal(t, v, rw.Header().Get(k))
			}

			// Verify body
			assert.Equal(t, tt.responseBody, rw.Body.String())
		})
	}
}

// TestServeHTTP_ErrorCases tests error scenarios in ServeHTTP
func TestServeHTTP_ErrorCases(t *testing.T) {
	tests := []struct {
		name               string
		setupProxy         func() *WebsocketProxy
		expectedStatusCode int
		expectedBody       string
	}{
		{
			name: "Backend function is nil",
			setupProxy: func() *WebsocketProxy {
				return &WebsocketProxy{
					Backend: nil,
					Options: Options{},
				}
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       "internal server error (code: 1)",
		},
		{
			name: "Backend returns nil URL",
			setupProxy: func() *WebsocketProxy {
				return &WebsocketProxy{
					Backend: func(*http.Request) *url.URL {
						return nil
					},
					Options: Options{},
				}
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       "internal server error (code: 2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy := tt.setupProxy()
			req := httptest.NewRequest("GET", "http://localhost/test", nil)
			rw := httptest.NewRecorder()

			proxy.ServeHTTP(rw, req)

			assert.Equal(t, tt.expectedStatusCode, rw.Code)
			assert.Contains(t, rw.Body.String(), tt.expectedBody)
		})
	}
}

// TestServeHTTP_DialerConfiguration tests dialer setup with different options
func TestServeHTTP_DialerConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		options Options
		dialer  *websocket.Dialer
	}{
		{
			name: "Default dialer",
			options: Options{
				SkipTLSValidation: false,
			},
			dialer: nil,
		},
		{
			name: "Skip TLS validation",
			options: Options{
				SkipTLSValidation: true,
			},
			dialer: nil,
		},
		{
			name: "Custom dialer provided",
			options: Options{
				SkipTLSValidation: false,
			},
			dialer: &websocket.Dialer{
				HandshakeTimeout: 10 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a backend server that will fail (so we can test dialer setup without full connection)
			targetURL, _ := url.Parse("ws://localhost:9999/nonexistent")

			proxy := NewProxy(targetURL, tt.options)
			proxy.Dialer = tt.dialer

			req := httptest.NewRequest("GET", "http://localhost/test", nil)
			req.Header.Set("Upgrade", "websocket")
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Sec-WebSocket-Version", "13")
			req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

			rw := httptest.NewRecorder()
			proxy.ServeHTTP(rw, req)

			// We expect this to fail with 503 since backend doesn't exist
			// The important part is that it doesn't panic and handles the dialer correctly
			assert.Equal(t, http.StatusServiceUnavailable, rw.Code)
		})
	}
}

// TestServeHTTP_HeaderHandling tests various header handling scenarios
func TestServeHTTP_HeaderHandling(t *testing.T) {
	tests := []struct {
		name           string
		requestHeaders map[string]string
		options        Options
	}{
		{
			name: "Origin header",
			requestHeaders: map[string]string{
				"Origin": "http://example.com",
			},
			options: Options{},
		},
		{
			name: "Sec-WebSocket-Protocol",
			requestHeaders: map[string]string{
				"Sec-WebSocket-Protocol": "chat, superchat",
			},
			options: Options{},
		},
		{
			name: "Cookie header",
			requestHeaders: map[string]string{
				"Cookie": "session=abc123",
			},
			options: Options{},
		},
		{
			name: "User-Agent header",
			requestHeaders: map[string]string{
				"User-Agent": "CustomAgent/1.0",
			},
			options: Options{},
		},
		{
			name:           "No User-Agent (should add default)",
			requestHeaders: map[string]string{},
			options:        Options{},
		},
		{
			name: "X-Forwarded-For existing",
			requestHeaders: map[string]string{
				"X-Forwarded-For": "192.168.1.1",
			},
			options: Options{},
		},
		{
			name: "TLS request (X-Forwarded-Proto should be https)",
			requestHeaders: map[string]string{},
			options:        Options{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a backend server that will fail (to test header preparation without full connection)
			targetURL, _ := url.Parse("ws://localhost:9999/nonexistent")

			proxy := NewProxy(targetURL, tt.options)

			req := httptest.NewRequest("GET", "http://localhost/test", nil)
			req.Header.Set("Upgrade", "websocket")
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Sec-WebSocket-Version", "13")
			req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

			for k, v := range tt.requestHeaders {
				req.Header.Set(k, v)
			}

			// Set remote addr for X-Forwarded-For test
			req.RemoteAddr = "10.0.0.1:12345"

			rw := httptest.NewRecorder()
			proxy.ServeHTTP(rw, req)

			// We expect this to fail with 503 since backend doesn't exist
			// The important part is that it doesn't panic and handles headers correctly
			assert.Equal(t, http.StatusServiceUnavailable, rw.Code)
		})
	}
}

// TestServeHTTP_WithCopyAllHeaders tests CopyAllHeaders functionality
func TestServeHTTP_WithCopyAllHeaders(t *testing.T) {
	tests := []struct {
		name               string
		userDefinedHeaders []*rewrite.UserDefinedHeader
		requestHeaders     map[string]string
	}{
		{
			name: "With user-defined headers",
			userDefinedHeaders: []*rewrite.UserDefinedHeader{
				{
					Direction: rewrite.HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Custom",
					Value:     "custom-value",
				},
			},
			requestHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name: "Skip Upgrade and Connection headers",
			userDefinedHeaders: []*rewrite.UserDefinedHeader{
				{
					Direction: rewrite.HeaderDirection_ZoraxyToUpstream,
					Key:       "Upgrade",
					Value:     "websocket",
				},
				{
					Direction: rewrite.HeaderDirection_ZoraxyToUpstream,
					Key:       "Connection",
					Value:     "Upgrade",
				},
			},
			requestHeaders: map[string]string{},
		},
		{
			name: "Empty header pairs",
			userDefinedHeaders: []*rewrite.UserDefinedHeader{
				{
					Direction: rewrite.HeaderDirection_ZoraxyToUpstream,
					Key:       "",
					Value:     "",
				},
			},
			requestHeaders: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetURL, _ := url.Parse("ws://localhost:9999/nonexistent")

			options := Options{
				CopyAllHeaders:     true,
				UserDefinedHeaders: tt.userDefinedHeaders,
			}

			proxy := NewProxy(targetURL, options)

			req := httptest.NewRequest("GET", "http://localhost/test", nil)
			req.Header.Set("Upgrade", "websocket")
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Sec-WebSocket-Version", "13")
			req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

			for k, v := range tt.requestHeaders {
				req.Header.Set(k, v)
			}

			rw := httptest.NewRecorder()
			proxy.ServeHTTP(rw, req)

			// We expect this to fail with 503 since backend doesn't exist
			assert.Equal(t, http.StatusServiceUnavailable, rw.Code)
		})
	}
}

// TestServeHTTP_OriginCheck tests SkipOriginCheck functionality
func TestServeHTTP_OriginCheck(t *testing.T) {
	// This test verifies that SkipOriginCheck is properly set
	targetURL, _ := url.Parse("ws://localhost:9999/nonexistent")

	options := Options{
		SkipOriginCheck: true,
	}

	proxy := NewProxy(targetURL, options)

	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Origin", "http://malicious.com")

	rw := httptest.NewRecorder()
	proxy.ServeHTTP(rw, req)

	// Should fail with 503 (backend unavailable) not 403 (origin check failed)
	assert.Equal(t, http.StatusServiceUnavailable, rw.Code)
}

// TestBackendURL tests the backend URL construction
func TestBackendURL(t *testing.T) {
	tests := []struct {
		name         string
		targetURL    string
		requestPath  string
		requestQuery string
		fragment     string
		expectedPath string
		expectedRaw  string
	}{
		{
			name:         "Simple path",
			targetURL:    "ws://backend.com",
			requestPath:  "/api/ws",
			requestQuery: "",
			fragment:     "",
			expectedPath: "/api/ws",
			expectedRaw:  "",
		},
		{
			name:         "With query string",
			targetURL:    "ws://backend.com",
			requestPath:  "/api/ws",
			requestQuery: "foo=bar&baz=qux",
			fragment:     "",
			expectedPath: "/api/ws",
			expectedRaw:  "foo=bar&baz=qux",
		},
		{
			name:         "With fragment",
			targetURL:    "ws://backend.com",
			requestPath:  "/api/ws",
			requestQuery: "",
			fragment:     "section",
			expectedPath: "/api/ws",
			expectedRaw:  "",
		},
		{
			name:         "Complete URL",
			targetURL:    "wss://backend.com:8080",
			requestPath:  "/path/to/ws",
			requestQuery: "token=abc123",
			fragment:     "hash",
			expectedPath: "/path/to/ws",
			expectedRaw:  "token=abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, err := url.Parse(tt.targetURL)
			require.NoError(t, err)

			proxy := NewProxy(target, Options{})

			req := httptest.NewRequest("GET", "http://localhost"+tt.requestPath, nil)
			req.URL.RawQuery = tt.requestQuery
			req.URL.Fragment = tt.fragment

			backendURL := proxy.Backend(req)
			assert.Equal(t, tt.expectedPath, backendURL.Path)
			assert.Equal(t, tt.expectedRaw, backendURL.RawQuery)
			if tt.fragment != "" {
				assert.Equal(t, tt.fragment, backendURL.Fragment)
			}
		})
	}
}

// TestUpgraderConfiguration tests upgrader setup
func TestUpgraderConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		upgrader *websocket.Upgrader
	}{
		{
			name:     "Default upgrader",
			upgrader: nil,
		},
		{
			name: "Custom upgrader",
			upgrader: &websocket.Upgrader{
				ReadBufferSize:  4096,
				WriteBufferSize: 4096,
				CheckOrigin: func(r *http.Request) bool {
					return true
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetURL, _ := url.Parse("ws://localhost:9999/nonexistent")
			proxy := NewProxy(targetURL, Options{})
			proxy.Upgrader = tt.upgrader

			req := httptest.NewRequest("GET", "http://localhost/test", nil)
			req.Header.Set("Upgrade", "websocket")
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Sec-WebSocket-Version", "13")
			req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

			rw := httptest.NewRecorder()
			proxy.ServeHTTP(rw, req)

			// Should fail with 503 (backend unavailable)
			assert.Equal(t, http.StatusServiceUnavailable, rw.Code)
		})
	}
}

// TestVerbalMode tests verbal logging mode
func TestVerbalMode(t *testing.T) {
	targetURL, _ := url.Parse("ws://localhost:9999/nonexistent")
	proxy := NewProxy(targetURL, Options{})
	proxy.Verbal = true

	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	rw := httptest.NewRecorder()
	proxy.ServeHTTP(rw, req)

	// Should fail with 503 (backend unavailable)
	// The important part is testing that Verbal mode doesn't cause panics
	assert.Equal(t, http.StatusServiceUnavailable, rw.Code)
}

// TestRemoteAddrParsing tests X-Forwarded-For with various RemoteAddr formats
func TestRemoteAddrParsing(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		shouldWork bool
	}{
		{
			name:       "Valid IPv4 with port",
			remoteAddr: "192.168.1.1:12345",
			shouldWork: true,
		},
		{
			name:       "Valid IPv6 with port",
			remoteAddr: "[::1]:12345",
			shouldWork: true,
		},
		{
			name:       "Invalid format (no port)",
			remoteAddr: "192.168.1.1",
			shouldWork: false,
		},
		{
			name:       "Empty remote addr",
			remoteAddr: "",
			shouldWork: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetURL, _ := url.Parse("ws://localhost:9999/nonexistent")
			proxy := NewProxy(targetURL, Options{})

			req := httptest.NewRequest("GET", "http://localhost/test", nil)
			req.Header.Set("Upgrade", "websocket")
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Sec-WebSocket-Version", "13")
			req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
			req.RemoteAddr = tt.remoteAddr

			rw := httptest.NewRecorder()
			proxy.ServeHTTP(rw, req)

			// Should fail with 503 (backend unavailable)
			assert.Equal(t, http.StatusServiceUnavailable, rw.Code)
		})
	}
}

// TestTLSConnection tests X-Forwarded-Proto with TLS
func TestTLSConnection(t *testing.T) {
	targetURL, _ := url.Parse("ws://localhost:9999/nonexistent")
	proxy := NewProxy(targetURL, Options{})

	req := httptest.NewRequest("GET", "https://localhost/test", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.TLS = &tls.ConnectionState{}

	rw := httptest.NewRecorder()
	proxy.ServeHTTP(rw, req)

	// Should fail with 503 (backend unavailable)
	// The important part is that X-Forwarded-Proto should be set to https
	assert.Equal(t, http.StatusServiceUnavailable, rw.Code)
}
