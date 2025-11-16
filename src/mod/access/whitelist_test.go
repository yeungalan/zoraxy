package access

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddCountryCodeToWhitelist(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("AddSingleCountry", func(t *testing.T) {
		rule.AddCountryCodeToWhitelist("US", "United States")

		whitelistCC := *rule.WhiteListCountryCode
		comment, ok := whitelistCC["us"]
		assert.True(t, ok)
		assert.Equal(t, "United States", comment)
	})

	t.Run("AddMultipleCountries", func(t *testing.T) {
		rule.AddCountryCodeToWhitelist("JP", "Japan")
		rule.AddCountryCodeToWhitelist("GB", "United Kingdom")

		whitelistCC := *rule.WhiteListCountryCode
		assert.Contains(t, whitelistCC, "jp")
		assert.Contains(t, whitelistCC, "gb")
		assert.Equal(t, "Japan", whitelistCC["jp"])
		assert.Equal(t, "United Kingdom", whitelistCC["gb"])
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		rule.AddCountryCodeToWhitelist("De", "Germany")

		whitelistCC := *rule.WhiteListCountryCode
		comment, ok := whitelistCC["de"]
		assert.True(t, ok)
		assert.Equal(t, "Germany", comment)
	})

	t.Run("OverwriteExisting", func(t *testing.T) {
		rule.AddCountryCodeToWhitelist("FR", "France")
		rule.AddCountryCodeToWhitelist("FR", "France - Updated")

		whitelistCC := *rule.WhiteListCountryCode
		assert.Equal(t, "France - Updated", whitelistCC["fr"])
	})

	t.Run("EmptyComment", func(t *testing.T) {
		rule.AddCountryCodeToWhitelist("IT", "")

		whitelistCC := *rule.WhiteListCountryCode
		comment, ok := whitelistCC["it"]
		assert.True(t, ok)
		assert.Empty(t, comment)
	})
}

func TestRemoveCountryCodeFromWhitelist(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("RemoveExisting", func(t *testing.T) {
		rule.AddCountryCodeToWhitelist("CA", "Canada")

		whitelistCC := *rule.WhiteListCountryCode
		assert.Contains(t, whitelistCC, "ca")

		rule.RemoveCountryCodeFromWhitelist("CA")

		whitelistCC = *rule.WhiteListCountryCode
		assert.NotContains(t, whitelistCC, "ca")
	})

	t.Run("RemoveNonExistent", func(t *testing.T) {
		// Should not panic
		rule.RemoveCountryCodeFromWhitelist("ZZ")
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		rule.AddCountryCodeToWhitelist("au", "Australia")

		whitelistCC := *rule.WhiteListCountryCode
		assert.Contains(t, whitelistCC, "au")

		rule.RemoveCountryCodeFromWhitelist("AU")

		whitelistCC = *rule.WhiteListCountryCode
		assert.NotContains(t, whitelistCC, "au")
	})
}

func TestIsCountryCodeWhitelisted(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("Whitelisted", func(t *testing.T) {
		rule.AddCountryCodeToWhitelist("NZ", "New Zealand")

		result := rule.IsCountryCodeWhitelisted("NZ")
		assert.True(t, result)
	})

	t.Run("NotWhitelisted", func(t *testing.T) {
		result := rule.IsCountryCodeWhitelisted("XX")
		assert.False(t, result)
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		rule.AddCountryCodeToWhitelist("se", "Sweden")

		assert.True(t, rule.IsCountryCodeWhitelisted("SE"))
		assert.True(t, rule.IsCountryCodeWhitelisted("se"))
		assert.True(t, rule.IsCountryCodeWhitelisted("Se"))
	})
}

func TestGetAllWhitelistedCountryCode(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("Empty", func(t *testing.T) {
		result := rule.GetAllWhitelistedCountryCode()
		assert.Empty(t, result)
	})

	t.Run("Multiple", func(t *testing.T) {
		rule.AddCountryCodeToWhitelist("US", "United States")
		rule.AddCountryCodeToWhitelist("JP", "Japan")
		rule.AddCountryCodeToWhitelist("GB", "United Kingdom")

		result := rule.GetAllWhitelistedCountryCode()
		assert.Len(t, result, 3)

		// Verify entry types
		for _, entry := range result {
			assert.Equal(t, EntryType_CountryCode, entry.EntryType)
			assert.NotEmpty(t, entry.CC)
		}

		// Verify all countries are present
		countryCodes := make(map[string]bool)
		for _, entry := range result {
			countryCodes[entry.CC] = true
		}
		assert.True(t, countryCodes["us"])
		assert.True(t, countryCodes["jp"])
		assert.True(t, countryCodes["gb"])
	})
}

func TestAddIPToWhiteList(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("AddSingleIP", func(t *testing.T) {
		rule.AddIPToWhiteList("192.168.1.1", "Test IP")

		whitelistIP := *rule.WhiteListIP
		comment, ok := whitelistIP["192.168.1.1"]
		assert.True(t, ok)
		assert.Equal(t, "Test IP", comment)
	})

	t.Run("AddMultipleIPs", func(t *testing.T) {
		rule.AddIPToWhiteList("10.0.0.1", "IP 1")
		rule.AddIPToWhiteList("10.0.0.2", "IP 2")

		whitelistIP := *rule.WhiteListIP
		assert.Contains(t, whitelistIP, "10.0.0.1")
		assert.Contains(t, whitelistIP, "10.0.0.2")
	})

	t.Run("AddCIDR", func(t *testing.T) {
		rule.AddIPToWhiteList("192.168.0.0/24", "Local subnet")

		whitelistIP := *rule.WhiteListIP
		comment, ok := whitelistIP["192.168.0.0/24"]
		assert.True(t, ok)
		assert.Equal(t, "Local subnet", comment)
	})

	t.Run("AddWildcard", func(t *testing.T) {
		rule.AddIPToWhiteList("192.168.*.*", "Wildcard range")

		whitelistIP := *rule.WhiteListIP
		assert.Contains(t, whitelistIP, "192.168.*.*")
	})

	t.Run("OverwriteExisting", func(t *testing.T) {
		rule.AddIPToWhiteList("172.16.0.1", "Original")
		rule.AddIPToWhiteList("172.16.0.1", "Updated")

		whitelistIP := *rule.WhiteListIP
		assert.Equal(t, "Updated", whitelistIP["172.16.0.1"])
	})
}

func TestRemoveIPFromWhiteList(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("RemoveExisting", func(t *testing.T) {
		rule.AddIPToWhiteList("203.0.113.0", "Test IP")

		whitelistIP := *rule.WhiteListIP
		assert.Contains(t, whitelistIP, "203.0.113.0")

		rule.RemoveIPFromWhiteList("203.0.113.0")

		whitelistIP = *rule.WhiteListIP
		assert.NotContains(t, whitelistIP, "203.0.113.0")
	})

	t.Run("RemoveNonExistent", func(t *testing.T) {
		// Should not panic
		rule.RemoveIPFromWhiteList("1.2.3.4")
	})

	t.Run("RemoveCIDR", func(t *testing.T) {
		rule.AddIPToWhiteList("10.0.0.0/8", "Private range")
		rule.RemoveIPFromWhiteList("10.0.0.0/8")

		whitelistIP := *rule.WhiteListIP
		assert.NotContains(t, whitelistIP, "10.0.0.0/8")
	})
}

func TestIsIPWhitelisted(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("ExactMatch", func(t *testing.T) {
		rule.AddIPToWhiteList("8.8.8.8", "Google DNS")

		result := rule.IsIPWhitelisted("8.8.8.8")
		assert.True(t, result)
	})

	t.Run("NotWhitelisted", func(t *testing.T) {
		result := rule.IsIPWhitelisted("1.1.1.1")
		assert.False(t, result)
	})

	t.Run("CIDRMatch", func(t *testing.T) {
		rule.AddIPToWhiteList("192.168.1.0/24", "Local subnet")

		assert.True(t, rule.IsIPWhitelisted("192.168.1.1"))
		assert.True(t, rule.IsIPWhitelisted("192.168.1.100"))
		assert.True(t, rule.IsIPWhitelisted("192.168.1.254"))
		assert.False(t, rule.IsIPWhitelisted("192.168.2.1"))
	})

	t.Run("WildcardMatch", func(t *testing.T) {
		rule.AddIPToWhiteList("10.0.*.*", "Wildcard range")

		assert.True(t, rule.IsIPWhitelisted("10.0.0.1"))
		assert.True(t, rule.IsIPWhitelisted("10.0.255.255"))
		assert.False(t, rule.IsIPWhitelisted("10.1.0.1"))
	})

	t.Run("LoopbackAllowed", func(t *testing.T) {
		rule.WhitelistAllowLocalAndLoopback = true
		controller.ServerPublicIP = "203.0.113.100"

		// Test loopback addresses
		assert.True(t, rule.IsIPWhitelisted("127.0.0.1"))
		assert.True(t, rule.IsIPWhitelisted("localhost"))
		assert.True(t, rule.IsIPWhitelisted("::1"))

		// Test public IP as loopback
		assert.True(t, rule.IsIPWhitelisted("203.0.113.100"))

		// Clean up
		rule.WhitelistAllowLocalAndLoopback = false
	})

	t.Run("PrivateRangesAllowed", func(t *testing.T) {
		rule.WhitelistAllowLocalAndLoopback = true

		// Test private IP ranges
		assert.True(t, rule.IsIPWhitelisted("10.0.0.1"))
		assert.True(t, rule.IsIPWhitelisted("172.16.0.1"))
		assert.True(t, rule.IsIPWhitelisted("192.168.1.1"))
		assert.True(t, rule.IsIPWhitelisted("169.254.1.1"))

		// Test non-private IP
		assert.False(t, rule.IsIPWhitelisted("8.8.8.8"))

		// Clean up
		rule.WhitelistAllowLocalAndLoopback = false
	})

	t.Run("LoopbackNotAllowed", func(t *testing.T) {
		rule.WhitelistAllowLocalAndLoopback = false

		assert.False(t, rule.IsIPWhitelisted("127.0.0.1"))
		assert.False(t, rule.IsIPWhitelisted("10.0.0.1"))
	})
}

func TestGetAllWhitelistedIp(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("Empty", func(t *testing.T) {
		result := rule.GetAllWhitelistedIp()
		assert.Empty(t, result)
	})

	t.Run("Multiple", func(t *testing.T) {
		rule.AddIPToWhiteList("192.168.1.1", "IP 1")
		rule.AddIPToWhiteList("10.0.0.0/8", "Private range")
		rule.AddIPToWhiteList("172.16.*.*", "Wildcard")

		result := rule.GetAllWhitelistedIp()
		assert.Len(t, result, 3)

		// Verify entry types
		for _, entry := range result {
			assert.Equal(t, EntryType_IP, entry.EntryType)
			assert.NotEmpty(t, entry.IP)
		}

		// Verify all IPs are present
		ips := make(map[string]bool)
		for _, entry := range result {
			ips[entry.IP] = true
		}
		assert.True(t, ips["192.168.1.1"])
		assert.True(t, ips["10.0.0.0/8"])
		assert.True(t, ips["172.16.*.*"])
	})
}

func TestWhitelistIntegration(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("ComplexScenario", func(t *testing.T) {
		// Enable whitelist
		rule.ToggleWhitelist(true)

		// Add country codes
		rule.AddCountryCodeToWhitelist("US", "United States")
		rule.AddCountryCodeToWhitelist("JP", "Japan")

		// Add IP addresses
		rule.AddIPToWhiteList("8.8.8.8", "Google DNS")
		rule.AddIPToWhiteList("192.168.0.0/16", "Private network")

		// Get all entries
		countries := rule.GetAllWhitelistedCountryCode()
		ips := rule.GetAllWhitelistedIp()

		assert.Len(t, countries, 2)
		assert.Len(t, ips, 2)

		// Verify specific entries
		assert.True(t, rule.IsCountryCodeWhitelisted("US"))
		assert.True(t, rule.IsCountryCodeWhitelisted("jp"))
		assert.True(t, rule.IsIPWhitelisted("8.8.8.8"))
		assert.True(t, rule.IsIPWhitelisted("192.168.1.1"))

		// Remove some entries
		rule.RemoveCountryCodeFromWhitelist("US")
		rule.RemoveIPFromWhiteList("8.8.8.8")

		// Verify removal
		assert.False(t, rule.IsCountryCodeWhitelisted("US"))
		assert.True(t, rule.IsCountryCodeWhitelisted("JP"))
		assert.False(t, rule.IsIPWhitelisted("8.8.8.8"))
		assert.True(t, rule.IsIPWhitelisted("192.168.1.1"))
	})
}

func TestWhitelistEntry(t *testing.T) {
	t.Run("CountryCodeEntry", func(t *testing.T) {
		entry := WhitelistEntry{
			EntryType: EntryType_CountryCode,
			CC:        "US",
			Comment:   "United States",
		}

		assert.Equal(t, EntryType_CountryCode, entry.EntryType)
		assert.Equal(t, "US", entry.CC)
		assert.Equal(t, "United States", entry.Comment)
		assert.Empty(t, entry.IP)
	})

	t.Run("IPEntry", func(t *testing.T) {
		entry := WhitelistEntry{
			EntryType: EntryType_IP,
			IP:        "192.168.1.1",
			Comment:   "Test IP",
		}

		assert.Equal(t, EntryType_IP, entry.EntryType)
		assert.Equal(t, "192.168.1.1", entry.IP)
		assert.Equal(t, "Test IP", entry.Comment)
		assert.Empty(t, entry.CC)
	})
}

func TestWhitelistConcurrency(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	// Test concurrent additions and removals
	done := make(chan bool)

	// Add country codes concurrently
	go func() {
		for i := 0; i < 10; i++ {
			rule.AddCountryCodeToWhitelist("US", "United States")
		}
		done <- true
	}()

	// Add IPs concurrently
	go func() {
		for i := 0; i < 10; i++ {
			rule.AddIPToWhiteList("192.168.1.1", "Test IP")
		}
		done <- true
	}()

	// Read concurrently
	go func() {
		for i := 0; i < 10; i++ {
			rule.IsCountryCodeWhitelisted("US")
			rule.IsIPWhitelisted("192.168.1.1")
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify data integrity
	assert.True(t, rule.IsCountryCodeWhitelisted("US"))
	assert.True(t, rule.IsIPWhitelisted("192.168.1.1"))
}
