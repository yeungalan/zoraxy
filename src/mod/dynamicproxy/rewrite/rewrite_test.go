package rewrite

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"imuslab.com/zoraxy/mod/dynamicproxy/permissionpolicy"
)

func TestSplitUpDownStreamHeaders_EmptyOptions(t *testing.T) {
	tests := []struct {
		name           string
		rewriteOptions *HeaderRewriteOptions
	}{
		{
			name: "Completely empty options",
			rewriteOptions: &HeaderRewriteOptions{
				UserDefinedHeaders:           []*UserDefinedHeader{},
				HSTSMaxAge:                   0,
				HSTSIncludeSubdomains:        false,
				EnablePermissionPolicyHeader: false,
				PermissionPolicy:             nil,
			},
		},
		{
			name: "Nil user defined headers",
			rewriteOptions: &HeaderRewriteOptions{
				UserDefinedHeaders:           nil,
				HSTSMaxAge:                   0,
				HSTSIncludeSubdomains:        false,
				EnablePermissionPolicyHeader: false,
				PermissionPolicy:             nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream, downstream := SplitUpDownStreamHeaders(tt.rewriteOptions)

			assert.Empty(t, upstream, "Upstream headers should be empty")
			assert.Empty(t, downstream, "Downstream headers should be empty")
		})
	}
}

func TestSplitUpDownStreamHeaders_OnlyUpstreamHeaders(t *testing.T) {
	tests := []struct {
		name               string
		userDefinedHeaders []*UserDefinedHeader
		expectedUpstream   [][]string
	}{
		{
			name: "Single upstream header",
			userDefinedHeaders: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Forwarded-For",
					Value:     "192.168.1.1",
					IsRemove:  false,
				},
			},
			expectedUpstream: [][]string{
				{"X-Forwarded-For", "192.168.1.1"},
			},
		},
		{
			name: "Multiple upstream headers",
			userDefinedHeaders: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Forwarded-For",
					Value:     "192.168.1.1",
					IsRemove:  false,
				},
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Real-IP",
					Value:     "10.0.0.1",
					IsRemove:  false,
				},
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Custom-Header",
					Value:     "custom-value",
					IsRemove:  false,
				},
			},
			expectedUpstream: [][]string{
				{"X-Forwarded-For", "192.168.1.1"},
				{"X-Real-IP", "10.0.0.1"},
				{"X-Custom-Header", "custom-value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &HeaderRewriteOptions{
				UserDefinedHeaders:           tt.userDefinedHeaders,
				HSTSMaxAge:                   0,
				HSTSIncludeSubdomains:        false,
				EnablePermissionPolicyHeader: false,
				PermissionPolicy:             nil,
			}

			upstream, downstream := SplitUpDownStreamHeaders(options)

			assert.Equal(t, tt.expectedUpstream, upstream)
			assert.Empty(t, downstream, "Downstream headers should be empty")
		})
	}
}

func TestSplitUpDownStreamHeaders_OnlyDownstreamHeaders(t *testing.T) {
	tests := []struct {
		name               string
		userDefinedHeaders []*UserDefinedHeader
		expectedDownstream [][]string
	}{
		{
			name: "Single downstream header",
			userDefinedHeaders: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToDownstream,
					Key:       "X-Frame-Options",
					Value:     "DENY",
					IsRemove:  false,
				},
			},
			expectedDownstream: [][]string{
				{"X-Frame-Options", "DENY"},
			},
		},
		{
			name: "Multiple downstream headers",
			userDefinedHeaders: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToDownstream,
					Key:       "X-Frame-Options",
					Value:     "DENY",
					IsRemove:  false,
				},
				{
					Direction: HeaderDirection_ZoraxyToDownstream,
					Key:       "X-Content-Type-Options",
					Value:     "nosniff",
					IsRemove:  false,
				},
				{
					Direction: HeaderDirection_ZoraxyToDownstream,
					Key:       "X-XSS-Protection",
					Value:     "1; mode=block",
					IsRemove:  false,
				},
			},
			expectedDownstream: [][]string{
				{"X-Frame-Options", "DENY"},
				{"X-Content-Type-Options", "nosniff"},
				{"X-XSS-Protection", "1; mode=block"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &HeaderRewriteOptions{
				UserDefinedHeaders:           tt.userDefinedHeaders,
				HSTSMaxAge:                   0,
				HSTSIncludeSubdomains:        false,
				EnablePermissionPolicyHeader: false,
				PermissionPolicy:             nil,
			}

			upstream, downstream := SplitUpDownStreamHeaders(options)

			assert.Empty(t, upstream, "Upstream headers should be empty")
			assert.Equal(t, tt.expectedDownstream, downstream)
		})
	}
}

func TestSplitUpDownStreamHeaders_MixedHeaders(t *testing.T) {
	tests := []struct {
		name               string
		userDefinedHeaders []*UserDefinedHeader
		expectedUpstream   [][]string
		expectedDownstream [][]string
	}{
		{
			name: "Mix of upstream and downstream headers",
			userDefinedHeaders: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Forwarded-For",
					Value:     "192.168.1.1",
					IsRemove:  false,
				},
				{
					Direction: HeaderDirection_ZoraxyToDownstream,
					Key:       "X-Frame-Options",
					Value:     "DENY",
					IsRemove:  false,
				},
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Real-IP",
					Value:     "10.0.0.1",
					IsRemove:  false,
				},
				{
					Direction: HeaderDirection_ZoraxyToDownstream,
					Key:       "X-Content-Type-Options",
					Value:     "nosniff",
					IsRemove:  false,
				},
			},
			expectedUpstream: [][]string{
				{"X-Forwarded-For", "192.168.1.1"},
				{"X-Real-IP", "10.0.0.1"},
			},
			expectedDownstream: [][]string{
				{"X-Frame-Options", "DENY"},
				{"X-Content-Type-Options", "nosniff"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &HeaderRewriteOptions{
				UserDefinedHeaders:           tt.userDefinedHeaders,
				HSTSMaxAge:                   0,
				HSTSIncludeSubdomains:        false,
				EnablePermissionPolicyHeader: false,
				PermissionPolicy:             nil,
			}

			upstream, downstream := SplitUpDownStreamHeaders(options)

			assert.Equal(t, tt.expectedUpstream, upstream)
			assert.Equal(t, tt.expectedDownstream, downstream)
		})
	}
}

func TestSplitUpDownStreamHeaders_RemoveHeaders(t *testing.T) {
	tests := []struct {
		name               string
		userDefinedHeaders []*UserDefinedHeader
		expectedUpstream   [][]string
		expectedDownstream [][]string
	}{
		{
			name: "Upstream header with remove flag",
			userDefinedHeaders: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Remove-Me",
					Value:     "should-be-empty",
					IsRemove:  true,
				},
			},
			expectedUpstream: [][]string{
				{"X-Remove-Me", ""},
			},
			expectedDownstream: [][]string{},
		},
		{
			name: "Downstream header with remove flag",
			userDefinedHeaders: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToDownstream,
					Key:       "X-Remove-Me",
					Value:     "should-be-empty",
					IsRemove:  true,
				},
			},
			expectedUpstream: [][]string{},
			expectedDownstream: [][]string{
				{"X-Remove-Me", ""},
			},
		},
		{
			name: "Mixed headers with and without remove flag",
			userDefinedHeaders: []*UserDefinedHeader{
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Keep-Me",
					Value:     "keep-value",
					IsRemove:  false,
				},
				{
					Direction: HeaderDirection_ZoraxyToUpstream,
					Key:       "X-Remove-Me",
					Value:     "remove-value",
					IsRemove:  true,
				},
				{
					Direction: HeaderDirection_ZoraxyToDownstream,
					Key:       "X-Keep-Too",
					Value:     "keep-value-2",
					IsRemove:  false,
				},
			},
			expectedUpstream: [][]string{
				{"X-Keep-Me", "keep-value"},
				{"X-Remove-Me", ""},
			},
			expectedDownstream: [][]string{
				{"X-Keep-Too", "keep-value-2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &HeaderRewriteOptions{
				UserDefinedHeaders:           tt.userDefinedHeaders,
				HSTSMaxAge:                   0,
				HSTSIncludeSubdomains:        false,
				EnablePermissionPolicyHeader: false,
				PermissionPolicy:             nil,
			}

			upstream, downstream := SplitUpDownStreamHeaders(options)

			assert.Equal(t, tt.expectedUpstream, upstream)
			assert.Equal(t, tt.expectedDownstream, downstream)
		})
	}
}

func TestSplitUpDownStreamHeaders_HSTS(t *testing.T) {
	tests := []struct {
		name                  string
		hstsMaxAge            int64
		hstsIncludeSubdomains bool
		expectedHeader        []string
	}{
		{
			name:                  "HSTS without subdomains",
			hstsMaxAge:            31536000,
			hstsIncludeSubdomains: false,
			expectedHeader:        []string{"Strict-Transport-Security", "max-age=31536000"},
		},
		{
			name:                  "HSTS with subdomains",
			hstsMaxAge:            31536000,
			hstsIncludeSubdomains: true,
			expectedHeader:        []string{"Strict-Transport-Security", "max-age=31536000; includeSubdomains"},
		},
		{
			name:                  "HSTS with different max age",
			hstsMaxAge:            63072000,
			hstsIncludeSubdomains: false,
			expectedHeader:        []string{"Strict-Transport-Security", "max-age=63072000"},
		},
		{
			name:                  "HSTS with short max age and subdomains",
			hstsMaxAge:            86400,
			hstsIncludeSubdomains: true,
			expectedHeader:        []string{"Strict-Transport-Security", "max-age=86400; includeSubdomains"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &HeaderRewriteOptions{
				UserDefinedHeaders:           []*UserDefinedHeader{},
				HSTSMaxAge:                   tt.hstsMaxAge,
				HSTSIncludeSubdomains:        tt.hstsIncludeSubdomains,
				EnablePermissionPolicyHeader: false,
				PermissionPolicy:             nil,
			}

			upstream, downstream := SplitUpDownStreamHeaders(options)

			assert.Empty(t, upstream)
			assert.Equal(t, 1, len(downstream))
			assert.Equal(t, tt.expectedHeader, downstream[0])
		})
	}
}

func TestSplitUpDownStreamHeaders_HSTS_Disabled(t *testing.T) {
	// Test that HSTS is not added when max age is 0
	options := &HeaderRewriteOptions{
		UserDefinedHeaders:           []*UserDefinedHeader{},
		HSTSMaxAge:                   0,
		HSTSIncludeSubdomains:        true, // This should be ignored
		EnablePermissionPolicyHeader: false,
		PermissionPolicy:             nil,
	}

	upstream, downstream := SplitUpDownStreamHeaders(options)

	assert.Empty(t, upstream)
	assert.Empty(t, downstream)
}

func TestSplitUpDownStreamHeaders_PermissionPolicy(t *testing.T) {
	tests := []struct {
		name               string
		enablePolicy       bool
		customPolicy       *permissionpolicy.PermissionsPolicy
		validateDownstream func(t *testing.T, downstream [][]string)
	}{
		{
			name:         "Permission policy enabled with default",
			enablePolicy: true,
			customPolicy: nil,
			validateDownstream: func(t *testing.T, downstream [][]string) {
				assert.Equal(t, 1, len(downstream))
				assert.Equal(t, "Permissions-Policy", downstream[0][0])
				assert.NotEmpty(t, downstream[0][1])
				// Verify it contains some expected default policies
				assert.Contains(t, downstream[0][1], "accelerometer=*")
				assert.Contains(t, downstream[0][1], "camera=*")
			},
		},
		{
			name:         "Permission policy enabled with custom policy",
			enablePolicy: true,
			customPolicy: &permissionpolicy.PermissionsPolicy{
				Camera:      []string{"self"},
				Microphone:  []string{"self"},
				Geolocation: []string{},
			},
			validateDownstream: func(t *testing.T, downstream [][]string) {
				assert.Equal(t, 1, len(downstream))
				assert.Equal(t, "Permissions-Policy", downstream[0][0])
				assert.NotEmpty(t, downstream[0][1])
			},
		},
		{
			name:         "Permission policy disabled",
			enablePolicy: false,
			customPolicy: nil,
			validateDownstream: func(t *testing.T, downstream [][]string) {
				assert.Empty(t, downstream)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &HeaderRewriteOptions{
				UserDefinedHeaders:           []*UserDefinedHeader{},
				HSTSMaxAge:                   0,
				HSTSIncludeSubdomains:        false,
				EnablePermissionPolicyHeader: tt.enablePolicy,
				PermissionPolicy:             tt.customPolicy,
			}

			upstream, downstream := SplitUpDownStreamHeaders(options)

			assert.Empty(t, upstream)
			tt.validateDownstream(t, downstream)
		})
	}
}

func TestSplitUpDownStreamHeaders_CombinedFeatures(t *testing.T) {
	tests := []struct {
		name               string
		options            *HeaderRewriteOptions
		expectedUpstream   int
		expectedDownstream int
		validateHeaders    func(t *testing.T, upstream [][]string, downstream [][]string)
	}{
		{
			name: "Custom headers + HSTS",
			options: &HeaderRewriteOptions{
				UserDefinedHeaders: []*UserDefinedHeader{
					{
						Direction: HeaderDirection_ZoraxyToUpstream,
						Key:       "X-Forwarded-For",
						Value:     "192.168.1.1",
						IsRemove:  false,
					},
					{
						Direction: HeaderDirection_ZoraxyToDownstream,
						Key:       "X-Frame-Options",
						Value:     "DENY",
						IsRemove:  false,
					},
				},
				HSTSMaxAge:                   31536000,
				HSTSIncludeSubdomains:        true,
				EnablePermissionPolicyHeader: false,
				PermissionPolicy:             nil,
			},
			expectedUpstream:   1,
			expectedDownstream: 2,
			validateHeaders: func(t *testing.T, upstream [][]string, downstream [][]string) {
				// Verify upstream
				assert.Equal(t, "X-Forwarded-For", upstream[0][0])
				assert.Equal(t, "192.168.1.1", upstream[0][1])

				// Verify downstream - should have custom header and HSTS
				hasFrameOptions := false
				hasHSTS := false
				for _, header := range downstream {
					if header[0] == "X-Frame-Options" {
						hasFrameOptions = true
						assert.Equal(t, "DENY", header[1])
					}
					if header[0] == "Strict-Transport-Security" {
						hasHSTS = true
						assert.Contains(t, header[1], "max-age=31536000")
					}
				}
				assert.True(t, hasFrameOptions, "Should have X-Frame-Options")
				assert.True(t, hasHSTS, "Should have HSTS header")
			},
		},
		{
			name: "Custom headers + Permission Policy",
			options: &HeaderRewriteOptions{
				UserDefinedHeaders: []*UserDefinedHeader{
					{
						Direction: HeaderDirection_ZoraxyToDownstream,
						Key:       "X-Content-Type-Options",
						Value:     "nosniff",
						IsRemove:  false,
					},
				},
				HSTSMaxAge:                   0,
				HSTSIncludeSubdomains:        false,
				EnablePermissionPolicyHeader: true,
				PermissionPolicy:             nil,
			},
			expectedUpstream:   0,
			expectedDownstream: 2,
			validateHeaders: func(t *testing.T, upstream [][]string, downstream [][]string) {
				assert.Empty(t, upstream)

				hasContentType := false
				hasPermissionPolicy := false
				for _, header := range downstream {
					if header[0] == "X-Content-Type-Options" {
						hasContentType = true
						assert.Equal(t, "nosniff", header[1])
					}
					if header[0] == "Permissions-Policy" {
						hasPermissionPolicy = true
					}
				}
				assert.True(t, hasContentType, "Should have X-Content-Type-Options")
				assert.True(t, hasPermissionPolicy, "Should have Permissions-Policy")
			},
		},
		{
			name: "All features combined",
			options: &HeaderRewriteOptions{
				UserDefinedHeaders: []*UserDefinedHeader{
					{
						Direction: HeaderDirection_ZoraxyToUpstream,
						Key:       "X-Forwarded-For",
						Value:     "192.168.1.1",
						IsRemove:  false,
					},
					{
						Direction: HeaderDirection_ZoraxyToDownstream,
						Key:       "X-Frame-Options",
						Value:     "DENY",
						IsRemove:  false,
					},
					{
						Direction: HeaderDirection_ZoraxyToDownstream,
						Key:       "X-Content-Type-Options",
						Value:     "nosniff",
						IsRemove:  false,
					},
				},
				HSTSMaxAge:                   31536000,
				HSTSIncludeSubdomains:        true,
				EnablePermissionPolicyHeader: true,
				PermissionPolicy:             nil,
			},
			expectedUpstream:   1,
			expectedDownstream: 4, // 2 custom + HSTS + Permission Policy
			validateHeaders: func(t *testing.T, upstream [][]string, downstream [][]string) {
				assert.Equal(t, "X-Forwarded-For", upstream[0][0])

				hasFrameOptions := false
				hasContentType := false
				hasHSTS := false
				hasPermissionPolicy := false
				for _, header := range downstream {
					switch header[0] {
					case "X-Frame-Options":
						hasFrameOptions = true
					case "X-Content-Type-Options":
						hasContentType = true
					case "Strict-Transport-Security":
						hasHSTS = true
					case "Permissions-Policy":
						hasPermissionPolicy = true
					}
				}
				assert.True(t, hasFrameOptions, "Should have X-Frame-Options")
				assert.True(t, hasContentType, "Should have X-Content-Type-Options")
				assert.True(t, hasHSTS, "Should have HSTS")
				assert.True(t, hasPermissionPolicy, "Should have Permissions-Policy")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream, downstream := SplitUpDownStreamHeaders(tt.options)

			assert.Equal(t, tt.expectedUpstream, len(upstream))
			assert.Equal(t, tt.expectedDownstream, len(downstream))

			if tt.validateHeaders != nil {
				tt.validateHeaders(t, upstream, downstream)
			}
		})
	}
}

func TestSplitUpDownStreamHeaders_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		options  *HeaderRewriteOptions
		validate func(t *testing.T, upstream [][]string, downstream [][]string)
	}{
		{
			name: "Empty header key",
			options: &HeaderRewriteOptions{
				UserDefinedHeaders: []*UserDefinedHeader{
					{
						Direction: HeaderDirection_ZoraxyToUpstream,
						Key:       "",
						Value:     "some-value",
						IsRemove:  false,
					},
				},
			},
			validate: func(t *testing.T, upstream [][]string, downstream [][]string) {
				assert.Equal(t, 1, len(upstream))
				assert.Equal(t, "", upstream[0][0])
			},
		},
		{
			name: "Empty header value",
			options: &HeaderRewriteOptions{
				UserDefinedHeaders: []*UserDefinedHeader{
					{
						Direction: HeaderDirection_ZoraxyToDownstream,
						Key:       "X-Empty-Value",
						Value:     "",
						IsRemove:  false,
					},
				},
			},
			validate: func(t *testing.T, upstream [][]string, downstream [][]string) {
				assert.Equal(t, 1, len(downstream))
				assert.Equal(t, "X-Empty-Value", downstream[0][0])
				assert.Equal(t, "", downstream[0][1])
			},
		},
		{
			name: "Very large HSTS max age",
			options: &HeaderRewriteOptions{
				UserDefinedHeaders:           []*UserDefinedHeader{},
				HSTSMaxAge:                   999999999999,
				HSTSIncludeSubdomains:        false,
				EnablePermissionPolicyHeader: false,
			},
			validate: func(t *testing.T, upstream [][]string, downstream [][]string) {
				assert.Equal(t, 1, len(downstream))
				assert.Equal(t, "Strict-Transport-Security", downstream[0][0])
				assert.Contains(t, downstream[0][1], "max-age=999999999999")
			},
		},
		{
			name: "Many custom headers",
			options: &HeaderRewriteOptions{
				UserDefinedHeaders: func() []*UserDefinedHeader {
					headers := make([]*UserDefinedHeader, 20)
					for i := 0; i < 20; i++ {
						direction := HeaderDirection_ZoraxyToUpstream
						if i%2 == 0 {
							direction = HeaderDirection_ZoraxyToDownstream
						}
						headers[i] = &UserDefinedHeader{
							Direction: direction,
							Key:       "X-Header-" + string(rune('A'+i)),
							Value:     "value-" + string(rune('0'+i)),
							IsRemove:  false,
						}
					}
					return headers
				}(),
			},
			validate: func(t *testing.T, upstream [][]string, downstream [][]string) {
				assert.Equal(t, 10, len(upstream))
				assert.Equal(t, 10, len(downstream))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream, downstream := SplitUpDownStreamHeaders(tt.options)
			tt.validate(t, upstream, downstream)
		})
	}
}

func TestSplitUpDownStreamHeaders_OrderPreservation(t *testing.T) {
	// Test that the order of headers is preserved
	options := &HeaderRewriteOptions{
		UserDefinedHeaders: []*UserDefinedHeader{
			{Direction: HeaderDirection_ZoraxyToUpstream, Key: "Header-1", Value: "Value-1"},
			{Direction: HeaderDirection_ZoraxyToUpstream, Key: "Header-2", Value: "Value-2"},
			{Direction: HeaderDirection_ZoraxyToUpstream, Key: "Header-3", Value: "Value-3"},
			{Direction: HeaderDirection_ZoraxyToDownstream, Key: "Header-A", Value: "Value-A"},
			{Direction: HeaderDirection_ZoraxyToDownstream, Key: "Header-B", Value: "Value-B"},
			{Direction: HeaderDirection_ZoraxyToDownstream, Key: "Header-C", Value: "Value-C"},
		},
	}

	upstream, downstream := SplitUpDownStreamHeaders(options)

	// Verify upstream order
	assert.Equal(t, "Header-1", upstream[0][0])
	assert.Equal(t, "Header-2", upstream[1][0])
	assert.Equal(t, "Header-3", upstream[2][0])

	// Verify downstream order
	assert.Equal(t, "Header-A", downstream[0][0])
	assert.Equal(t, "Header-B", downstream[1][0])
	assert.Equal(t, "Header-C", downstream[2][0])
}
