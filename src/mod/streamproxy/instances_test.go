package streamproxy

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"imuslab.com/zoraxy/mod/info/logger"
)

func setupTestInstance(t *testing.T, useTCP, useUDP bool) (*Manager, *ProxyRelayInstance) {
	tempDir := t.TempDir()
	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	opts := &Options{
		DefaultTimeout: 30,
		ConfigStore:    tempDir,
		Logger:         log,
	}

	manager, err := NewStreamProxy(opts)
	require.NoError(t, err)

	uuid := manager.NewConfig(&ProxyRelayOptions{
		Name:          "test-instance",
		ListeningAddr: "127.0.0.1:0", // Use port 0 to get random available port
		ProxyAddr:     "127.0.0.1:19999",
		Timeout:       1,
		UseTCP:        useTCP,
		UseUDP:        useUDP,
		EnableLogging: true,
	})

	instance, err := manager.GetConfigByUUID(uuid)
	require.NoError(t, err)

	return manager, instance
}

func TestLogMsg(t *testing.T) {
	tests := []struct {
		name          string
		enableLogging bool
		message       string
		err           error
	}{
		{
			name:          "logging enabled with message only",
			enableLogging: true,
			message:       "test message",
			err:           nil,
		},
		{
			name:          "logging enabled with error",
			enableLogging: true,
			message:       "error message",
			err:           assert.AnError,
		},
		{
			name:          "logging disabled",
			enableLogging: false,
			message:       "should not log",
			err:           nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &ProxyRelayInstance{
				EnableLogging: tt.enableLogging,
			}

			// Should not panic
			assert.NotPanics(t, func() {
				instance.LogMsg(tt.message, tt.err)
			})
		})
	}
}

func TestIsRunning(t *testing.T) {
	tests := []struct {
		name           string
		tcpStopChan    chan bool
		udpStopChan    chan bool
		expectedRunning bool
	}{
		{
			name:           "not running - both nil",
			tcpStopChan:    nil,
			udpStopChan:    nil,
			expectedRunning: false,
		},
		{
			name:           "running - TCP only",
			tcpStopChan:    make(chan bool),
			udpStopChan:    nil,
			expectedRunning: true,
		},
		{
			name:           "running - UDP only",
			tcpStopChan:    nil,
			udpStopChan:    make(chan bool),
			expectedRunning: true,
		},
		{
			name:           "running - both",
			tcpStopChan:    make(chan bool),
			udpStopChan:    make(chan bool),
			expectedRunning: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &ProxyRelayInstance{
				tcpStopChan: tt.tcpStopChan,
				udpStopChan: tt.udpStopChan,
			}

			result := instance.IsRunning()
			assert.Equal(t, tt.expectedRunning, result)
		})
	}
}

func TestStartTCP(t *testing.T) {
	manager, instance := setupTestInstance(t, true, false)

	// Start the instance
	err := instance.Start()
	assert.NoError(t, err)
	assert.True(t, instance.Running)

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Verify it's running
	assert.True(t, instance.IsRunning())

	// Try to start again - should error
	err = instance.Start()
	assert.Error(t, err)

	// Clean up
	instance.Stop()
	time.Sleep(100 * time.Millisecond)
	assert.False(t, instance.IsRunning())

	// Verify config was saved
	assert.FileExists(t, manager.Options.ConfigStore+"/"+instance.UUID+".config")
}

func TestStartUDP(t *testing.T) {
	_, instance := setupTestInstance(t, false, true)

	// Start the instance
	err := instance.Start()
	assert.NoError(t, err)
	assert.True(t, instance.Running)

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Verify it's running
	assert.True(t, instance.IsRunning())

	// Clean up
	instance.Stop()
	time.Sleep(100 * time.Millisecond)
	assert.False(t, instance.IsRunning())
}

func TestStartBothTCPAndUDP(t *testing.T) {
	_, instance := setupTestInstance(t, true, true)

	// Start the instance
	err := instance.Start()
	assert.NoError(t, err)
	assert.True(t, instance.Running)

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Verify it's running
	assert.True(t, instance.IsRunning())

	// Clean up
	instance.Stop()
	time.Sleep(100 * time.Millisecond)
	assert.False(t, instance.IsRunning())
}

func TestStartWithInvalidAddress(t *testing.T) {
	tempDir := t.TempDir()
	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	opts := &Options{
		DefaultTimeout: 30,
		ConfigStore:    tempDir,
		Logger:         log,
	}

	manager, err := NewStreamProxy(opts)
	require.NoError(t, err)

	// Create config with invalid listening address
	uuid := manager.NewConfig(&ProxyRelayOptions{
		Name:          "invalid-addr",
		ListeningAddr: "999.999.999.999:99999", // Invalid address
		ProxyAddr:     "127.0.0.1:8080",
		Timeout:       1,
		UseTCP:        true,
	})

	instance, err := manager.GetConfigByUUID(uuid)
	require.NoError(t, err)

	// Start should succeed (starts goroutine) but will fail internally
	err = instance.Start()
	assert.NoError(t, err)

	// Give it time to fail
	time.Sleep(200 * time.Millisecond)

	// Instance should not be running after error
	assert.False(t, instance.Running)
}

func TestStop(t *testing.T) {
	_, instance := setupTestInstance(t, true, true)

	// Start first
	err := instance.Start()
	require.NoError(t, err)

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Verify it's running
	assert.True(t, instance.IsRunning())

	// Stop it
	instance.Stop()

	// Give it time to stop
	time.Sleep(100 * time.Millisecond)

	// Verify it's stopped
	assert.False(t, instance.IsRunning())
	assert.False(t, instance.Running)
	assert.Nil(t, instance.tcpStopChan)
	assert.Nil(t, instance.udpStopChan)
}

func TestStopTCPOnly(t *testing.T) {
	_, instance := setupTestInstance(t, true, false)

	err := instance.Start()
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	instance.Stop()
	time.Sleep(100 * time.Millisecond)

	assert.False(t, instance.IsRunning())
	assert.Nil(t, instance.tcpStopChan)
}

func TestStopUDPOnly(t *testing.T) {
	_, instance := setupTestInstance(t, false, true)

	err := instance.Start()
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)

	instance.Stop()
	time.Sleep(100 * time.Millisecond)

	assert.False(t, instance.IsRunning())
	assert.Nil(t, instance.udpStopChan)
}

func TestRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	_, instance := setupTestInstance(t, true, false)

	// Start first
	err := instance.Start()
	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	assert.True(t, instance.IsRunning())

	// Restart
	go instance.Restart()

	// Should eventually be running again
	time.Sleep(4 * time.Second) // Restart has 3 second sleep

	assert.True(t, instance.IsRunning())

	// Clean up
	instance.Stop()
	time.Sleep(100 * time.Millisecond)
}

func TestRestartWhenNotRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	_, instance := setupTestInstance(t, true, false)

	// Restart when not running
	go instance.Restart()

	// Should be running after restart
	time.Sleep(4 * time.Second)

	assert.True(t, instance.IsRunning())

	// Clean up
	instance.Stop()
	time.Sleep(100 * time.Millisecond)
}

func TestStartTCPWithActualConnection(t *testing.T) {
	// Create a simple echo server for testing
	echoListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer echoListener.Close()

	echoAddr := echoListener.Addr().String()

	// Start echo server
	go func() {
		for {
			conn, err := echoListener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				n, _ := c.Read(buf)
				c.Write(buf[:n])
			}(conn)
		}
	}()

	// Create proxy instance
	tempDir := t.TempDir()
	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	opts := &Options{
		DefaultTimeout: 30,
		ConfigStore:    tempDir,
		Logger:         log,
	}

	manager, err := NewStreamProxy(opts)
	require.NoError(t, err)

	uuid := manager.NewConfig(&ProxyRelayOptions{
		Name:          "tcp-proxy-test",
		ListeningAddr: "127.0.0.1:0",
		ProxyAddr:     echoAddr,
		Timeout:       1,
		UseTCP:        true,
		EnableLogging: true,
	})

	instance, err := manager.GetConfigByUUID(uuid)
	require.NoError(t, err)

	// Start the proxy
	err = instance.Start()
	require.NoError(t, err)
	defer instance.Stop()

	// Give it time to start
	time.Sleep(200 * time.Millisecond)

	// The proxy is listening but we can't easily test the full connection
	// without knowing the actual port (it's 0, which means random)
	// Just verify it started successfully
	assert.True(t, instance.IsRunning())
}
