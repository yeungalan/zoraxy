package streamproxy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"imuslab.com/zoraxy/mod/info/logger"
)

func setupTestManager(t *testing.T) (*Manager, string) {
	// Create temporary directory for test configs
	tempDir := t.TempDir()

	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	opts := &Options{
		DefaultTimeout:       30,
		AccessControlHandler: nil, // Will be set to default
		ConfigStore:          tempDir,
		Logger:               log,
	}

	manager, err := NewStreamProxy(opts)
	require.NoError(t, err)
	require.NotNil(t, manager)

	return manager, tempDir
}

func TestNewStreamProxy(t *testing.T) {
	tests := []struct {
		name        string
		setupOpts   func(t *testing.T) *Options
		expectError bool
	}{
		{
			name: "valid options with new directory",
			setupOpts: func(t *testing.T) *Options {
				tempDir := t.TempDir()
				log, _ := logger.NewLogger("test", tempDir)
				return &Options{
					DefaultTimeout: 30,
					ConfigStore:    filepath.Join(tempDir, "newdir"),
					Logger:         log,
				}
			},
			expectError: false,
		},
		{
			name: "valid options with existing directory",
			setupOpts: func(t *testing.T) *Options {
				tempDir := t.TempDir()
				log, _ := logger.NewLogger("test", tempDir)
				return &Options{
					DefaultTimeout: 30,
					ConfigStore:    tempDir,
					Logger:         log,
				}
			},
			expectError: false,
		},
		{
			name: "nil access control handler should use default",
			setupOpts: func(t *testing.T) *Options {
				tempDir := t.TempDir()
				log, _ := logger.NewLogger("test", tempDir)
				return &Options{
					DefaultTimeout:       30,
					AccessControlHandler: nil,
					ConfigStore:          tempDir,
					Logger:               log,
				}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.setupOpts(t)
			manager, err := NewStreamProxy(opts)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, manager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
				assert.NotNil(t, manager.Options)
				assert.NotNil(t, manager.Options.AccessControlHandler)
			}
		})
	}
}

func TestNewConfig(t *testing.T) {
	manager, _ := setupTestManager(t)

	tests := []struct {
		name          string
		options       *ProxyRelayOptions
		validateFunc  func(t *testing.T, uuid string, m *Manager)
	}{
		{
			name: "create TCP config",
			options: &ProxyRelayOptions{
				Name:                 "test-tcp",
				ListeningAddr:        "127.0.0.1:9001",
				ProxyAddr:            "127.0.0.1:9002",
				Timeout:              30,
				UseTCP:               true,
				UseUDP:               false,
				ProxyProtocolVersion: ProxyProtocolDisabled,
				EnableLogging:        true,
			},
			validateFunc: func(t *testing.T, uuid string, m *Manager) {
				assert.NotEmpty(t, uuid)
				cfg, err := m.GetConfigByUUID(uuid)
				require.NoError(t, err)
				assert.Equal(t, "test-tcp", cfg.Name)
				assert.True(t, cfg.UseTCP)
				assert.False(t, cfg.UseUDP)
			},
		},
		{
			name: "create UDP config",
			options: &ProxyRelayOptions{
				Name:                 "test-udp",
				ListeningAddr:        "127.0.0.1:9003",
				ProxyAddr:            "127.0.0.1:9004",
				Timeout:              30,
				UseTCP:               false,
				UseUDP:               true,
				ProxyProtocolVersion: ProxyProtocolDisabled,
				EnableLogging:        false,
			},
			validateFunc: func(t *testing.T, uuid string, m *Manager) {
				assert.NotEmpty(t, uuid)
				cfg, err := m.GetConfigByUUID(uuid)
				require.NoError(t, err)
				assert.Equal(t, "test-udp", cfg.Name)
				assert.False(t, cfg.UseTCP)
				assert.True(t, cfg.UseUDP)
				assert.False(t, cfg.EnableLogging)
			},
		},
		{
			name: "create config with proxy protocol v1",
			options: &ProxyRelayOptions{
				Name:                 "test-proxyv1",
				ListeningAddr:        "127.0.0.1:9005",
				ProxyAddr:            "127.0.0.1:9006",
				Timeout:              60,
				UseTCP:               true,
				UseUDP:               false,
				ProxyProtocolVersion: ProxyProtocolV1,
				EnableLogging:        true,
			},
			validateFunc: func(t *testing.T, uuid string, m *Manager) {
				cfg, err := m.GetConfigByUUID(uuid)
				require.NoError(t, err)
				assert.Equal(t, ProxyProtocolV1, cfg.ProxyProtocolVersion)
			},
		},
		{
			name: "create config with proxy protocol v2",
			options: &ProxyRelayOptions{
				Name:                 "test-proxyv2",
				ListeningAddr:        "127.0.0.1:9007",
				ProxyAddr:            "127.0.0.1:9008",
				Timeout:              60,
				UseTCP:               true,
				UseUDP:               true,
				ProxyProtocolVersion: ProxyProtocolV2,
				EnableLogging:        true,
			},
			validateFunc: func(t *testing.T, uuid string, m *Manager) {
				cfg, err := m.GetConfigByUUID(uuid)
				require.NoError(t, err)
				assert.Equal(t, ProxyProtocolV2, cfg.ProxyProtocolVersion)
				assert.True(t, cfg.UseTCP)
				assert.True(t, cfg.UseUDP)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uuid := manager.NewConfig(tt.options)
			tt.validateFunc(t, uuid, manager)
		})
	}
}

func TestGetConfigByUUID(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Create a test config
	uuid := manager.NewConfig(&ProxyRelayOptions{
		Name:          "test-get",
		ListeningAddr: "127.0.0.1:9009",
		ProxyAddr:     "127.0.0.1:9010",
		Timeout:       30,
		UseTCP:        true,
	})

	tests := []struct {
		name        string
		uuid        string
		expectError bool
	}{
		{
			name:        "get existing config",
			uuid:        uuid,
			expectError: false,
		},
		{
			name:        "get non-existent config",
			uuid:        "non-existent-uuid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := manager.GetConfigByUUID(tt.uuid)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
				assert.Equal(t, tt.uuid, cfg.UUID)
			}
		})
	}
}

func TestEditConfig(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Create initial config
	uuid := manager.NewConfig(&ProxyRelayOptions{
		Name:          "original",
		ListeningAddr: "127.0.0.1:9011",
		ProxyAddr:     "127.0.0.1:9012",
		Timeout:       30,
		UseTCP:        true,
		UseUDP:        false,
	})

	tests := []struct {
		name        string
		updateConfig *ProxyRuleUpdateConfig
		expectError bool
		validateFunc func(t *testing.T, m *Manager)
	}{
		{
			name: "update name only",
			updateConfig: &ProxyRuleUpdateConfig{
				InstanceUUID: uuid,
				NewName:      "updated-name",
				UseTCP:       true,
			},
			expectError: false,
			validateFunc: func(t *testing.T, m *Manager) {
				cfg, _ := m.GetConfigByUUID(uuid)
				assert.Equal(t, "updated-name", cfg.Name)
			},
		},
		{
			name: "update listening address",
			updateConfig: &ProxyRuleUpdateConfig{
				InstanceUUID:     uuid,
				NewListeningAddr: "127.0.0.1:9999",
				UseTCP:           true,
			},
			expectError: false,
			validateFunc: func(t *testing.T, m *Manager) {
				cfg, _ := m.GetConfigByUUID(uuid)
				assert.Equal(t, "127.0.0.1:9999", cfg.ListeningAddress)
			},
		},
		{
			name: "update proxy address",
			updateConfig: &ProxyRuleUpdateConfig{
				InstanceUUID: uuid,
				NewProxyAddr: "127.0.0.1:8888",
				UseTCP:       true,
			},
			expectError: false,
			validateFunc: func(t *testing.T, m *Manager) {
				cfg, _ := m.GetConfigByUUID(uuid)
				assert.Equal(t, "127.0.0.1:8888", cfg.ProxyTargetAddr)
			},
		},
		{
			name: "enable UDP",
			updateConfig: &ProxyRuleUpdateConfig{
				InstanceUUID: uuid,
				UseTCP:       true,
				UseUDP:       true,
			},
			expectError: false,
			validateFunc: func(t *testing.T, m *Manager) {
				cfg, _ := m.GetConfigByUUID(uuid)
				assert.True(t, cfg.UseUDP)
			},
		},
		{
			name: "update timeout",
			updateConfig: &ProxyRuleUpdateConfig{
				InstanceUUID: uuid,
				NewTimeout:   120,
				UseTCP:       true,
			},
			expectError: false,
			validateFunc: func(t *testing.T, m *Manager) {
				cfg, _ := m.GetConfigByUUID(uuid)
				assert.Equal(t, 120, cfg.Timeout)
			},
		},
		{
			name: "invalid timeout should error",
			updateConfig: &ProxyRuleUpdateConfig{
				InstanceUUID: uuid,
				NewTimeout:   -10,
				UseTCP:       true,
			},
			expectError: true,
			validateFunc: nil,
		},
		{
			name: "update proxy protocol version",
			updateConfig: &ProxyRuleUpdateConfig{
				InstanceUUID:         uuid,
				ProxyProtocolVersion: 2,
				UseTCP:               true,
			},
			expectError: false,
			validateFunc: func(t *testing.T, m *Manager) {
				cfg, _ := m.GetConfigByUUID(uuid)
				assert.Equal(t, ProxyProtocolV2, cfg.ProxyProtocolVersion)
			},
		},
		{
			name: "non-existent uuid should error",
			updateConfig: &ProxyRuleUpdateConfig{
				InstanceUUID: "non-existent",
				NewName:      "test",
			},
			expectError: true,
			validateFunc: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.EditConfig(tt.updateConfig)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validateFunc != nil {
					tt.validateFunc(t, manager)
				}
			}
		})
	}
}

func TestRemoveConfig(t *testing.T) {
	manager, _ := setupTestManager(t)

	// Create a config to remove
	uuid := manager.NewConfig(&ProxyRelayOptions{
		Name:          "to-be-removed",
		ListeningAddr: "127.0.0.1:9013",
		ProxyAddr:     "127.0.0.1:9014",
		Timeout:       30,
		UseTCP:        true,
	})

	// Verify it exists
	_, err := manager.GetConfigByUUID(uuid)
	require.NoError(t, err)

	// Remove it
	err = manager.RemoveConfig(uuid)
	assert.NoError(t, err)

	// Verify it's gone
	_, err = manager.GetConfigByUUID(uuid)
	assert.Error(t, err)

	// Try to remove non-existent config
	err = manager.RemoveConfig("non-existent")
	assert.Error(t, err)
}

func TestSaveConfigToDatabase(t *testing.T) {
	manager, tempDir := setupTestManager(t)

	// Create a config
	uuid := manager.NewConfig(&ProxyRelayOptions{
		Name:          "save-test",
		ListeningAddr: "127.0.0.1:9015",
		ProxyAddr:     "127.0.0.1:9016",
		Timeout:       30,
		UseTCP:        true,
	})

	// Save to database
	manager.SaveConfigToDatabase()

	// Verify config file exists
	configFile := filepath.Join(tempDir, uuid+".config")
	assert.FileExists(t, configFile)

	// Verify config file contains valid JSON
	data, err := os.ReadFile(configFile)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestConvertProxyProtocolVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected ProxyProtocolVersion
	}{
		{"disabled", 0, ProxyProtocolDisabled},
		{"version 1", 1, ProxyProtocolV1},
		{"version 2", 2, ProxyProtocolV2},
		{"invalid", 99, ProxyProtocolDisabled},
		{"negative", -1, ProxyProtocolDisabled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertIntToProxyProtocolVersion(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertProxyProtocolVersionToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    ProxyProtocolVersion
		expected int
	}{
		{"disabled", ProxyProtocolDisabled, 0},
		{"version 1", ProxyProtocolV1, 1},
		{"version 2", ProxyProtocolV2, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertProxyProtocolVersionToInt(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestManagerLogf(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) *Manager
		message string
		err     error
	}{
		{
			name: "with logger",
			setup: func(t *testing.T) *Manager {
				m, _ := setupTestManager(t)
				return m
			},
			message: "test message",
			err:     nil,
		},
		{
			name: "without logger",
			setup: func(t *testing.T) *Manager {
				tempDir := t.TempDir()
				opts := &Options{
					ConfigStore: tempDir,
					Logger:      nil,
				}
				m, _ := NewStreamProxy(opts)
				return m
			},
			message: "test message without logger",
			err:     nil,
		},
		{
			name: "with error",
			setup: func(t *testing.T) *Manager {
				m, _ := setupTestManager(t)
				return m
			},
			message: "error message",
			err:     assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup(t)
			// Should not panic
			assert.NotPanics(t, func() {
				m.logf(tt.message, tt.err)
			})
		})
	}
}

func TestNewStreamProxyWithExistingConfigs(t *testing.T) {
	tempDir := t.TempDir()

	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	// Create a manager and add configs
	opts := &Options{
		DefaultTimeout: 30,
		ConfigStore:    tempDir,
		Logger:         log,
	}

	manager1, err := NewStreamProxy(opts)
	require.NoError(t, err)

	// Create some configs
	uuid1 := manager1.NewConfig(&ProxyRelayOptions{
		Name:          "persistent-1",
		ListeningAddr: "127.0.0.1:9017",
		ProxyAddr:     "127.0.0.1:9018",
		Timeout:       30,
		UseTCP:        true,
	})

	uuid2 := manager1.NewConfig(&ProxyRelayOptions{
		Name:          "persistent-2",
		ListeningAddr: "127.0.0.1:9019",
		ProxyAddr:     "127.0.0.1:9020",
		Timeout:       30,
		UseUDP:        true,
	})

	// Create a new manager with same config store
	manager2, err := NewStreamProxy(opts)
	require.NoError(t, err)

	// Verify configs were loaded
	assert.Len(t, manager2.Configs, 2)

	// Verify we can get the configs by UUID
	cfg1, err := manager2.GetConfigByUUID(uuid1)
	assert.NoError(t, err)
	assert.Equal(t, "persistent-1", cfg1.Name)

	cfg2, err := manager2.GetConfigByUUID(uuid2)
	assert.NoError(t, err)
	assert.Equal(t, "persistent-2", cfg2.Name)
}
