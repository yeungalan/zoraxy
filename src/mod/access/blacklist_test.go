package access

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddCountryCodeToBlackList(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("AddSingleCountry", func(t *testing.T) {
		rule.AddCountryCodeToBlackList("CN", "China")

		blacklistCC := *rule.BlackListContryCode
		comment, ok := blacklistCC["cn"]
		assert.True(t, ok)
		assert.Equal(t, "China", comment)
	})

	t.Run("AddMultipleCountries", func(t *testing.T) {
		rule.AddCountryCodeToBlackList("RU", "Russia")
		rule.AddCountryCodeToBlackList("KP", "North Korea")

		blacklistCC := *rule.BlackListContryCode
		assert.Contains(t, blacklistCC, "ru")
		assert.Contains(t, blacklistCC, "kp")
		assert.Equal(t, "Russia", blacklistCC["ru"])
		assert.Equal(t, "North Korea", blacklistCC["kp"])
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		rule.AddCountryCodeToBlackList("Ir", "Iran")

		blacklistCC := *rule.BlackListContryCode
		comment, ok := blacklistCC["ir"]
		assert.True(t, ok)
		assert.Equal(t, "Iran", comment)
	})

	t.Run("OverwriteExisting", func(t *testing.T) {
		rule.AddCountryCodeToBlackList("XX", "Original")
		rule.AddCountryCodeToBlackList("XX", "Updated")

		blacklistCC := *rule.BlackListContryCode
		assert.Equal(t, "Updated", blacklistCC["xx"])
	})

	t.Run("EmptyComment", func(t *testing.T) {
		rule.AddCountryCodeToBlackList("YY", "")

		blacklistCC := *rule.BlackListContryCode
		comment, ok := blacklistCC["yy"]
		assert.True(t, ok)
		assert.Empty(t, comment)
	})
}

func TestRemoveCountryCodeFromBlackList(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("RemoveExisting", func(t *testing.T) {
		rule.AddCountryCodeToBlackList("SY", "Syria")

		blacklistCC := *rule.BlackListContryCode
		assert.Contains(t, blacklistCC, "sy")

		rule.RemoveCountryCodeFromBlackList("SY")

		blacklistCC = *rule.BlackListContryCode
		assert.NotContains(t, blacklistCC, "sy")
	})

	t.Run("RemoveNonExistent", func(t *testing.T) {
		// Should not panic
		rule.RemoveCountryCodeFromBlackList("ZZ")
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		rule.AddCountryCodeToBlackList("af", "Afghanistan")

		blacklistCC := *rule.BlackListContryCode
		assert.Contains(t, blacklistCC, "af")

		rule.RemoveCountryCodeFromBlackList("AF")

		blacklistCC = *rule.BlackListContryCode
		assert.NotContains(t, blacklistCC, "af")
	})
}

func TestIsCountryCodeBlacklisted(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("Blacklisted", func(t *testing.T) {
		rule.AddCountryCodeToBlackList("IQ", "Iraq")

		result := rule.IsCountryCodeBlacklisted("IQ")
		assert.True(t, result)
	})

	t.Run("NotBlacklisted", func(t *testing.T) {
		result := rule.IsCountryCodeBlacklisted("US")
		assert.False(t, result)
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		rule.AddCountryCodeToBlackList("ly", "Libya")

		assert.True(t, rule.IsCountryCodeBlacklisted("LY"))
		assert.True(t, rule.IsCountryCodeBlacklisted("ly"))
		assert.True(t, rule.IsCountryCodeBlacklisted("Ly"))
	})
}

func TestGetAllBlacklistedCountryCode(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("Empty", func(t *testing.T) {
		result := rule.GetAllBlacklistedCountryCode()
		assert.Empty(t, result)
	})

	t.Run("Multiple", func(t *testing.T) {
		rule.AddCountryCodeToBlackList("CN", "China")
		rule.AddCountryCodeToBlackList("RU", "Russia")
		rule.AddCountryCodeToBlackList("KP", "North Korea")

		result := rule.GetAllBlacklistedCountryCode()
		assert.Len(t, result, 3)

		// Verify all countries are present
		countryMap := make(map[string]bool)
		for _, cc := range result {
			countryMap[cc] = true
		}
		assert.True(t, countryMap["cn"])
		assert.True(t, countryMap["ru"])
		assert.True(t, countryMap["kp"])
	})
}

func TestAddIPToBlackList(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("AddSingleIP", func(t *testing.T) {
		rule.AddIPToBlackList("203.0.113.1", "Malicious IP")

		blacklistIP := *rule.BlackListIP
		comment, ok := blacklistIP["203.0.113.1"]
		assert.True(t, ok)
		assert.Equal(t, "Malicious IP", comment)
	})

	t.Run("AddMultipleIPs", func(t *testing.T) {
		rule.AddIPToBlackList("198.51.100.1", "Spam")
		rule.AddIPToBlackList("198.51.100.2", "Attack")

		blacklistIP := *rule.BlackListIP
		assert.Contains(t, blacklistIP, "198.51.100.1")
		assert.Contains(t, blacklistIP, "198.51.100.2")
	})

	t.Run("AddCIDR", func(t *testing.T) {
		rule.AddIPToBlackList("192.0.2.0/24", "Blocked subnet")

		blacklistIP := *rule.BlackListIP
		comment, ok := blacklistIP["192.0.2.0/24"]
		assert.True(t, ok)
		assert.Equal(t, "Blocked subnet", comment)
	})

	t.Run("AddWildcard", func(t *testing.T) {
		rule.AddIPToBlackList("198.51.*.*", "Wildcard block")

		blacklistIP := *rule.BlackListIP
		assert.Contains(t, blacklistIP, "198.51.*.*")
	})

	t.Run("OverwriteExisting", func(t *testing.T) {
		rule.AddIPToBlackList("198.18.0.1", "Original")
		rule.AddIPToBlackList("198.18.0.1", "Updated")

		blacklistIP := *rule.BlackListIP
		assert.Equal(t, "Updated", blacklistIP["198.18.0.1"])
	})
}

func TestRemoveIPFromBlackList(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("RemoveExisting", func(t *testing.T) {
		rule.AddIPToBlackList("198.19.0.1", "Test IP")

		blacklistIP := *rule.BlackListIP
		assert.Contains(t, blacklistIP, "198.19.0.1")

		rule.RemoveIPFromBlackList("198.19.0.1")

		blacklistIP = *rule.BlackListIP
		assert.NotContains(t, blacklistIP, "198.19.0.1")
	})

	t.Run("RemoveNonExistent", func(t *testing.T) {
		// Should not panic
		rule.RemoveIPFromBlackList("1.2.3.4")
	})

	t.Run("RemoveCIDR", func(t *testing.T) {
		rule.AddIPToBlackList("172.16.0.0/12", "Block range")
		rule.RemoveIPFromBlackList("172.16.0.0/12")

		blacklistIP := *rule.BlackListIP
		assert.NotContains(t, blacklistIP, "172.16.0.0/12")
	})
}

func TestGetAllBlacklistedIp(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("Empty", func(t *testing.T) {
		result := rule.GetAllBlacklistedIp()
		assert.Empty(t, result)
	})

	t.Run("Multiple", func(t *testing.T) {
		rule.AddIPToBlackList("203.0.113.1", "IP 1")
		rule.AddIPToBlackList("198.51.100.0/24", "Range")
		rule.AddIPToBlackList("192.0.2.*", "Wildcard")

		result := rule.GetAllBlacklistedIp()
		assert.Len(t, result, 3)

		// Verify all IPs are present
		ipMap := make(map[string]bool)
		for _, ip := range result {
			ipMap[ip] = true
		}
		assert.True(t, ipMap["203.0.113.1"])
		assert.True(t, ipMap["198.51.100.0/24"])
		assert.True(t, ipMap["192.0.2.*"])
	})
}

func TestIsIPBlacklisted(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("ExactMatch", func(t *testing.T) {
		rule.AddIPToBlackList("203.0.113.50", "Blocked IP")

		result := rule.IsIPBlacklisted("203.0.113.50")
		assert.True(t, result)
	})

	t.Run("NotBlacklisted", func(t *testing.T) {
		result := rule.IsIPBlacklisted("8.8.8.8")
		assert.False(t, result)
	})

	t.Run("CIDRMatch", func(t *testing.T) {
		rule.AddIPToBlackList("198.51.100.0/24", "Blocked subnet")

		assert.True(t, rule.IsIPBlacklisted("198.51.100.1"))
		assert.True(t, rule.IsIPBlacklisted("198.51.100.100"))
		assert.True(t, rule.IsIPBlacklisted("198.51.100.254"))
		assert.False(t, rule.IsIPBlacklisted("198.51.101.1"))
	})

	t.Run("WildcardMatch", func(t *testing.T) {
		rule.AddIPToBlackList("192.0.2.*", "Wildcard block")

		assert.True(t, rule.IsIPBlacklisted("192.0.2.1"))
		assert.True(t, rule.IsIPBlacklisted("192.0.2.255"))
		assert.False(t, rule.IsIPBlacklisted("192.0.3.1"))
	})

	t.Run("MultipleWildcards", func(t *testing.T) {
		rule.AddIPToBlackList("10.20.*.*", "Wide block")

		assert.True(t, rule.IsIPBlacklisted("10.20.0.1"))
		assert.True(t, rule.IsIPBlacklisted("10.20.255.255"))
		assert.False(t, rule.IsIPBlacklisted("10.21.0.1"))
	})
}

func TestGetBlacklistedIPComment(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("ExactIPMatch", func(t *testing.T) {
		rule.AddIPToBlackList("203.0.113.100", "Malicious actor")

		comment, err := rule.GetBlacklistedIPComment("203.0.113.100")
		assert.NoError(t, err)
		assert.Equal(t, "Malicious actor", comment)
	})

	t.Run("CIDRMatch", func(t *testing.T) {
		rule.AddIPToBlackList("198.51.100.0/24", "Blocked range")

		comment, err := rule.GetBlacklistedIPComment("198.51.100.50")
		assert.NoError(t, err)
		assert.Equal(t, "Blocked range", comment)
	})

	t.Run("WildcardMatch", func(t *testing.T) {
		rule.AddIPToBlackList("192.0.2.*", "Wildcard block")

		comment, err := rule.GetBlacklistedIPComment("192.0.2.123")
		assert.NoError(t, err)
		assert.Equal(t, "Wildcard block", comment)
	})

	t.Run("CountryMatch", func(t *testing.T) {

		rule.AddCountryCodeToBlackList("CN", "Blocked country")

		comment, err := rule.GetBlacklistedIPComment("5.6.7.8")
		assert.NoError(t, err)
		assert.Equal(t, "Blocked country", comment)
	})

	t.Run("NotBlacklisted", func(t *testing.T) {

		comment, err := rule.GetBlacklistedIPComment("8.8.8.8")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found in blacklist")
		assert.Empty(t, comment)
	})

	t.Run("GeoDBError", func(t *testing.T) {

		comment, err := rule.GetBlacklistedIPComment("invalid")
		assert.Error(t, err)
		assert.Empty(t, comment)
	})
}

func TestGetParent(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("HasParent", func(t *testing.T) {
		parent := rule.GetParent()
		assert.NotNil(t, parent)
		assert.Equal(t, controller, parent)
	})

	t.Run("NoParent", func(t *testing.T) {
		orphanRule := &AccessRule{
			ID:     "orphan",
			parent: nil,
		}

		parent := orphanRule.GetParent()
		assert.Nil(t, parent)
	})
}

func TestBlacklistIntegration(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("ComplexScenario", func(t *testing.T) {
		// Enable blacklist
		rule.ToggleBlacklist(true)

		// Add country codes
		rule.AddCountryCodeToBlackList("CN", "China")
		rule.AddCountryCodeToBlackList("RU", "Russia")

		// Add IP addresses
		rule.AddIPToBlackList("203.0.113.1", "Spam source")
		rule.AddIPToBlackList("198.51.100.0/24", "Attack range")

		// Get all entries
		countries := rule.GetAllBlacklistedCountryCode()
		ips := rule.GetAllBlacklistedIp()

		assert.Len(t, countries, 2)
		assert.Len(t, ips, 2)

		// Verify specific entries
		assert.True(t, rule.IsCountryCodeBlacklisted("CN"))
		assert.True(t, rule.IsCountryCodeBlacklisted("ru"))
		assert.True(t, rule.IsIPBlacklisted("203.0.113.1"))
		assert.True(t, rule.IsIPBlacklisted("198.51.100.50"))

		// Remove some entries
		rule.RemoveCountryCodeFromBlackList("CN")
		rule.RemoveIPFromBlackList("203.0.113.1")

		// Verify removal
		assert.False(t, rule.IsCountryCodeBlacklisted("CN"))
		assert.True(t, rule.IsCountryCodeBlacklisted("RU"))
		assert.False(t, rule.IsIPBlacklisted("203.0.113.1"))
		assert.True(t, rule.IsIPBlacklisted("198.51.100.50"))
	})

	t.Run("BlacklistWithWhitelist", func(t *testing.T) {
		rule.ToggleBlacklist(true)
		rule.ToggleWhitelist(true)

		// Add to blacklist
		rule.AddCountryCodeToBlackList("XX", "Blocked")
		rule.AddIPToBlackList("1.2.3.4", "Blocked IP")

		// Add to whitelist
		rule.AddCountryCodeToWhitelist("US", "Allowed")
		rule.AddIPToWhiteList("5.6.7.8", "Allowed IP")

		// Mock GeoDB responses


		// Blacklist should take precedence
		assert.False(t, rule.AllowIpAccess("1.2.3.4"))

		// Whitelisted IP should be allowed if not blacklisted
		assert.True(t, rule.AllowIpAccess("5.6.7.8"))
	})
}

func TestBlacklistConcurrency(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	// Test concurrent additions and removals
	done := make(chan bool)

	// Add country codes concurrently
	go func() {
		for i := 0; i < 10; i++ {
			rule.AddCountryCodeToBlackList("CN", "China")
		}
		done <- true
	}()

	// Add IPs concurrently
	go func() {
		for i := 0; i < 10; i++ {
			rule.AddIPToBlackList("203.0.113.1", "Blocked")
		}
		done <- true
	}()

	// Read concurrently
	go func() {
		for i := 0; i < 10; i++ {
			rule.IsCountryCodeBlacklisted("CN")
			rule.IsIPBlacklisted("203.0.113.1")
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Verify data integrity
	assert.True(t, rule.IsCountryCodeBlacklisted("CN"))
	assert.True(t, rule.IsIPBlacklisted("203.0.113.1"))
}

func TestBlacklistEdgeCases(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("EmptyIPString", func(t *testing.T) {
		result := rule.IsIPBlacklisted("")
		assert.False(t, result)
	})

	t.Run("EmptyCountryCode", func(t *testing.T) {
		result := rule.IsCountryCodeBlacklisted("")
		assert.False(t, result)
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		rule.AddCountryCodeToBlackList("A@", "Invalid code")
		assert.True(t, rule.IsCountryCodeBlacklisted("a@"))
	})

	t.Run("VeryLongIP", func(t *testing.T) {
		longIP := "192.168.1.1.1.1.1.1.1.1"
		rule.AddIPToBlackList(longIP, "Invalid IP")
		assert.True(t, rule.IsIPBlacklisted(longIP))
	})
}
