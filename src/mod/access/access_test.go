package access

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"imuslab.com/zoraxy/mod/database"
	"imuslab.com/zoraxy/mod/database/dbinc"
	"imuslab.com/zoraxy/mod/geodb"
	"imuslab.com/zoraxy/mod/info/logger"
)

// Helper function to create a temporary config directory
func createTempConfigDir(t *testing.T) string {
	tempDir, err := os.MkdirTemp("", "access_test_*")
	assert.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})
	return tempDir
}

// Helper function to create a test controller
func createTestController(t *testing.T) *Controller {
	tempDir := createTempConfigDir(t)

	// Create a temporary database file
	tempDB, err := os.CreateTemp("", "test_db_*.db")
	assert.NoError(t, err)
	tempDBPath := tempDB.Name()
	tempDB.Close()
	t.Cleanup(func() {
		os.Remove(tempDBPath)
	})

	// Use a simple logger that doesn't write to files
	testLogger := logger.Logger{
		Prefix:    "test",
		LogFolder: "",
	}

	// Create a minimal test database
	testDB, err := database.NewDatabase(tempDBPath, dbinc.BackendBoltDB)
	assert.NoError(t, err)
	t.Cleanup(func() {
		testDB.Close()
	})

	// Create a real GeoDB for testing (uses embedded data)
	testGeoDB, err := geodb.NewGeoDb(nil, &geodb.StoreOptions{
		AllowSlowIpv4LookUp: false,
		AllowSlowIpv6Lookup: false,
	})
	assert.NoError(t, err)
	t.Cleanup(func() {
		testGeoDB.Close()
	})

	options := &Options{
		Logger:                testLogger,
		ConfigFolder:          tempDir,
		GeoDB:                 testGeoDB,
		Database:              testDB,
		PublicIpCheckInterval: 3600,
	}

	controller, err := NewAccessController(options)
	assert.NoError(t, err)
	assert.NotNil(t, controller)

	// Stop the public IP updater to avoid background goroutines in tests
	controller.StopPublicIPUpdater()

	return controller
}

func TestNewAccessController(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		tempDir := createTempConfigDir(t)

		// Create a temporary database file
		tempDB, err := os.CreateTemp("", "test_db_*.db")
		assert.NoError(t, err)
		tempDBPath := tempDB.Name()
		tempDB.Close()
		defer os.Remove(tempDBPath)

		testLogger := logger.Logger{
			Prefix:    "test",
			LogFolder: "",
		}

		testDB, err := database.NewDatabase(tempDBPath, dbinc.BackendBoltDB)
		assert.NoError(t, err)
		defer testDB.Close()

		options := &Options{
			Logger:       testLogger,
			ConfigFolder: tempDir,
			GeoDB:        nil, // Not needed for initialization
			Database:     testDB,
		}

		controller, err := NewAccessController(options)
		assert.NoError(t, err)
		assert.NotNil(t, controller)
		assert.NotNil(t, controller.DefaultAccessRule)
		assert.Equal(t, "default", controller.DefaultAccessRule.ID)
		assert.Equal(t, "Default", controller.DefaultAccessRule.Name)
		assert.False(t, controller.DefaultAccessRule.BlacklistEnabled)
		assert.False(t, controller.DefaultAccessRule.WhitelistEnabled)

		// Verify default.json was created
		defaultFile := filepath.Join(tempDir, "default.json")
		assert.FileExists(t, defaultFile)

		controller.Close()
	})

	t.Run("MissingDatabase", func(t *testing.T) {
		tempDir := createTempConfigDir(t)

		testLogger := logger.Logger{
			Prefix:    "test",
			LogFolder: "",
		}

		options := &Options{
			Logger:       testLogger,
			ConfigFolder: tempDir,
			GeoDB:        nil,
			Database:     nil, // Missing database
		}

		controller, err := NewAccessController(options)
		assert.Error(t, err)
		assert.Nil(t, controller)
		assert.Contains(t, err.Error(), "missing database access")
	})

	t.Run("CreateConfigFolder", func(t *testing.T) {
		tempDir := createTempConfigDir(t)
		configDir := filepath.Join(tempDir, "new_config_folder")

		// Create a temporary database file
		tempDB, err := os.CreateTemp("", "test_db_*.db")
		assert.NoError(t, err)
		tempDBPath := tempDB.Name()
		tempDB.Close()
		defer os.Remove(tempDBPath)

		testLogger := logger.Logger{
			Prefix:    "test",
			LogFolder: "",
		}

		testDB, err := database.NewDatabase(tempDBPath, dbinc.BackendBoltDB)
		assert.NoError(t, err)
		defer testDB.Close()

		options := &Options{
			Logger:       testLogger,
			ConfigFolder: configDir,
			GeoDB:        nil,
			Database:     testDB,
		}

		controller, err := NewAccessController(options)
		assert.NoError(t, err)
		assert.NotNil(t, controller)
		assert.DirExists(t, configDir)

		controller.Close()
	})

	t.Run("LoadExistingDefaultConfig", func(t *testing.T) {
		tempDir := createTempConfigDir(t)

		// Create a custom default config
		customDefault := AccessRule{
			ID:                   "default",
			Name:                 "CustomDefault",
			Desc:                 "Custom default rule",
			BlacklistEnabled:     true,
			WhitelistEnabled:     false,
			WhiteListCountryCode: &map[string]string{},
			WhiteListIP:          &map[string]string{},
			BlackListContryCode:  &map[string]string{},
			BlackListIP:          &map[string]string{},
		}

		defaultFile := filepath.Join(tempDir, "default.json")
		js, _ := json.MarshalIndent(customDefault, "", " ")
		err := os.WriteFile(defaultFile, js, 0775)
		assert.NoError(t, err)

		// Create a temporary database file
		tempDB, err := os.CreateTemp("", "test_db_*.db")
		assert.NoError(t, err)
		tempDBPath := tempDB.Name()
		tempDB.Close()
		defer os.Remove(tempDBPath)

		testLogger := logger.Logger{
			Prefix:    "test",
			LogFolder: "",
		}

		testDB, err := database.NewDatabase(tempDBPath, dbinc.BackendBoltDB)
		assert.NoError(t, err)
		defer testDB.Close()

		options := &Options{
			Logger:       testLogger,
			ConfigFolder: tempDir,
			GeoDB:        nil,
			Database:     testDB,
		}

		controller, err := NewAccessController(options)
		assert.NoError(t, err)
		assert.NotNil(t, controller)
		assert.Equal(t, "CustomDefault", controller.DefaultAccessRule.Name)
		assert.True(t, controller.DefaultAccessRule.BlacklistEnabled)

		controller.Close()
	})

	t.Run("LoadExistingAccessRules", func(t *testing.T) {
		tempDir := createTempConfigDir(t)

		// Create a custom access rule
		customRule := AccessRule{
			ID:                   "custom1",
			Name:                 "Custom Rule 1",
			Desc:                 "Test custom rule",
			BlacklistEnabled:     false,
			WhitelistEnabled:     true,
			WhiteListCountryCode: &map[string]string{"us": "United States"},
			WhiteListIP:          &map[string]string{"192.168.1.1": "Test IP"},
			BlackListContryCode:  &map[string]string{},
			BlackListIP:          &map[string]string{},
		}

		customFile := filepath.Join(tempDir, "custom1.json")
		js, _ := json.MarshalIndent(customRule, "", " ")
		err := os.WriteFile(customFile, js, 0775)
		assert.NoError(t, err)

		// Create a temporary database file
		tempDB, err := os.CreateTemp("", "test_db_*.db")
		assert.NoError(t, err)
		tempDBPath := tempDB.Name()
		tempDB.Close()
		defer os.Remove(tempDBPath)

		testLogger := logger.Logger{
			Prefix:    "test",
			LogFolder: "",
		}

		testDB, err := database.NewDatabase(tempDBPath, dbinc.BackendBoltDB)
		assert.NoError(t, err)
		defer testDB.Close()

		options := &Options{
			Logger:       testLogger,
			ConfigFolder: tempDir,
			GeoDB:        nil,
			Database:     testDB,
		}

		controller, err := NewAccessController(options)
		assert.NoError(t, err)
		assert.NotNil(t, controller)

		// Verify the custom rule was loaded
		loadedRule, err := controller.GetAccessRuleByID("custom1")
		assert.NoError(t, err)
		assert.NotNil(t, loadedRule)
		assert.Equal(t, "Custom Rule 1", loadedRule.Name)
		assert.True(t, loadedRule.WhitelistEnabled)

		controller.Close()
	})

	t.Run("DefaultPublicIpCheckInterval", func(t *testing.T) {
		tempDir := createTempConfigDir(t)

		// Create a temporary database file
		tempDB, err := os.CreateTemp("", "test_db_*.db")
		assert.NoError(t, err)
		tempDBPath := tempDB.Name()
		tempDB.Close()
		defer os.Remove(tempDBPath)

		testLogger := logger.Logger{
			Prefix:    "test",
			LogFolder: "",
		}

		testDB, err := database.NewDatabase(tempDBPath, dbinc.BackendBoltDB)
		assert.NoError(t, err)
		defer testDB.Close()

		options := &Options{
			Logger:                testLogger,
			ConfigFolder:          tempDir,
			GeoDB:                 nil,
			Database:              testDB,
			PublicIpCheckInterval: 0, // Should default to 12 hours
		}

		controller, err := NewAccessController(options)
		assert.NoError(t, err)
		assert.NotNil(t, controller)
		assert.Equal(t, int64(12*60*60), controller.Options.PublicIpCheckInterval)

		controller.Close()
	})
}

func TestGetGlobalAccessRule(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	rule, err := controller.GetGlobalAccessRule()
	assert.NoError(t, err)
	assert.NotNil(t, rule)
	assert.Equal(t, "default", rule.ID)
	assert.Equal(t, "Default", rule.Name)
}

func TestGetAccessRuleByID(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	t.Run("GetDefault", func(t *testing.T) {
		rule, err := controller.GetAccessRuleByID("default")
		assert.NoError(t, err)
		assert.NotNil(t, rule)
		assert.Equal(t, "default", rule.ID)
	})

	t.Run("GetDefaultWithEmptyID", func(t *testing.T) {
		rule, err := controller.GetAccessRuleByID("")
		assert.NoError(t, err)
		assert.NotNil(t, rule)
		assert.Equal(t, "default", rule.ID)
	})

	t.Run("GetNonExistentRule", func(t *testing.T) {
		rule, err := controller.GetAccessRuleByID("nonexistent")
		assert.Error(t, err)
		assert.Nil(t, rule)
		assert.Contains(t, err.Error(), "target access rule not exists")
	})

	t.Run("GetCustomRule", func(t *testing.T) {
		// Add a custom rule first
		newRule := &AccessRule{
			ID:                   "test1",
			Name:                 "Test Rule",
			Desc:                 "Test Description",
			BlacklistEnabled:     false,
			WhitelistEnabled:     false,
			WhiteListCountryCode: &map[string]string{},
			WhiteListIP:          &map[string]string{},
			BlackListContryCode:  &map[string]string{},
			BlackListIP:          &map[string]string{},
		}

		err := controller.AddNewAccessRule(newRule)
		assert.NoError(t, err)

		// Now retrieve it
		rule, err := controller.GetAccessRuleByID("test1")
		assert.NoError(t, err)
		assert.NotNil(t, rule)
		assert.Equal(t, "Test Rule", rule.Name)
	})
}

func TestListAllAccessRules(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	t.Run("OnlyDefault", func(t *testing.T) {
		rules := controller.ListAllAccessRules()
		assert.Len(t, rules, 1)
		assert.Equal(t, "default", rules[0].ID)
	})

	t.Run("WithMultipleRules", func(t *testing.T) {
		// Add multiple rules
		for i := 1; i <= 3; i++ {
			newRule := &AccessRule{
				ID:                   "test" + string(rune('0'+i)),
				Name:                 "Test Rule " + string(rune('0'+i)),
				Desc:                 "Description " + string(rune('0'+i)),
				BlacklistEnabled:     false,
				WhitelistEnabled:     false,
				WhiteListCountryCode: &map[string]string{},
				WhiteListIP:          &map[string]string{},
				BlackListContryCode:  &map[string]string{},
				BlackListIP:          &map[string]string{},
			}
			err := controller.AddNewAccessRule(newRule)
			assert.NoError(t, err)
		}

		rules := controller.ListAllAccessRules()
		assert.Len(t, rules, 4) // 1 default + 3 custom

		// Verify default is in the list
		foundDefault := false
		for _, rule := range rules {
			if rule.ID == "default" {
				foundDefault = true
				break
			}
		}
		assert.True(t, foundDefault)
	})
}

func TestAccessRuleExists(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	t.Run("DefaultExists", func(t *testing.T) {
		exists := controller.AccessRuleExists("default")
		assert.True(t, exists)
	})

	t.Run("NonExistentRule", func(t *testing.T) {
		exists := controller.AccessRuleExists("nonexistent")
		assert.False(t, exists)
	})

	t.Run("CustomRuleExists", func(t *testing.T) {
		newRule := &AccessRule{
			ID:                   "exists_test",
			Name:                 "Exists Test",
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

		exists := controller.AccessRuleExists("exists_test")
		assert.True(t, exists)
	})
}

func TestAddNewAccessRule(t *testing.T) {
	controller := createTestController(t); tempDir := controller.Options.ConfigFolder
	defer controller.Close()

	t.Run("Success", func(t *testing.T) {
		newRule := &AccessRule{
			ID:                   "new_rule",
			Name:                 "New Rule",
			Desc:                 "New rule description",
			BlacklistEnabled:     true,
			WhitelistEnabled:     false,
			WhiteListCountryCode: &map[string]string{},
			WhiteListIP:          &map[string]string{},
			BlackListContryCode:  &map[string]string{},
			BlackListIP:          &map[string]string{},
		}

		err := controller.AddNewAccessRule(newRule)
		assert.NoError(t, err)

		// Verify rule was added
		rule, err := controller.GetAccessRuleByID("new_rule")
		assert.NoError(t, err)
		assert.NotNil(t, rule)
		assert.Equal(t, "New Rule", rule.Name)
		assert.True(t, rule.BlacklistEnabled)

		// Verify parent was set
		assert.NotNil(t, rule.parent)
		assert.Equal(t, controller, rule.parent)

		// Verify file was created
		ruleFile := filepath.Join(tempDir, "new_rule.json")
		assert.FileExists(t, ruleFile)
	})

	t.Run("DuplicateID", func(t *testing.T) {
		rule1 := &AccessRule{
			ID:                   "duplicate",
			Name:                 "First",
			Desc:                 "First rule",
			BlacklistEnabled:     false,
			WhitelistEnabled:     false,
			WhiteListCountryCode: &map[string]string{},
			WhiteListIP:          &map[string]string{},
			BlackListContryCode:  &map[string]string{},
			BlackListIP:          &map[string]string{},
		}

		err := controller.AddNewAccessRule(rule1)
		assert.NoError(t, err)

		rule2 := &AccessRule{
			ID:                   "duplicate",
			Name:                 "Second",
			Desc:                 "Second rule",
			BlacklistEnabled:     false,
			WhitelistEnabled:     false,
			WhiteListCountryCode: &map[string]string{},
			WhiteListIP:          &map[string]string{},
			BlackListContryCode:  &map[string]string{},
			BlackListIP:          &map[string]string{},
		}

		err = controller.AddNewAccessRule(rule2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access rule already exists")
	})

	t.Run("NilMapsAreInitialized", func(t *testing.T) {
		newRule := &AccessRule{
			ID:   "nil_maps_test",
			Name: "Nil Maps Test",
			Desc: "Testing nil map initialization",
			// All maps are nil
		}

		err := controller.AddNewAccessRule(newRule)
		assert.NoError(t, err)

		rule, err := controller.GetAccessRuleByID("nil_maps_test")
		assert.NoError(t, err)
		assert.NotNil(t, rule.BlackListContryCode)
		assert.NotNil(t, rule.BlackListIP)
		assert.NotNil(t, rule.WhiteListCountryCode)
		assert.NotNil(t, rule.WhiteListIP)
	})
}

func TestUpdateAccessRule(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	t.Run("UpdateCustomRule", func(t *testing.T) {
		// Add a rule first
		newRule := &AccessRule{
			ID:                   "update_test",
			Name:                 "Original Name",
			Desc:                 "Original Description",
			BlacklistEnabled:     false,
			WhitelistEnabled:     false,
			WhiteListCountryCode: &map[string]string{},
			WhiteListIP:          &map[string]string{},
			BlackListContryCode:  &map[string]string{},
			BlackListIP:          &map[string]string{},
		}
		err := controller.AddNewAccessRule(newRule)
		assert.NoError(t, err)

		// Update the rule
		err = controller.UpdateAccessRule("update_test", "Updated Name", "Updated Description")
		assert.NoError(t, err)

		// Verify the update
		rule, err := controller.GetAccessRuleByID("update_test")
		assert.NoError(t, err)
		assert.Equal(t, "Updated Name", rule.Name)
		assert.Equal(t, "Updated Description", rule.Desc)
	})

	t.Run("UpdateDefaultRule", func(t *testing.T) {
		err := controller.UpdateAccessRule("default", "New Default Name", "New Default Description")
		assert.NoError(t, err)

		rule, err := controller.GetGlobalAccessRule()
		assert.NoError(t, err)
		assert.Equal(t, "New Default Name", rule.Name)
		assert.Equal(t, "New Default Description", rule.Desc)
	})

	t.Run("UpdateNonExistentRule", func(t *testing.T) {
		err := controller.UpdateAccessRule("nonexistent", "Name", "Desc")
		assert.Error(t, err)
	})
}

func TestRemoveAccessRuleByID(t *testing.T) {
	controller := createTestController(t); tempDir := controller.Options.ConfigFolder
	defer controller.Close()

	t.Run("RemoveCustomRule", func(t *testing.T) {
		// Add a rule first
		newRule := &AccessRule{
			ID:                   "remove_test",
			Name:                 "Remove Test",
			Desc:                 "Will be removed",
			BlacklistEnabled:     false,
			WhitelistEnabled:     false,
			WhiteListCountryCode: &map[string]string{},
			WhiteListIP:          &map[string]string{},
			BlackListContryCode:  &map[string]string{},
			BlackListIP:          &map[string]string{},
		}
		err := controller.AddNewAccessRule(newRule)
		assert.NoError(t, err)

		// Verify it exists
		exists := controller.AccessRuleExists("remove_test")
		assert.True(t, exists)

		// Remove it
		err = controller.RemoveAccessRuleByID("remove_test")
		assert.NoError(t, err)

		// Verify it's gone
		exists = controller.AccessRuleExists("remove_test")
		assert.False(t, exists)

		// Verify file is deleted
		ruleFile := filepath.Join(tempDir, "remove_test.json")
		assert.NoFileExists(t, ruleFile)
	})

	t.Run("CannotRemoveDefault", func(t *testing.T) {
		err := controller.RemoveAccessRuleByID("default")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "default access rule cannot be removed")
	})

	t.Run("RemoveNonExistentRule", func(t *testing.T) {
		err := controller.RemoveAccessRuleByID("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access rule not exists")
	})
}

func TestDeleteAccessRuleByID(t *testing.T) {
	controller := createTestController(t); tempDir := controller.Options.ConfigFolder
	defer controller.Close()

	t.Run("DeleteCustomRule", func(t *testing.T) {
		// Add a rule first
		newRule := &AccessRule{
			ID:                   "delete_test",
			Name:                 "Delete Test",
			Desc:                 "Will be deleted",
			BlacklistEnabled:     false,
			WhitelistEnabled:     false,
			WhiteListCountryCode: &map[string]string{},
			WhiteListIP:          &map[string]string{},
			BlackListContryCode:  &map[string]string{},
			BlackListIP:          &map[string]string{},
		}
		err := controller.AddNewAccessRule(newRule)
		assert.NoError(t, err)

		// Delete it
		err = controller.DeleteAccessRuleByID("delete_test")
		assert.NoError(t, err)

		// Verify it's gone from runtime
		exists := controller.AccessRuleExists("delete_test")
		assert.False(t, exists)

		// Verify file is deleted
		ruleFile := filepath.Join(tempDir, "delete_test.json")
		assert.NoFileExists(t, ruleFile)
	})

	t.Run("DeleteNonExistentRule", func(t *testing.T) {
		err := controller.DeleteAccessRuleByID("nonexistent")
		assert.Error(t, err)
	})
}

func TestClose(t *testing.T) {
	controller := createTestController(t)

	// Close should stop the public IP updater
	controller.Close()

	// Verify ticker was stopped
	assert.Nil(t, controller.publicIpTicker)
	assert.Nil(t, controller.publicIpTickerStop)
}

func TestDeepCopy(t *testing.T) {
	t.Run("CopyMap", func(t *testing.T) {
		original := map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}

		copied := deepCopy(original)

		// Verify values are equal
		assert.Equal(t, original, copied)

		// Modify the copy
		copied["key1"] = "modified"
		copied["key4"] = "new"

		// Verify original is unchanged
		assert.Equal(t, "value1", original["key1"])
		assert.NotContains(t, original, "key4")
	})

	t.Run("EmptyMap", func(t *testing.T) {
		original := map[string]string{}
		copied := deepCopy(original)
		assert.Empty(t, copied)
	})
}

func TestConcurrentAccess(t *testing.T) {
	controller := createTestController(t)
	defer controller.Close()

	// Test concurrent read/write operations
	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrent additions
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			newRule := &AccessRule{
				ID:                   "concurrent_" + string(rune('0'+id)),
				Name:                 "Concurrent Rule",
				Desc:                 "Test",
				BlacklistEnabled:     false,
				WhitelistEnabled:     false,
				WhiteListCountryCode: &map[string]string{},
				WhiteListIP:          &map[string]string{},
				BlackListContryCode:  &map[string]string{},
				BlackListIP:          &map[string]string{},
			}
			controller.AddNewAccessRule(newRule)
		}(i)
	}

	wg.Wait()

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			controller.ListAllAccessRules()
			controller.AccessRuleExists("default")
			controller.GetGlobalAccessRule()
		}()
	}

	wg.Wait()
}
