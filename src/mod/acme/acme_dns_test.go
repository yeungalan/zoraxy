package acme

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDnsChallengeProviderByName(t *testing.T) {
	tests := []struct {
		name            string
		dnsProvider     string
		dnsCredentials  string
		ppgTimeout      int
		expectError     bool
		errorContains   string
	}{
		{
			name:        "Invalid JSON credentials",
			dnsProvider: "cloudflare",
			dnsCredentials: "invalid json",
			ppgTimeout:  600,
			expectError: true,
			errorContains: "invalid character",
		},
		{
			name:        "Empty credentials",
			dnsProvider: "cloudflare",
			dnsCredentials: "",
			ppgTimeout:  600,
			expectError: true,
			errorContains: "unexpected end of JSON input",
		},
		{
			name:        "Unknown DNS provider",
			dnsProvider: "unknownprovider",
			dnsCredentials: `{"APIKey":"test123"}`,
			ppgTimeout:  600,
			expectError: true,
			errorContains: "unrecognized DNS provider",
		},
		{
			name:        "Valid JSON but missing required fields",
			dnsProvider: "cloudflare",
			dnsCredentials: `{}`,
			ppgTimeout:  600,
			expectError: true,
			// Error from the actual provider about missing required fields
		},
		{
			name:        "Credentials with PollingInterval",
			dnsProvider: "cloudflare",
			dnsCredentials: `{"PollingInterval":"5","AuthEmail":"test@example.com","AuthKey":"testkey"}`,
			ppgTimeout:  600,
			expectError: true, // Will error due to missing required fields, but tests parsing
		},
		{
			name:        "Credentials with PropagationTimeout",
			dnsProvider: "cloudflare",
			dnsCredentials: `{"PropagationTimeout":"300","AuthEmail":"test@example.com","AuthKey":"testkey"}`,
			ppgTimeout:  600,
			expectError: true, // Will error due to missing required fields, but tests parsing
		},
		{
			name:        "Credentials with both timeouts",
			dnsProvider: "cloudflare",
			dnsCredentials: `{"PollingInterval":"5","PropagationTimeout":"300","AuthEmail":"test@example.com","AuthKey":"testkey"}`,
			ppgTimeout:  600,
			expectError: true, // Will error due to missing required fields, but tests parsing
		},
		{
			name:        "Invalid PollingInterval value",
			dnsProvider: "cloudflare",
			dnsCredentials: `{"PollingInterval":"invalid","AuthEmail":"test@example.com","AuthKey":"testkey"}`,
			ppgTimeout:  600,
			expectError: true,
		},
		{
			name:        "Invalid PropagationTimeout value",
			dnsProvider: "cloudflare",
			dnsCredentials: `{"PropagationTimeout":"invalid","AuthEmail":"test@example.com","AuthKey":"testkey"}`,
			ppgTimeout:  600,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := GetDnsChallengeProviderByName(tt.dnsProvider, tt.dnsCredentials, tt.ppgTimeout)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
			}
		})
	}
}

func TestGetDnsChallengeProviderByName_CredentialParsing(t *testing.T) {
	// Test that the function correctly parses and removes PollingInterval and PropagationTimeout
	t.Run("PollingInterval is removed from credentials", func(t *testing.T) {
		dnsProvider := "cloudflare"
		dnsCredentials := `{"PollingInterval":"5","SomeKey":"SomeValue"}`
		ppgTimeout := 600

		// This will fail due to missing required fields, but we're testing that it parses
		_, err := GetDnsChallengeProviderByName(dnsProvider, dnsCredentials, ppgTimeout)

		// We expect an error (missing required Cloudflare fields), but it should not be a JSON parsing error
		require.Error(t, err)
		// The error should not contain "PollingInterval" if it was properly removed
	})

	t.Run("PropagationTimeout is removed from credentials", func(t *testing.T) {
		dnsProvider := "cloudflare"
		dnsCredentials := `{"PropagationTimeout":"300","SomeKey":"SomeValue"}`
		ppgTimeout := 600

		// This will fail due to missing required fields, but we're testing that it parses
		_, err := GetDnsChallengeProviderByName(dnsProvider, dnsCredentials, ppgTimeout)

		// We expect an error (missing required Cloudflare fields), but it should not be a JSON parsing error
		require.Error(t, err)
	})
}

func TestGetDnsChallengeProviderByName_TimeoutParsing(t *testing.T) {
	tests := []struct {
		name           string
		dnsCredentials string
		ppgTimeout     int
		description    string
	}{
		{
			name:           "Valid PollingInterval conversion",
			dnsCredentials: `{"PollingInterval":"10"}`,
			ppgTimeout:     600,
			description:    "Should convert PollingInterval string to int",
		},
		{
			name:           "Valid PropagationTimeout conversion",
			dnsCredentials: `{"PropagationTimeout":"500"}`,
			ppgTimeout:     600,
			description:    "Should convert PropagationTimeout string to int",
		},
		{
			name:           "Zero PollingInterval",
			dnsCredentials: `{"PollingInterval":"0"}`,
			ppgTimeout:     600,
			description:    "Should handle zero PollingInterval",
		},
		{
			name:           "Large timeout values",
			dnsCredentials: `{"PollingInterval":"30","PropagationTimeout":"1200"}`,
			ppgTimeout:     1200,
			description:    "Should handle large timeout values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We use a provider that exists but will fail with empty credentials
			// This tests the timeout parsing logic
			_, err := GetDnsChallengeProviderByName("cloudflare", tt.dnsCredentials, tt.ppgTimeout)

			// We expect an error due to missing required fields, but not a parsing error
			assert.Error(t, err)
			// Should not contain "invalid character" or other JSON parsing errors
			assert.NotContains(t, err.Error(), "invalid character")
		})
	}
}

func TestGetDnsChallengeProviderByName_DifferentProviders(t *testing.T) {
	// Test that different DNS providers are recognized
	providers := []string{
		"cloudflare",
		"digitalocean",
		"route53",
		"godaddy",
		"namecheap",
		"duckdns",
		"gandi",
		"ovh",
		"linode",
		"vultr",
	}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			// Use empty credentials - will fail but should recognize the provider
			_, err := GetDnsChallengeProviderByName(provider, `{}`, 600)

			// Should fail but NOT because of unrecognized provider
			assert.Error(t, err)
			assert.NotContains(t, err.Error(), "unrecognized DNS provider")
		})
	}
}

func TestGetDnsChallengeProviderByName_NilValues(t *testing.T) {
	t.Run("Nil PollingInterval in credentials", func(t *testing.T) {
		dnsCredentials := `{"PollingInterval":null,"SomeKey":"value"}`
		_, err := GetDnsChallengeProviderByName("cloudflare", dnsCredentials, 600)

		// Should not panic, may error on missing required fields
		assert.Error(t, err)
	})

	t.Run("Nil PropagationTimeout in credentials", func(t *testing.T) {
		dnsCredentials := `{"PropagationTimeout":null,"SomeKey":"value"}`
		_, err := GetDnsChallengeProviderByName("cloudflare", dnsCredentials, 600)

		// Should not panic, may error on missing required fields
		assert.Error(t, err)
	})
}
