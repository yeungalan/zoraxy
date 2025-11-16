package reverseproxy

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewReverseProxy tests the creation of a new reverse proxy
func TestNewReverseProxy(t *testing.T) {
	tests := []struct {
		name           string
		targetURL      string
		requestPath    string
		requestQuery   string
		expectedPath   string
		expectedQuery  string
		expectedHost   string
		expectedScheme string
	}{
		{
			name:           "basic http proxy",
			targetURL:      "http://backend.example.com",
			requestPath:    "/api/users",
			requestQuery:   "limit=10",
			expectedPath:   "/api/users",
			expectedQuery:  "limit=10",
			expectedHost:   "backend.example.com",
			expectedScheme: "http",
		},
		{
			name:           "https proxy",
			targetURL:      "https://secure.example.com",
			requestPath:    "/data",
			requestQuery:   "",
			expectedPath:   "/data",
			expectedQuery:  "",
			expectedHost:   "secure.example.com",
			expectedScheme: "https",
		},
		{
			name:           "proxy with base path",
			targetURL:      "http://backend.example.com/base",
			requestPath:    "/dir",
			requestQuery:   "",
			expectedPath:   "/base/dir",
			expectedQuery:  "",
			expectedHost:   "backend.example.com",
			expectedScheme: "http",
		},
		{
			name:           "proxy with base path ending in slash",
			targetURL:      "http://backend.example.com/base/",
			requestPath:    "/dir",
			requestQuery:   "",
			expectedPath:   "/base/dir",
			expectedQuery:  "",
			expectedHost:   "backend.example.com",
			expectedScheme: "http",
		},
		{
			name:           "proxy with query in target",
			targetURL:      "http://backend.example.com?a=10",
			requestPath:    "/test",
			requestQuery:   "b=20",
			expectedPath:   "/test",
			expectedQuery:  "a=10&b=20",
			expectedHost:   "backend.example.com",
			expectedScheme: "http",
		},
		{
			name:           "proxy with query only in target",
			targetURL:      "http://backend.example.com?a=10",
			requestPath:    "/test",
			requestQuery:   "",
			expectedPath:   "/test",
			expectedQuery:  "a=10",
			expectedHost:   "backend.example.com",
			expectedScheme: "http",
		},
		{
			name:           "proxy with query only in request",
			targetURL:      "http://backend.example.com",
			requestPath:    "/test",
			requestQuery:   "b=20",
			expectedPath:   "/test",
			expectedQuery:  "b=20",
			expectedHost:   "backend.example.com",
			expectedScheme: "http",
		},
		{
			name:           "proxy with port",
			targetURL:      "http://backend.example.com:8080",
			requestPath:    "/api",
			requestQuery:   "",
			expectedPath:   "/api",
			expectedQuery:  "",
			expectedHost:   "backend.example.com:8080",
			expectedScheme: "http",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, err := url.Parse(tt.targetURL)
			require.NoError(t, err)

			proxy := NewReverseProxy(target)
			require.NotNil(t, proxy)
			require.NotNil(t, proxy.Director)
			assert.False(t, proxy.Verbal)

			// Create a test request
			req := httptest.NewRequest(http.MethodGet, "http://original.example.com"+tt.requestPath+"?"+tt.requestQuery, nil)

			// Apply director
			proxy.Director(req)

			assert.Equal(t, tt.expectedScheme, req.URL.Scheme)
			assert.Equal(t, tt.expectedHost, req.URL.Host)
			assert.Equal(t, tt.expectedPath, req.URL.Path)
			assert.Equal(t, tt.expectedQuery, req.URL.RawQuery)
			assert.Equal(t, tt.expectedHost, req.Host)
		})
	}
}

// TestSingleJoiningSlash tests the path joining logic
func TestSingleJoiningSlash(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected string
	}{
		{
			name:     "both with slashes",
			a:        "/base/",
			b:        "/path",
			expected: "/base/path",
		},
		{
			name:     "neither with slashes",
			a:        "base",
			b:        "path",
			expected: "base/path",
		},
		{
			name:     "a with slash, b without",
			a:        "base/",
			b:        "path",
			expected: "base/path",
		},
		{
			name:     "a without slash, b with",
			a:        "base",
			b:        "/path",
			expected: "base/path",
		},
		{
			name:     "empty a",
			a:        "",
			b:        "/path",
			expected: "/path",
		},
		{
			name:     "empty b",
			a:        "/base/",
			b:        "",
			expected: "/base/",
		},
		{
			name:     "both empty",
			a:        "",
			b:        "",
			expected: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := singleJoiningSlash(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCopyHeader tests header copying functionality
func TestCopyHeader(t *testing.T) {
	tests := []struct {
		name     string
		src      http.Header
		existing http.Header
		expected http.Header
	}{
		{
			name: "copy single header",
			src: http.Header{
				"Content-Type": []string{"application/json"},
			},
			existing: http.Header{},
			expected: http.Header{
				"Content-Type": []string{"application/json"},
			},
		},
		{
			name: "copy multiple values",
			src: http.Header{
				"Set-Cookie": []string{"session=abc", "token=xyz"},
			},
			existing: http.Header{},
			expected: http.Header{
				"Set-Cookie": []string{"session=abc", "token=xyz"},
			},
		},
		{
			name: "append to existing header",
			src: http.Header{
				"X-Custom": []string{"value1"},
			},
			existing: http.Header{
				"X-Custom": []string{"value0"},
			},
			expected: http.Header{
				"X-Custom": []string{"value0", "value1"},
			},
		},
		{
			name: "copy multiple headers",
			src: http.Header{
				"Content-Type":   []string{"application/json"},
				"Content-Length": []string{"123"},
				"X-Custom":       []string{"value"},
			},
			existing: http.Header{},
			expected: http.Header{
				"Content-Type":   []string{"application/json"},
				"Content-Length": []string{"123"},
				"X-Custom":       []string{"value"},
			},
		},
		{
			name:     "copy from empty header",
			src:      http.Header{},
			existing: http.Header{"X-Existing": []string{"value"}},
			expected: http.Header{"X-Existing": []string{"value"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := tt.existing
			if dst == nil {
				dst = make(http.Header)
			}
			copyHeader(dst, tt.src)
			assert.Equal(t, tt.expected, dst)
		})
	}
}

// TestRemoveHeaders tests hop-by-hop header removal
func TestRemoveHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    http.Header
		expected http.Header
	}{
		{
			name: "remove hop-by-hop headers",
			input: http.Header{
				"Content-Type":      []string{"application/json"},
				"Proxy-Connection":  []string{"keep-alive"},
				"Keep-Alive":        []string{"timeout=5"},
				"Proxy-Authenticate": []string{"Basic"},
				"Proxy-Authorization": []string{"Bearer token"},
				"Te":                []string{"trailers"},
				"Trailer":           []string{"Expires"},
				"Transfer-Encoding": []string{"chunked"},
			},
			expected: http.Header{
				"Content-Type": []string{"application/json"},
			},
		},
		{
			name: "remove headers listed in Connection",
			input: http.Header{
				"Connection":   []string{"X-Custom, X-Another"},
				"X-Custom":     []string{"value1"},
				"X-Another":    []string{"value2"},
				"Content-Type": []string{"text/html"},
			},
			expected: http.Header{
				"Connection":   []string{"X-Custom, X-Another"},
				"Content-Type": []string{"text/html"},
			},
		},
		{
			name: "restore Upgrade from Zr-Origin-Upgrade",
			input: http.Header{
				"Zr-Origin-Upgrade": []string{"websocket"},
				"Content-Type":      []string{"text/html"},
			},
			expected: http.Header{
				"Upgrade":      []string{"websocket"},
				"Content-Type": []string{"text/html"},
			},
		},
		{
			name: "multiple headers in Connection with spaces",
			input: http.Header{
				"Connection":   []string{"  X-Foo  ,  X-Bar  "},
				"X-Foo":        []string{"value1"},
				"X-Bar":        []string{"value2"},
				"Content-Type": []string{"text/html"},
			},
			expected: http.Header{
				"Connection":   []string{"  X-Foo  ,  X-Bar  "},
				"Content-Type": []string{"text/html"},
			},
		},
		{
			name:     "no headers to remove",
			input:    http.Header{"Content-Type": []string{"application/json"}},
			expected: http.Header{"Content-Type": []string{"application/json"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			removeHeaders(tt.input)
			assert.Equal(t, tt.expected, tt.input)
		})
	}
}

// TestAddXForwardedForHeader tests X-Forwarded-For header handling
func TestAddXForwardedForHeader(t *testing.T) {
	tests := []struct {
		name           string
		remoteAddr     string
		existingHeader []string
		expectedHeader string
	}{
		{
			name:           "add new X-Forwarded-For",
			remoteAddr:     "192.168.1.1:12345",
			existingHeader: nil,
			expectedHeader: "192.168.1.1",
		},
		{
			name:           "append to existing X-Forwarded-For",
			remoteAddr:     "192.168.1.2:12345",
			existingHeader: []string{"10.0.0.1"},
			expectedHeader: "10.0.0.1, 192.168.1.2",
		},
		{
			name:           "append to multiple existing values",
			remoteAddr:     "192.168.1.3:12345",
			existingHeader: []string{"10.0.0.1", "10.0.0.2"},
			expectedHeader: "10.0.0.1, 10.0.0.2, 192.168.1.3",
		},
		{
			name:           "IPv6 address",
			remoteAddr:     "[::1]:12345",
			existingHeader: nil,
			expectedHeader: "::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.existingHeader != nil {
				req.Header["X-Forwarded-For"] = tt.existingHeader
			}

			addXForwardedForHeader(req)

			assert.Equal(t, tt.expectedHeader, req.Header.Get("X-Forwarded-For"))
		})
	}
}

// TestProxyHTTP tests HTTP proxying functionality
func TestProxyHTTP(t *testing.T) {
	// Create a backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", "test")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer backend.Close()

	t.Run("successful proxy", func(t *testing.T) {
		backendURL, _ := url.Parse(backend.URL)
		proxy := NewReverseProxy(backendURL)

		req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
		req.Header.Set("X-Custom", "value")
		rw := httptest.NewRecorder()

		err := proxy.ProxyHTTP(rw, req)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, rw.Code)
		assert.Equal(t, "test", rw.Header().Get("X-Backend"))
		assert.Equal(t, `{"status":"ok"}`, rw.Body.String())
	})

	t.Run("with custom transport", func(t *testing.T) {
		backendURL, _ := url.Parse(backend.URL)
		proxy := NewReverseProxy(backendURL)
		proxy.Transport = http.DefaultTransport

		req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
		rw := httptest.NewRecorder()

		err := proxy.ProxyHTTP(rw, req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rw.Code)
	})

	t.Run("with verbal logging", func(t *testing.T) {
		backendURL, _ := url.Parse(backend.URL)
		proxy := NewReverseProxy(backendURL)
		proxy.Verbal = true

		var logBuf bytes.Buffer
		proxy.ErrorLog = log.New(&logBuf, "", 0)

		req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
		rw := httptest.NewRecorder()

		err := proxy.ProxyHTTP(rw, req)
		require.NoError(t, err)
	})

	t.Run("transport error", func(t *testing.T) {
		// Use an invalid URL that will cause connection failure
		backendURL, _ := url.Parse("http://invalid-backend-that-does-not-exist.local:99999")
		proxy := NewReverseProxy(backendURL)
		proxy.Verbal = true

		var logBuf bytes.Buffer
		proxy.ErrorLog = log.New(&logBuf, "", 0)

		req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
		rw := httptest.NewRecorder()

		err := proxy.ProxyHTTP(rw, req)
		assert.Error(t, err)
	})

	t.Run("with ModifyResponse", func(t *testing.T) {
		backendURL, _ := url.Parse(backend.URL)
		proxy := NewReverseProxy(backendURL)
		proxy.ModifyResponse = func(resp *http.Response) error {
			resp.Header.Set("X-Modified", "true")
			return nil
		}

		req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
		rw := httptest.NewRecorder()

		err := proxy.ProxyHTTP(rw, req)
		require.NoError(t, err)
		assert.Equal(t, "true", rw.Header().Get("X-Modified"))
	})

	t.Run("ModifyResponse error", func(t *testing.T) {
		backendURL, _ := url.Parse(backend.URL)
		proxy := NewReverseProxy(backendURL)
		proxy.Verbal = true
		proxy.ModifyResponse = func(resp *http.Response) error {
			return errors.New("modification failed")
		}

		var logBuf bytes.Buffer
		proxy.ErrorLog = log.New(&logBuf, "", 0)

		req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
		rw := httptest.NewRecorder()

		err := proxy.ProxyHTTP(rw, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "modification failed")
	})
}

// TestProxyHTTPWithTrailers tests trailer handling
func TestProxyHTTPWithTrailers(t *testing.T) {
	// Create a backend server that sends trailers
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Trailer", "X-Trailer-Header")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response body"))
		w.Header().Set("X-Trailer-Header", "trailer-value")
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	proxy := NewReverseProxy(backendURL)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	rw := httptest.NewRecorder()

	err := proxy.ProxyHTTP(rw, req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rw.Code)
}

// TestProxyHTTPRemovesHopByHopHeaders tests that hop-by-hop headers are removed
func TestProxyHTTPRemovesHopByHopHeaders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that hop-by-hop headers were removed from request
		assert.Empty(t, r.Header.Get("Proxy-Connection"))
		assert.Empty(t, r.Header.Get("Keep-Alive"))

		// Check that X-Forwarded-For was added
		assert.NotEmpty(t, r.Header.Get("X-Forwarded-For"))

		w.Header().Set("Proxy-Connection", "should-be-removed")
		w.Header().Set("Keep-Alive", "should-be-removed")
		w.Header().Set("X-Custom", "should-remain")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	proxy := NewReverseProxy(backendURL)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	req.Header.Set("Proxy-Connection", "keep-alive")
	req.Header.Set("Keep-Alive", "timeout=5")
	req.Header.Set("X-Custom", "value")
	rw := httptest.NewRecorder()

	err := proxy.ProxyHTTP(rw, req)
	require.NoError(t, err)

	// Check that hop-by-hop headers were removed from response
	assert.Empty(t, rw.Header().Get("Proxy-Connection"))
	assert.Empty(t, rw.Header().Get("Keep-Alive"))
	assert.Equal(t, "should-remain", rw.Header().Get("X-Custom"))
}

// TestServeHTTP tests the main ServeHTTP entry point
func TestServeHTTP(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer backend.Close()

	t.Run("GET request", func(t *testing.T) {
		backendURL, _ := url.Parse(backend.URL)
		proxy := NewReverseProxy(backendURL)

		req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
		rw := httptest.NewRecorder()

		err := proxy.ServeHTTP(rw, req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rw.Code)
	})

	t.Run("POST request", func(t *testing.T) {
		backendURL, _ := url.Parse(backend.URL)
		proxy := NewReverseProxy(backendURL)

		req := httptest.NewRequest(http.MethodPost, "http://example.com/test", strings.NewReader("body"))
		rw := httptest.NewRecorder()

		err := proxy.ServeHTTP(rw, req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, rw.Code)
	})
}

// TestMaxLatencyWriter tests the flush interval writer
func TestMaxLatencyWriter(t *testing.T) {
	t.Run("write", func(t *testing.T) {
		var buf bytes.Buffer
		wf := &testWriteFlusher{Writer: &buf}

		mlw := &maxLatencyWriter{
			dst:     wf,
			latency: 10 * time.Millisecond,
			done:    make(chan bool),
		}

		// Write some data
		n, err := mlw.Write([]byte("test"))
		assert.NoError(t, err)
		assert.Equal(t, 4, n)
		assert.Equal(t, "test", buf.String())
	})

	t.Run("stop sends done signal", func(t *testing.T) {
		var buf bytes.Buffer
		wf := &testWriteFlusher{Writer: &buf}

		mlw := &maxLatencyWriter{
			dst:     wf,
			latency: 100 * time.Millisecond,
			done:    make(chan bool),
		}

		// Start a goroutine that waits for done signal
		doneCalled := make(chan bool, 1)
		go func() {
			<-mlw.done
			doneCalled <- true
		}()

		// Stop should send to done channel
		mlw.stop()

		// Verify done was called
		select {
		case <-doneCalled:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("stop() did not send to done channel")
		}
	})
}

// TestCopyResponse tests the copyResponse method
func TestCopyResponse(t *testing.T) {
	t.Run("copy without flush interval", func(t *testing.T) {
		proxy := &ReverseProxy{
			FlushInterval: 0,
		}

		src := strings.NewReader("test data")
		var dst bytes.Buffer

		proxy.copyResponse(&dst, src)
		assert.Equal(t, "test data", dst.String())
	})

	t.Run("copy with flush interval and flusher", func(t *testing.T) {
		proxy := &ReverseProxy{
			FlushInterval: 10 * time.Millisecond,
		}

		src := strings.NewReader("test data with flushing")
		wf := &testWriteFlusher{Writer: &bytes.Buffer{}}

		proxy.copyResponse(wf, src)
		// Just verify data was copied correctly
		assert.Equal(t, "test data with flushing", wf.Writer.(*bytes.Buffer).String())
	})

	t.Run("copy with non-flusher writer", func(t *testing.T) {
		proxy := &ReverseProxy{
			FlushInterval: 10 * time.Millisecond,
		}

		src := strings.NewReader("test data")
		var dst bytes.Buffer

		// Should not panic when dst is not a writeFlusher
		proxy.copyResponse(&dst, src)
		assert.Equal(t, "test data", dst.String())
	})
}

// TestLogf tests the logging functionality
func TestLogf(t *testing.T) {
	t.Run("log with custom logger", func(t *testing.T) {
		var buf bytes.Buffer
		proxy := &ReverseProxy{
			ErrorLog: log.New(&buf, "", 0),
		}

		proxy.logf("test message: %s", "hello")
		assert.Contains(t, buf.String(), "test message: hello")
	})

	t.Run("log without custom logger", func(t *testing.T) {
		proxy := &ReverseProxy{}
		// Should not panic when ErrorLog is nil
		proxy.logf("test message")
	})
}

// TestProxyHTTPWithDifferentMethods tests various HTTP methods
func TestProxyHTTPWithDifferentMethods(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
		http.MethodHead,
		http.MethodOptions,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			receivedMethod := ""
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedMethod = r.Method
				w.WriteHeader(http.StatusOK)
			}))
			defer backend.Close()

			backendURL, _ := url.Parse(backend.URL)
			proxy := NewReverseProxy(backendURL)

			var body io.Reader
			if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
				body = strings.NewReader("test body")
			}

			req := httptest.NewRequest(method, "http://example.com/test", body)
			rw := httptest.NewRecorder()

			err := proxy.ProxyHTTP(rw, req)
			require.NoError(t, err)
			assert.Equal(t, method, receivedMethod)
		})
	}
}

// TestProxyHTTPWithLargeResponse tests handling of large responses
func TestProxyHTTPWithLargeResponse(t *testing.T) {
	// Create a large response (1MB)
	largeData := strings.Repeat("x", 1024*1024)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeData))
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	proxy := NewReverseProxy(backendURL)
	proxy.FlushInterval = 10 * time.Millisecond

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	rw := httptest.NewRecorder()

	err := proxy.ProxyHTTP(rw, req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rw.Code)
	assert.Equal(t, len(largeData), len(rw.Body.String()))
}

// TestProxyHTTPWithTimeout tests timeout behavior
func TestProxyHTTPWithTimeout(t *testing.T) {
	// Create a backend that delays response
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	proxy := NewReverseProxy(backendURL)

	// Use a transport with a very short timeout
	transport := &http.Transport{
		ResponseHeaderTimeout: 10 * time.Millisecond,
	}
	proxy.Transport = transport

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	rw := httptest.NewRecorder()

	err := proxy.ProxyHTTP(rw, req)
	assert.Error(t, err)
}

// TestProxyWithCustomDirector tests using a custom director
func TestProxyWithCustomDirector(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	proxy := NewReverseProxy(backendURL)

	// Add custom director logic
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("X-Custom-Header", "custom-value")
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	rw := httptest.NewRecorder()

	err := proxy.ProxyHTTP(rw, req)
	require.NoError(t, err)
}

// TestProxyHTTPStatusCodes tests various status codes
func TestProxyHTTPStatusCodes(t *testing.T) {
	statusCodes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusNoContent,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
	}

	for _, code := range statusCodes {
		t.Run(http.StatusText(code), func(t *testing.T) {
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			}))
			defer backend.Close()

			backendURL, _ := url.Parse(backend.URL)
			proxy := NewReverseProxy(backendURL)

			req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
			rw := httptest.NewRecorder()

			err := proxy.ProxyHTTP(rw, req)
			require.NoError(t, err)
			assert.Equal(t, code, rw.Code)
		})
	}
}

// TestReverseProxyFields tests ReverseProxy struct fields
func TestReverseProxyFields(t *testing.T) {
	t.Run("default timeout", func(t *testing.T) {
		proxy := &ReverseProxy{}
		assert.Equal(t, time.Duration(0), proxy.Timeout)
	})

	t.Run("custom timeout", func(t *testing.T) {
		proxy := &ReverseProxy{
			Timeout: 10 * time.Second,
		}
		assert.Equal(t, 10*time.Second, proxy.Timeout)
	})

	t.Run("custom flush interval", func(t *testing.T) {
		proxy := &ReverseProxy{
			FlushInterval: 50 * time.Millisecond,
		}
		assert.Equal(t, 50*time.Millisecond, proxy.FlushInterval)
	})
}

// Helper type for testing writeFlusher
type testWriteFlusher struct {
	io.Writer
	flushed bool
	mu      sync.Mutex
}

func (t *testWriteFlusher) Flush() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.flushed = true
}

// TestProxyWithRequestBody tests proxying requests with bodies
func TestProxyWithRequestBody(t *testing.T) {
	receivedBody := ""
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	proxy := NewReverseProxy(backendURL)

	requestBody := "test request body"
	req := httptest.NewRequest(http.MethodPost, "http://example.com/test", strings.NewReader(requestBody))
	rw := httptest.NewRecorder()

	err := proxy.ProxyHTTP(rw, req)
	require.NoError(t, err)
	assert.Equal(t, requestBody, receivedBody)
	assert.Equal(t, "response", rw.Body.String())
}

// TestProxyPreservesRequestHeaders tests that important headers are preserved
func TestProxyPreservesRequestHeaders(t *testing.T) {
	receivedHeaders := make(http.Header)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range r.Header {
			receivedHeaders[k] = v
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	proxy := NewReverseProxy(backendURL)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()

	err := proxy.ProxyHTTP(rw, req)
	require.NoError(t, err)

	assert.Equal(t, "test-agent", receivedHeaders.Get("User-Agent"))
	assert.Equal(t, "application/json", receivedHeaders.Get("Accept"))
	assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	t.Run("nil director", func(t *testing.T) {
		proxy := &ReverseProxy{
			Director: nil,
		}

		req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
		rw := httptest.NewRecorder()

		// Should panic with nil director
		assert.Panics(t, func() {
			proxy.ProxyHTTP(rw, req)
		})
	})

	t.Run("empty URL path", func(t *testing.T) {
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer backend.Close()

		backendURL, _ := url.Parse(backend.URL)
		proxy := NewReverseProxy(backendURL)

		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		rw := httptest.NewRecorder()

		err := proxy.ProxyHTTP(rw, req)
		require.NoError(t, err)
	})

	t.Run("request with fragment", func(t *testing.T) {
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer backend.Close()

		backendURL, _ := url.Parse(backend.URL)
		proxy := NewReverseProxy(backendURL)

		req := httptest.NewRequest(http.MethodGet, "http://example.com/test#fragment", nil)
		rw := httptest.NewRecorder()

		err := proxy.ProxyHTTP(rw, req)
		require.NoError(t, err)
	})
}

// TestConnectionHeaderHandling tests Connection header edge cases
func TestConnectionHeaderHandling(t *testing.T) {
	t.Run("empty connection header", func(t *testing.T) {
		header := http.Header{
			"Connection":   []string{""},
			"Content-Type": []string{"text/html"},
		}
		removeHeaders(header)
		assert.Equal(t, "text/html", header.Get("Content-Type"))
	})

	t.Run("connection header with only spaces", func(t *testing.T) {
		header := http.Header{
			"Connection":   []string{"   "},
			"Content-Type": []string{"text/html"},
		}
		removeHeaders(header)
		assert.Equal(t, "text/html", header.Get("Content-Type"))
	})

	t.Run("connection header with empty elements", func(t *testing.T) {
		header := http.Header{
			"Connection":   []string{",,,"},
			"Content-Type": []string{"text/html"},
		}
		removeHeaders(header)
		assert.Equal(t, "text/html", header.Get("Content-Type"))
	})
}

// TestXForwardedForEdgeCases tests X-Forwarded-For edge cases
func TestXForwardedForEdgeCases(t *testing.T) {
	t.Run("invalid remote addr", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		req.RemoteAddr = "invalid"

		addXForwardedForHeader(req)
		// Should not panic, header should not be set
		assert.Empty(t, req.Header.Get("X-Forwarded-For"))
	})

	t.Run("remote addr without port", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
		req.RemoteAddr = "192.168.1.1"

		addXForwardedForHeader(req)
		// Should not add header when port is missing
		assert.Empty(t, req.Header.Get("X-Forwarded-For"))
	})
}

// TestProxyHTTPCloseConnection tests connection closing behavior
func TestProxyHTTPCloseConnection(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that Close is set to false
		assert.False(t, r.Close)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	proxy := NewReverseProxy(backendURL)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
	req.Close = true // Set to true in original request
	rw := httptest.NewRecorder()

	err := proxy.ProxyHTTP(rw, req)
	require.NoError(t, err)
}

// TestDefaultTimeout tests the default timeout constant
func TestDefaultTimeout(t *testing.T) {
	assert.Equal(t, time.Minute*5, defaultTimeout)
}
