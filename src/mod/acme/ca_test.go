package acme

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadCAApiServerFromName(t *testing.T) {
	tests := []struct {
		name        string
		caName      string
		expectError bool
		expectURL   bool // We check if URL is non-empty rather than exact match
	}{
		{
			name:        "Let's Encrypt (common CA)",
			caName:      "Let's Encrypt",
			expectError: false,
			expectURL:   true,
		},
		{
			name:        "Buypass (exact match)",
			caName:      "Buypass",
			expectError: false,
			expectURL:   true,
		},
		{
			name:        "Buypass AS with suffix (should handle Buypass AS-983163327)",
			caName:      "Buypass AS-983163327",
			expectError: false,
			expectURL:   true,
		},
		{
			name:        "ZeroSSL (if supported)",
			caName:      "ZeroSSL",
			expectError: false,
			expectURL:   true,
		},
		{
			name:        "Unsupported CA",
			caName:      "NonExistentCA",
			expectError: true,
			expectURL:   false,
		},
		{
			name:        "Empty CA name",
			caName:      "",
			expectError: true,
			expectURL:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := loadCAApiServerFromName(tt.caName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not supported")
			} else {
				assert.NoError(t, err)
				if tt.expectURL {
					assert.NotEmpty(t, url)
					// URL should start with https://
					assert.Contains(t, url, "https://")
				}
			}
		})
	}
}

func TestIsSupportedCA(t *testing.T) {
	tests := []struct {
		name       string
		caName     string
		isSupported bool
	}{
		{
			name:       "Let's Encrypt is supported",
			caName:     "Let's Encrypt",
			isSupported: true,
		},
		{
			name:       "Buypass is supported",
			caName:     "Buypass",
			isSupported: true,
		},
		{
			name:       "Buypass with AS prefix is supported",
			caName:     "Buypass AS-983163327",
			isSupported: true,
		},
		{
			name:       "ZeroSSL is supported",
			caName:     "ZeroSSL",
			isSupported: true,
		},
		{
			name:       "Unknown CA is not supported",
			caName:     "UnknownCA",
			isSupported: false,
		},
		{
			name:       "Empty CA name is not supported",
			caName:     "",
			isSupported: false,
		},
		{
			name:       "Random string is not supported",
			caName:     "Random CA Provider",
			isSupported: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isSupported := IsSupportedCA(tt.caName)
			assert.Equal(t, tt.isSupported, isSupported)
		})
	}
}

func TestCaDefInitialization(t *testing.T) {
	// Test that the caDef is properly initialized from the embedded ca.json
	t.Run("Production CAs are loaded", func(t *testing.T) {
		assert.NotNil(t, caDef.Production)
		assert.NotEmpty(t, caDef.Production)
	})

	t.Run("Test CAs are loaded", func(t *testing.T) {
		assert.NotNil(t, caDef.Test)
		// Test may or may not have entries, so we just check it's initialized
	})

	t.Run("Let's Encrypt exists in production", func(t *testing.T) {
		url, exists := caDef.Production["Let's Encrypt"]
		assert.True(t, exists)
		assert.NotEmpty(t, url)
	})
}

func TestBuypassHandling(t *testing.T) {
	// Test the special handling of Buypass certificates with organization section
	tests := []struct {
		name     string
		caName   string
		expected string
	}{
		{
			name:     "Buypass without suffix",
			caName:   "Buypass",
			expected: "Buypass",
		},
		{
			name:     "Buypass with AS prefix and number",
			caName:   "Buypass AS-983163327",
			expected: "Buypass AS-983163327", // Input preserved but handled
		},
		{
			name:     "Buypass AS with different number",
			caName:   "Buypass AS-123456789",
			expected: "Buypass AS-123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The function should handle Buypass prefixes correctly
			url, err := loadCAApiServerFromName(tt.caName)
			assert.NoError(t, err)
			assert.NotEmpty(t, url)
			// All Buypass variants should return the same URL
			if tt.caName == "Buypass" || len(tt.caName) > 7 && tt.caName[:7] == "Buypass" {
				buypassURL, _ := loadCAApiServerFromName("Buypass")
				assert.Equal(t, buypassURL, url)
			}
		})
	}
}
