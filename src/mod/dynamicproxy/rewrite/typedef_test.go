package rewrite

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserDefinedHeader_GetDirection(t *testing.T) {
	tests := []struct {
		name              string
		header            *UserDefinedHeader
		expectedDirection HeaderDirection
	}{
		{
			name: "Upstream direction",
			header: &UserDefinedHeader{
				Direction: HeaderDirection_ZoraxyToUpstream,
				Key:       "X-Custom-Header",
				Value:     "test-value",
				IsRemove:  false,
			},
			expectedDirection: HeaderDirection_ZoraxyToUpstream,
		},
		{
			name: "Downstream direction",
			header: &UserDefinedHeader{
				Direction: HeaderDirection_ZoraxyToDownstream,
				Key:       "X-Custom-Header",
				Value:     "test-value",
				IsRemove:  false,
			},
			expectedDirection: HeaderDirection_ZoraxyToDownstream,
		},
		{
			name: "Direction with remove flag",
			header: &UserDefinedHeader{
				Direction: HeaderDirection_ZoraxyToUpstream,
				Key:       "X-Remove-Header",
				Value:     "",
				IsRemove:  true,
			},
			expectedDirection: HeaderDirection_ZoraxyToUpstream,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			direction := tt.header.GetDirection()
			assert.Equal(t, tt.expectedDirection, direction)
		})
	}
}

func TestUserDefinedHeader_Copy(t *testing.T) {
	tests := []struct {
		name   string
		header *UserDefinedHeader
	}{
		{
			name: "Copy basic header",
			header: &UserDefinedHeader{
				Direction: HeaderDirection_ZoraxyToUpstream,
				Key:       "X-Custom-Header",
				Value:     "test-value",
				IsRemove:  false,
			},
		},
		{
			name: "Copy header with remove flag",
			header: &UserDefinedHeader{
				Direction: HeaderDirection_ZoraxyToDownstream,
				Key:       "X-Remove-Header",
				Value:     "",
				IsRemove:  true,
			},
		},
		{
			name: "Copy header with dynamic variable",
			header: &UserDefinedHeader{
				Direction: HeaderDirection_ZoraxyToUpstream,
				Key:       "X-Forwarded-Host",
				Value:     "$host",
				IsRemove:  false,
			},
		},
		{
			name: "Copy header with empty value",
			header: &UserDefinedHeader{
				Direction: HeaderDirection_ZoraxyToDownstream,
				Key:       "X-Empty",
				Value:     "",
				IsRemove:  false,
			},
		},
		{
			name: "Copy header with special characters",
			header: &UserDefinedHeader{
				Direction: HeaderDirection_ZoraxyToUpstream,
				Key:       "X-Special-Chars",
				Value:     "value with spaces & special chars: @#$%",
				IsRemove:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy
			copied := tt.header.Copy()

			// Verify all fields are equal
			assert.Equal(t, tt.header.Direction, copied.Direction)
			assert.Equal(t, tt.header.Key, copied.Key)
			assert.Equal(t, tt.header.Value, copied.Value)
			assert.Equal(t, tt.header.IsRemove, copied.IsRemove)

			// Verify it's a different instance (deep copy)
			assert.NotSame(t, tt.header, copied)

			// Modify the copy and ensure original is unchanged
			copied.Key = "Modified-Key"
			copied.Value = "Modified-Value"
			copied.IsRemove = !copied.IsRemove

			assert.NotEqual(t, tt.header.Key, copied.Key, "Original should not be affected by copy modification")
			assert.NotEqual(t, tt.header.Value, copied.Value, "Original should not be affected by copy modification")
		})
	}
}

func TestUserDefinedHeader_Copy_Independence(t *testing.T) {
	// Test that modifying a copy doesn't affect the original
	original := &UserDefinedHeader{
		Direction: HeaderDirection_ZoraxyToUpstream,
		Key:       "X-Original",
		Value:     "original-value",
		IsRemove:  false,
	}

	copy1 := original.Copy()
	copy2 := original.Copy()

	// Modify first copy
	copy1.Key = "X-Copy1"
	copy1.Value = "copy1-value"

	// Modify second copy
	copy2.Key = "X-Copy2"
	copy2.Value = "copy2-value"

	// Original should remain unchanged
	assert.Equal(t, "X-Original", original.Key)
	assert.Equal(t, "original-value", original.Value)

	// Copies should be independent of each other
	assert.NotEqual(t, copy1.Key, copy2.Key)
	assert.NotEqual(t, copy1.Value, copy2.Value)
}

func TestHeaderDirection_Constants(t *testing.T) {
	// Test that the header direction constants have expected values
	assert.Equal(t, HeaderDirection(0), HeaderDirection_ZoraxyToUpstream)
	assert.Equal(t, HeaderDirection(1), HeaderDirection_ZoraxyToDownstream)
}

func TestHeaderRewriteOptions_Structure(t *testing.T) {
	// Test that HeaderRewriteOptions can be created with various configurations
	tests := []struct {
		name    string
		options *HeaderRewriteOptions
	}{
		{
			name: "Empty options",
			options: &HeaderRewriteOptions{
				UserDefinedHeaders:           nil,
				HSTSMaxAge:                   0,
				HSTSIncludeSubdomains:        false,
				EnablePermissionPolicyHeader: false,
				PermissionPolicy:             nil,
			},
		},
		{
			name: "Options with custom headers",
			options: &HeaderRewriteOptions{
				UserDefinedHeaders: []*UserDefinedHeader{
					{
						Direction: HeaderDirection_ZoraxyToUpstream,
						Key:       "X-Custom",
						Value:     "value",
						IsRemove:  false,
					},
				},
				HSTSMaxAge:                   0,
				HSTSIncludeSubdomains:        false,
				EnablePermissionPolicyHeader: false,
				PermissionPolicy:             nil,
			},
		},
		{
			name: "Options with HSTS",
			options: &HeaderRewriteOptions{
				UserDefinedHeaders:           nil,
				HSTSMaxAge:                   31536000,
				HSTSIncludeSubdomains:        true,
				EnablePermissionPolicyHeader: false,
				PermissionPolicy:             nil,
			},
		},
		{
			name: "Options with permission policy",
			options: &HeaderRewriteOptions{
				UserDefinedHeaders:           nil,
				HSTSMaxAge:                   0,
				HSTSIncludeSubdomains:        false,
				EnablePermissionPolicyHeader: true,
				PermissionPolicy:             nil,
			},
		},
		{
			name: "Options with all features",
			options: &HeaderRewriteOptions{
				UserDefinedHeaders: []*UserDefinedHeader{
					{
						Direction: HeaderDirection_ZoraxyToUpstream,
						Key:       "X-Custom",
						Value:     "value",
						IsRemove:  false,
					},
				},
				HSTSMaxAge:                   31536000,
				HSTSIncludeSubdomains:        true,
				EnablePermissionPolicyHeader: true,
				PermissionPolicy:             nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the structure can be created
			assert.NotNil(t, tt.options)
		})
	}
}
