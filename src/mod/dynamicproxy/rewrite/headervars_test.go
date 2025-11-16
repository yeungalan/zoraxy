package rewrite

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetHeaderVariableValuesFromRequest(t *testing.T) {
	tests := []struct {
		name     string
		setupReq func() *httptest.ResponseRecorder
		validate func(*testing.T, map[string]string)
	}{
		{
			name: "Basic GET request with all headers",
			setupReq: func() *httptest.ResponseRecorder {
				return nil
			},
			validate: func(t *testing.T, vars map[string]string) {
				req := httptest.NewRequest("GET", "https://example.com/test?foo=bar", nil)
				req.Host = "example.com"
				req.RemoteAddr = "192.168.1.1:12345"
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("User-Agent", "TestAgent")
				req.Header.Set("Referer", "https://referer.com")

				vars = GetHeaderVariableValuesFromRequest(req)

				assert.Equal(t, "example.com", vars["$host"])
				assert.Equal(t, "192.168.1.1:12345", vars["$remote_addr"])
				assert.Equal(t, "192.168.1.1", vars["$remote_ip"])
				assert.Equal(t, "https://example.com/test?foo=bar", vars["$request_uri"])
				assert.Equal(t, "GET", vars["$request_method"])
				assert.Equal(t, "0", vars["$content_length"])
				assert.Equal(t, "application/json", vars["$content_type"])
				assert.Equal(t, "/test", vars["$uri"])
				assert.Equal(t, "foo=bar", vars["$args"])
				assert.Equal(t, "https", vars["$scheme"])
				assert.Equal(t, "foo=bar", vars["$query_string"])
				assert.Equal(t, "TestAgent", vars["$http_user_agent"])
				assert.Equal(t, "https://referer.com", vars["$http_referer"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, nil)
		})
	}
}

func TestGetHeaderVariableValuesFromRequest_EdgeCases(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		url              string
		remoteAddr       string
		headers          map[string]string
		expectedVars     map[string]string
		checkRemoteIP    bool
		expectedRemoteIP string
	}{
		{
			name:       "POST request with body",
			method:     "POST",
			url:        "http://api.example.com/users?page=1&limit=10",
			remoteAddr: "10.0.0.5:54321",
			headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
				"User-Agent":   "Mozilla/5.0",
			},
			expectedVars: map[string]string{
				"$host":            "api.example.com",
				"$remote_addr":     "10.0.0.5:54321",
				"$request_uri":     "http://api.example.com/users?page=1&limit=10",
				"$request_method":  "POST",
				"$uri":             "/users",
				"$args":            "page=1&limit=10",
				"$query_string":    "page=1&limit=10",
				"$scheme":          "http",
				"$content_type":    "application/x-www-form-urlencoded",
				"$http_user_agent": "Mozilla/5.0",
			},
			checkRemoteIP:    true,
			expectedRemoteIP: "10.0.0.5",
		},
		{
			name:       "Request without query string",
			method:     "GET",
			url:        "https://example.com/path/to/resource",
			remoteAddr: "192.168.0.100:8080",
			headers:    map[string]string{},
			expectedVars: map[string]string{
				"$host":           "example.com",
				"$request_uri":    "https://example.com/path/to/resource",
				"$uri":            "/path/to/resource",
				"$args":           "",
				"$query_string":   "",
				"$scheme":         "https",
				"$request_method": "GET",
			},
			checkRemoteIP:    true,
			expectedRemoteIP: "192.168.0.100",
		},
		{
			name:       "Request without scheme (relative URL)",
			method:     "GET",
			url:        "/api/endpoint?test=1",
			remoteAddr: "127.0.0.1:9999",
			headers:    map[string]string{},
			expectedVars: map[string]string{
				"$request_uri":    "/api/endpoint?test=1",
				"$uri":            "/api/endpoint",
				"$args":           "test=1",
				"$query_string":   "test=1",
				"$scheme":         "",
				"$request_method": "GET",
			},
		},
		{
			name:       "Request with IPv6 address",
			method:     "GET",
			url:        "https://example.com/test",
			remoteAddr: "[::1]:8080",
			headers:    map[string]string{},
			expectedVars: map[string]string{
				"$host":        "example.com",
				"$request_uri": "https://example.com/test",
				"$remote_addr": "[::1]:8080",
			},
			checkRemoteIP:    true,
			expectedRemoteIP: "::1",
		},
		{
			name:       "Request with malformed RemoteAddr (no port)",
			method:     "GET",
			url:        "https://example.com/test",
			remoteAddr: "192.168.1.1",
			headers:    map[string]string{},
			expectedVars: map[string]string{
				"$request_uri": "https://example.com/test",
				"$remote_addr": "192.168.1.1",
			},
			checkRemoteIP:    true,
			expectedRemoteIP: "192.168.1.1", // Should fallback to full RemoteAddr
		},
		{
			name:       "Request without User-Agent and Referer",
			method:     "DELETE",
			url:        "https://api.example.com/resource/123",
			remoteAddr: "203.0.113.5:12345",
			headers:    map[string]string{},
			expectedVars: map[string]string{
				"$request_uri":     "https://api.example.com/resource/123",
				"$http_user_agent": "",
				"$http_referer":    "",
				"$request_method":  "DELETE",
			},
		},
		{
			name:       "Request with empty Content-Type",
			method:     "PUT",
			url:        "https://example.com/upload",
			remoteAddr: "198.51.100.10:3000",
			headers:    map[string]string{},
			expectedVars: map[string]string{
				"$request_uri":    "https://example.com/upload",
				"$content_type":   "",
				"$request_method": "PUT",
			},
		},
		{
			name:       "Request with complex query parameters",
			method:     "GET",
			url:        "https://search.example.com/search?q=test+query&sort=desc&filter[]=a&filter[]=b",
			remoteAddr: "172.16.0.1:5000",
			headers:    map[string]string{},
			expectedVars: map[string]string{
				"$request_uri":  "https://search.example.com/search?q=test+query&sort=desc&filter[]=a&filter[]=b",
				"$args":         "q=test+query&sort=desc&filter[]=a&filter[]=b",
				"$query_string": "q=test+query&sort=desc&filter[]=a&filter[]=b",
				"$uri":          "/search",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.RemoteAddr = tt.remoteAddr

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			vars := GetHeaderVariableValuesFromRequest(req)

			for key, expectedValue := range tt.expectedVars {
				assert.Equal(t, expectedValue, vars[key], "Variable %s mismatch", key)
			}

			if tt.checkRemoteIP {
				assert.Equal(t, tt.expectedRemoteIP, vars["$remote_ip"], "Remote IP mismatch")
			}
		})
	}
}

func TestCustomHeadersIncludeDynamicVariables(t *testing.T) {
	tests := []struct {
		name           string
		headers        []*UserDefinedHeader
		expectedHasVar bool
	}{
		{
			name:           "No headers (nil slice)",
			headers:        nil,
			expectedHasVar: false,
		},
		{
			name:           "No headers (empty slice)",
			headers:        []*UserDefinedHeader{},
			expectedHasVar: false,
		},
		{
			name: "Headers without dynamic variables",
			headers: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Custom-Header",
					Value:     "staticValue",
					IsRemove:  false,
				},
				{
					Direction: HeaderDirection_ZoraxyToDownstream,
					Key:       "X-Another-Header",
					Value:     "staticValue",
					IsRemove:  false,
				},
			},
			expectedHasVar: false,
		},
		{
			name: "Headers with one dynamic variable",
			headers: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Custom-Header",
					Value:     "$host",
					IsRemove:  false,
				},
			},
			expectedHasVar: true,
		},
		{
			name: "Headers with multiple dynamic variables",
			headers: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Custom-Header",
					Value:     "$host",
					IsRemove:  false,
				},
				{
					Direction: HeaderDirection_ZoraxyToDownstream,
					Key:       "X-Another-Header",
					Value:     "$remote_addr",
					IsRemove:  false,
				},
			},
			expectedHasVar: true,
		},
		{
			name: "Headers with dollar sign in middle of value",
			headers: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Price",
					Value:     "Price is $100",
					IsRemove:  false,
				},
			},
			expectedHasVar: true,
		},
		{
			name: "Mixed static and dynamic headers",
			headers: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Static",
					Value:     "static value",
					IsRemove:  false,
				},
				{
					Direction: HeaderDirection_ZoraxyToDownstream,
					Key:       "X-Dynamic",
					Value:     "$uri",
					IsRemove:  false,
				},
			},
			expectedHasVar: true,
		},
		{
			name: "Headers marked for removal with dynamic variable",
			headers: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Remove",
					Value:     "$host",
					IsRemove:  true,
				},
			},
			expectedHasVar: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasVar := CustomHeadersIncludeDynamicVariables(tt.headers)
			assert.Equal(t, tt.expectedHasVar, hasVar)
		})
	}
}

func TestPopulateRequestHeaderVariables(t *testing.T) {
	tests := []struct {
		name               string
		url                string
		remoteAddr         string
		host               string
		method             string
		userDefinedHeaders []*UserDefinedHeader
		expectedResults    []*UserDefinedHeader
	}{
		{
			name:       "Single variable substitution",
			url:        "https://example.com/test?foo=bar",
			remoteAddr: "192.168.1.1:12345",
			host:       "example.com",
			method:     "GET",
			userDefinedHeaders: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Forwarded-Host",
					Value:     "$host",
				},
			},
			expectedResults: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Forwarded-Host",
					Value:     "example.com",
				},
			},
		},
		{
			name:       "Multiple variables in one header",
			url:        "https://api.example.com/users/123",
			remoteAddr: "10.0.0.5:54321",
			host:       "api.example.com",
			method:     "POST",
			userDefinedHeaders: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToDownstream,
					Key:       "X-Request-Info",
					Value:     "Method: $request_method, Host: $host, URI: $uri",
				},
			},
			expectedResults: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToDownstream,
					Key:       "X-Request-Info",
					Value:     "Method: POST, Host: api.example.com, URI: /users/123",
				},
			},
		},
		{
			name:       "Multiple headers with different variables",
			url:        "https://example.com/test?foo=bar",
			remoteAddr: "192.168.1.1:12345",
			host:       "example.com",
			method:     "GET",
			userDefinedHeaders: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Forwarded-Host",
					Value:     "$host",
				},
				{
					Direction: HeaderDirection_ZoraxyToDownstream,
					Key:       "X-Client-IP",
					Value:     "$remote_addr",
				},
				{
					Direction: HeaderDirection_ZoraxyToDownstream,
					Key:       "X-Custom-Header",
					Value:     "$request_uri",
				},
			},
			expectedResults: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Forwarded-Host",
					Value:     "example.com",
				},
				{
					Direction: HeaderDirection_ZoraxyToDownstream,
					Key:       "X-Client-IP",
					Value:     "192.168.1.1:12345",
				},
				{
					Direction: HeaderDirection_ZoraxyToDownstream,
					Key:       "X-Custom-Header",
					Value:     "https://example.com/test?foo=bar",
				},
			},
		},
		{
			name:       "No dynamic variables (early exit)",
			url:        "https://example.com/test",
			remoteAddr: "192.168.1.1:12345",
			host:       "example.com",
			method:     "GET",
			userDefinedHeaders: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Static-Header",
					Value:     "static-value",
				},
			},
			expectedResults: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Static-Header",
					Value:     "static-value",
				},
			},
		},
		{
			name:               "Empty headers slice",
			url:                "https://example.com/test",
			remoteAddr:         "192.168.1.1:12345",
			host:               "example.com",
			method:             "GET",
			userDefinedHeaders: []*UserDefinedHeader{},
			expectedResults:    []*UserDefinedHeader{},
		},
		{
			name:               "Nil headers slice",
			url:                "https://example.com/test",
			remoteAddr:         "192.168.1.1:12345",
			host:               "example.com",
			method:             "GET",
			userDefinedHeaders: nil,
			expectedResults:    nil,
		},
		{
			name:       "Header with IsRemove flag",
			url:        "https://example.com/test",
			remoteAddr: "192.168.1.1:12345",
			host:       "example.com",
			method:     "GET",
			userDefinedHeaders: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Remove-Header",
					Value:     "$host",
					IsRemove:  true,
				},
			},
			expectedResults: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Remove-Header",
					Value:     "example.com",
					IsRemove:  true,
				},
			},
		},
		{
			name:       "Variables with query string",
			url:        "https://example.com/search?q=test&page=1",
			remoteAddr: "192.168.1.1:12345",
			host:       "example.com",
			method:     "GET",
			userDefinedHeaders: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Query",
					Value:     "$args",
				},
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Query-String",
					Value:     "$query_string",
				},
			},
			expectedResults: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Query",
					Value:     "q=test&page=1",
				},
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Query-String",
					Value:     "q=test&page=1",
				},
			},
		},
		{
			name:       "All available variables",
			url:        "https://example.com/api/test?foo=bar",
			remoteAddr: "192.168.1.1:12345",
			host:       "example.com",
			method:     "POST",
			userDefinedHeaders: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-All-Vars",
					Value:     "$host|$remote_addr|$uri|$args|$scheme|$request_method",
				},
			},
			expectedResults: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-All-Vars",
					Value:     "example.com|192.168.1.1:12345|/api/test|foo=bar|https|POST",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			req.Host = tt.host
			req.RemoteAddr = tt.remoteAddr

			resultHeaders := PopulateRequestHeaderVariables(req, tt.userDefinedHeaders)

			assert.Equal(t, len(tt.expectedResults), len(resultHeaders))

			for i, expected := range tt.expectedResults {
				assert.Equal(t, expected.Direction, resultHeaders[i].Direction)
				assert.Equal(t, expected.Key, resultHeaders[i].Key)
				assert.Equal(t, expected.Value, resultHeaders[i].Value)
				assert.Equal(t, expected.IsRemove, resultHeaders[i].IsRemove)
			}
		})
	}
}

func TestPopulateRequestHeaderVariables_DeepCopy(t *testing.T) {
	// Test that PopulateRequestHeaderVariables creates deep copies and doesn't modify originals
	req := httptest.NewRequest("GET", "https://example.com/test", nil)
	req.Host = "example.com"
	req.RemoteAddr = "192.168.1.1:12345"

	originalHeaders := []*UserDefinedHeader{
		{
			Direction: HeaderDirection_ZoraxyToUpstream,
			Key:       "X-Host",
			Value:     "$host",
		},
	}

	// Store original value
	originalValue := originalHeaders[0].Value

	// Populate variables
	resultHeaders := PopulateRequestHeaderVariables(req, originalHeaders)

	// Original should be unchanged
	assert.Equal(t, originalValue, originalHeaders[0].Value)

	// Result should have substituted value
	assert.Equal(t, "example.com", resultHeaders[0].Value)

	// Modifying result should not affect original
	resultHeaders[0].Value = "modified"
	assert.Equal(t, originalValue, originalHeaders[0].Value)
}
