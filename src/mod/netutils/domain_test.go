package netutils_test

import (
	"strings"
	"testing"

	"imuslab.com/zoraxy/mod/netutils"

	"github.com/stretchr/testify/assert"
)

// Test NormalizeDomain
func TestNormalizeDomain(t *testing.T) {
	tests := []struct {
		name      string
		domain    string
		expected  string
		expectErr bool
	}{
		// Valid domains
		{"simple domain", "example.com", "example.com", false},
		{"uppercase domain", "EXAMPLE.COM", "example.com", false},
		{"mixed case domain", "Example.Com", "example.com", false},
		{"subdomain", "sub.example.com", "sub.example.com", false},
		{"deep subdomain", "a.b.c.d.example.com", "a.b.c.d.example.com", false},
		{"with spaces", "  example.com  ", "example.com", false},
		{"FQDN with trailing dot", "example.com.", "example.com", false},
		{"wildcard subdomain", "*.example.com", "*.example.com", false},
		{"domain with hyphen", "my-domain.com", "my-domain.com", false},
		{"domain with multiple hyphens", "my-cool-domain.com", "my-cool-domain.com", false},
		{"numeric subdomain", "123.example.com", "123.example.com", false},
		{"alphanumeric", "abc123.example.com", "abc123.example.com", false},
		{"single char label", "a.b.c", "a.b.c", false},
		{"63 char label", "a12345678901234567890123456789012345678901234567890123456789012.com", "a12345678901234567890123456789012345678901234567890123456789012.com", false},

		// Invalid domains
		{"empty string", "", "", true},
		{"only spaces", "   ", "", true},
		{"only dot", ".", "", true},
		{"double dot", "example..com", "", true},
		{"starts with dot", ".example.com", "", true},
		{"ends with dot after trim", "example.com..", "", true},
		{"label > 63 chars", "a123456789012345678901234567890123456789012345678901234567890123.com", "", true},
		{"label starts with hyphen", "-example.com", "", true},
		{"label ends with hyphen", "example-.com", "", true},
		{"hyphen at both ends", "-example-.com", "", true},
		{"special char underscore", "ex_ample.com", "", true},
		{"special char space", "ex ample.com", "", true},
		{"special char @", "ex@ample.com", "", true},
		{"special char #", "example#.com", "", true},
		{"numbers only", "123.456", "123.456", false}, // Actually valid
		{"just hyphen", "-", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := netutils.NormalizeDomain(tt.domain)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// Test NormalizeDomain edge cases
func TestNormalizeDomainEdgeCases(t *testing.T) {
	t.Run("Wildcard only in first label", func(t *testing.T) {
		result, err := netutils.NormalizeDomain("*.example.com")
		assert.NoError(t, err)
		assert.Equal(t, "*.example.com", result)
	})

	t.Run("Multiple wildcard attempts", func(t *testing.T) {
		_, err := netutils.NormalizeDomain("*.*.example.com")
		assert.Error(t, err) // Second asterisk is not in first position
	})

	t.Run("Wildcard in middle", func(t *testing.T) {
		_, err := netutils.NormalizeDomain("sub.*.example.com")
		assert.Error(t, err) // Wildcard not in first label
	})

	t.Run("Very long valid domain", func(t *testing.T) {
		// Create a domain with maximum allowed length (253 chars)
		// Use labels of length 63 + dots
		longDomain := "abcdefghijklmnopqrstuvwxyz0123456789012345678901234567890123456." + // 63
			"abcdefghijklmnopqrstuvwxyz0123456789012345678901234567890123456." + // 63
			"abcdefghijklmnopqrstuvwxyz0123456789012345678901234567890123456." + // 63
			"abcdefghijklmnopqrstuvwxyz0123456789012345678.com" // Remaining to reach near 253
		result, err := netutils.NormalizeDomain(longDomain)
		if len(longDomain) <= 253 {
			assert.NoError(t, err)
			assert.Equal(t, strings.ToLower(longDomain), result)
		} else {
			assert.Error(t, err)
		}
	})

	t.Run("Domain with trailing dot normalized", func(t *testing.T) {
		result, err := netutils.NormalizeDomain("example.com.")
		assert.NoError(t, err)
		assert.Equal(t, "example.com", result)
	})

	t.Run("Domain with multiple trailing dots", func(t *testing.T) {
		_, err := netutils.NormalizeDomain("example.com..")
		// After trimming one dot, we get "example.com."
		// After trimming suffix, we get "example.com"
		// But there's still an empty label from the second dot
		assert.Error(t, err) // Should error due to empty label
	})

	t.Run("Uppercase with wildcard", func(t *testing.T) {
		result, err := netutils.NormalizeDomain("*.EXAMPLE.COM")
		assert.NoError(t, err)
		assert.Equal(t, "*.example.com", result)
	})

	t.Run("Internationalized domain (unicode)", func(t *testing.T) {
		// Note: This function doesn't handle IDN/Punycode
		_, err := netutils.NormalizeDomain("münchen.de")
		assert.NoError(t, err) // Unicode is allowed by unicode.IsLetter
	})
}
