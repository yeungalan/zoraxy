package access

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"imuslab.com/zoraxy/mod/geodb"
)

func TestAllowIpAccess(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("BlacklistDisabled_WhitelistDisabled", func(t *testing.T) {
		rule.BlacklistEnabled = false
		rule.WhitelistEnabled = false

		allowed := rule.AllowIpAccess("1.2.3.4")
		assert.True(t, allowed)
	})

	t.Run("IPBlacklisted", func(t *testing.T) {
		rule.BlacklistEnabled = true
		rule.WhitelistEnabled = false


		// Add IP to blacklist
		(*rule.BlackListIP)["1.2.3.4"] = "Blocked IP"

		allowed := rule.AllowIpAccess("1.2.3.4")
		assert.False(t, allowed)

		// Clean up
		delete(*rule.BlackListIP, "1.2.3.4")
	})

	t.Run("CountryBlacklisted", func(t *testing.T) {
		rule.BlacklistEnabled = true
		rule.WhitelistEnabled = false


		// Add country to blacklist
		(*rule.BlackListContryCode)["cn"] = "Blocked Country"

		allowed := rule.AllowIpAccess("5.6.7.8")
		assert.False(t, allowed)

		// Clean up
		delete(*rule.BlackListContryCode, "cn")
	})

	t.Run("WhitelistEnabled_IPNotWhitelisted", func(t *testing.T) {
		rule.BlacklistEnabled = false
		rule.WhitelistEnabled = true


		allowed := rule.AllowIpAccess("10.0.0.1")
		assert.False(t, allowed)
	})

	t.Run("WhitelistEnabled_IPWhitelisted", func(t *testing.T) {
		rule.BlacklistEnabled = false
		rule.WhitelistEnabled = true


		// Add IP to whitelist
		(*rule.WhiteListIP)["10.0.0.2"] = "Allowed IP"

		allowed := rule.AllowIpAccess("10.0.0.2")
		assert.True(t, allowed)

		// Clean up
		delete(*rule.WhiteListIP, "10.0.0.2")
	})

	t.Run("BlacklistAndWhitelist_IPBlacklisted", func(t *testing.T) {
		rule.BlacklistEnabled = true
		rule.WhitelistEnabled = true


		// Add IP to both lists (blacklist takes precedence)
		(*rule.BlackListIP)["1.1.1.1"] = "Blocked"
		(*rule.WhiteListIP)["1.1.1.1"] = "Allowed"

		allowed := rule.AllowIpAccess("1.1.1.1")
		assert.False(t, allowed)

		// Clean up
		delete(*rule.BlackListIP, "1.1.1.1")
		delete(*rule.WhiteListIP, "1.1.1.1")
	})
}

func TestAllowConnectionAccess(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule
	rule.BlacklistEnabled = false
	rule.WhitelistEnabled = false

	t.Run("TCPConnection", func(t *testing.T) {
		// Create a TCP listener
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		assert.NoError(t, err)
		defer listener.Close()

		// Connect to it
		conn, err := net.Dial("tcp", listener.Addr().String())
		assert.NoError(t, err)
		defer conn.Close()

		allowed := rule.AllowConnectionAccess(conn)
		assert.True(t, allowed)
	})

	t.Run("NonTCPConnection", func(t *testing.T) {
		// For non-TCP connections, it should return true
		// We'll use a mock connection type
		mockConn := &mockNetConn{
			remoteAddr: &mockAddr{addr: "unix:/tmp/test.sock"},
		}

		allowed := rule.AllowConnectionAccess(mockConn)
		assert.True(t, allowed)
	})
}

// Mock connection for testing
type mockNetConn struct {
	net.Conn
	remoteAddr net.Addr
}

func (m *mockNetConn) RemoteAddr() net.Addr {
	return m.remoteAddr
}

type mockAddr struct {
	addr string
}

func (m *mockAddr) Network() string {
	return "mock"
}

func (m *mockAddr) String() string {
	return m.addr
}

func TestToggleBlacklist(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("EnableBlacklist", func(t *testing.T) {
		rule.ToggleBlacklist(true)
		assert.True(t, rule.BlacklistEnabled)

		// Verify it was saved
		configFile := filepath.Join(controller.Options.ConfigFolder, "default.json")
		assert.FileExists(t, configFile)
	})

	t.Run("DisableBlacklist", func(t *testing.T) {
		rule.ToggleBlacklist(false)
		assert.False(t, rule.BlacklistEnabled)
	})
}

func TestToggleWhitelist(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("EnableWhitelist", func(t *testing.T) {
		rule.ToggleWhitelist(true)
		assert.True(t, rule.WhitelistEnabled)

		// Verify it was saved
		configFile := filepath.Join(controller.Options.ConfigFolder, "default.json")
		assert.FileExists(t, configFile)
	})

	t.Run("DisableWhitelist", func(t *testing.T) {
		rule.ToggleWhitelist(false)
		assert.False(t, rule.WhitelistEnabled)
	})
}

func TestToggleAllowLoopback(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("EnableLoopback", func(t *testing.T) {
		rule.ToggleAllowLoopback(true)
		assert.True(t, rule.WhitelistAllowLocalAndLoopback)
	})

	t.Run("DisableLoopback", func(t *testing.T) {
		rule.ToggleAllowLoopback(false)
		assert.False(t, rule.WhitelistAllowLocalAndLoopback)
	})
}

func TestIsBlacklisted(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("BlacklistDisabled", func(t *testing.T) {
		rule.BlacklistEnabled = false
		result := rule.IsBlacklisted("1.2.3.4")
		assert.False(t, result)
	})

	t.Run("EmptyIPAddress", func(t *testing.T) {
		rule.BlacklistEnabled = true
		result := rule.IsBlacklisted("")
		assert.False(t, result)
	})

	t.Run("GeoDBError", func(t *testing.T) {
		rule.BlacklistEnabled = true


		result := rule.IsBlacklisted("invalid")
		assert.False(t, result)
	})

	t.Run("CountryCodeBlacklisted", func(t *testing.T) {
		rule.BlacklistEnabled = true


		(*rule.BlackListContryCode)["us"] = "Blocked"

		result := rule.IsBlacklisted("8.8.8.8")
		assert.True(t, result)

		// Clean up
		delete(*rule.BlackListContryCode, "us")
	})

	t.Run("IPBlacklisted", func(t *testing.T) {
		rule.BlacklistEnabled = true


		(*rule.BlackListIP)["9.9.9.9"] = "Blocked IP"

		result := rule.IsBlacklisted("9.9.9.9")
		assert.True(t, result)

		// Clean up
		delete(*rule.BlackListIP, "9.9.9.9")
	})

	t.Run("NotBlacklisted", func(t *testing.T) {
		rule.BlacklistEnabled = true


		result := rule.IsBlacklisted("7.7.7.7")
		assert.False(t, result)
	})
}

func TestIsWhitelisted(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule := controller.DefaultAccessRule

	t.Run("WhitelistDisabled", func(t *testing.T) {
		rule.WhitelistEnabled = false
		result := rule.IsWhitelisted("1.2.3.4")
		assert.True(t, result) // Default is true when disabled
	})

	t.Run("EmptyIPAddress", func(t *testing.T) {
		rule.WhitelistEnabled = true
		result := rule.IsWhitelisted("")
		assert.True(t, result) // Default is true on error
	})

	t.Run("GeoDBError", func(t *testing.T) {
		rule.WhitelistEnabled = true


		result := rule.IsWhitelisted("invalid")
		assert.True(t, result) // Default is true on error
	})

	t.Run("CountryCodeWhitelisted", func(t *testing.T) {
		rule.WhitelistEnabled = true


		(*rule.WhiteListCountryCode)["us"] = "Allowed"

		result := rule.IsWhitelisted("8.8.4.4")
		assert.True(t, result)

		// Clean up
		delete(*rule.WhiteListCountryCode, "us")
	})

	t.Run("IPWhitelisted", func(t *testing.T) {
		rule.WhitelistEnabled = true


		(*rule.WhiteListIP)["9.9.9.9"] = "Allowed IP"

		result := rule.IsWhitelisted("9.9.9.9")
		assert.True(t, result)

		// Clean up
		delete(*rule.WhiteListIP, "9.9.9.9")
	})

	t.Run("NotWhitelisted", func(t *testing.T) {
		rule.WhitelistEnabled = true


		result := rule.IsWhitelisted("7.7.7.7")
		assert.False(t, result)
	})
}

func TestSaveChanges(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	t.Run("Success", func(t *testing.T) {
		rule := controller.DefaultAccessRule
		rule.Name = "Modified Default"
		rule.Desc = "Modified Description"

		err := rule.SaveChanges()
		assert.NoError(t, err)

		// Verify file exists and contains the changes
		configFile := filepath.Join(controller.Options.ConfigFolder, "default.json")
		assert.FileExists(t, configFile)

		// Read and verify contents
		data, err := os.ReadFile(configFile)
		assert.NoError(t, err)
		assert.Contains(t, string(data), "Modified Default")
		assert.Contains(t, string(data), "Modified Description")
	})

	t.Run("DetachedFromController", func(t *testing.T) {
		rule := &AccessRule{
			ID:     "detached",
			Name:   "Detached Rule",
			parent: nil, // No parent
		}

		err := rule.SaveChanges()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access rule detached from controller")
	})

	t.Run("InvalidPath", func(t *testing.T) {
		// Create a rule with parent but invalid config folder
		badController := &Controller{
			Options: &Options{
				ConfigFolder: "/nonexistent/path/that/does/not/exist",
			},
		}

		rule := &AccessRule{
			ID:                   "test",
			Name:                 "Test",
			WhiteListCountryCode: &map[string]string{},
			WhiteListIP:          &map[string]string{},
			BlackListContryCode:  &map[string]string{},
			BlackListIP:          &map[string]string{},
			parent:               badController,
		}

		err := rule.SaveChanges()
		assert.Error(t, err)
	})
}

func TestDeleteConfigFile(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	t.Run("Success", func(t *testing.T) {
		// Add a new rule
		newRule := &AccessRule{
			ID:                   "delete_config_test",
			Name:                 "Delete Config Test",
			Desc:                 "Test",
			BlacklistEnabled:     false,
			WhitelistEnabled:     false,
			WhiteListCountryCode: &map[string]string{},
			WhiteListIP:          &map[string]string{},
			BlackListContryCode:  &map[string]string{},
			BlackListIP:          &map[string]string{},
		}
		err := controller.AddNewAccessRule(newRule)
		assert.NoError(t, err)

		configFile := filepath.Join(controller.Options.ConfigFolder, "delete_config_test.json")
		assert.FileExists(t, configFile)

		// Delete the config file
		rule, _ := controller.GetAccessRuleByID("delete_config_test")
		err = rule.DeleteConfigFile()
		assert.NoError(t, err)

		// Verify file is deleted
		assert.NoFileExists(t, configFile)
	})

	t.Run("FileDoesNotExist", func(t *testing.T) {
		rule := &AccessRule{
			ID:     "nonexistent_file",
			parent: controller,
		}

		err := rule.DeleteConfigFile()
		assert.Error(t, err)
	})
}
