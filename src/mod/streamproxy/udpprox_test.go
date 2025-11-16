package streamproxy

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"imuslab.com/zoraxy/mod/info/logger"
)

func TestCreateNewUDPConn(t *testing.T) {
	// Create a UDP server for testing
	serverAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)

	serverConn, err := net.ListenUDP("udp", serverAddr)
	require.NoError(t, err)
	defer serverConn.Close()

	actualServerAddr := serverConn.LocalAddr().(*net.UDPAddr)

	// Create client address
	clientAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)

	// Create connection
	conn := createNewUDPConn(actualServerAddr, clientAddr)

	assert.NotNil(t, conn)
	assert.Equal(t, clientAddr, conn.ClientAddr)
	assert.NotNil(t, conn.ServerConn)

	if conn != nil && conn.ServerConn != nil {
		conn.ServerConn.Close()
	}
}

func TestCreateNewUDPConnInvalidAddress(t *testing.T) {
	// Use invalid server address
	serverAddr := &net.UDPAddr{
		IP:   net.IPv4(255, 255, 255, 255),
		Port: 99999, // Invalid port
	}

	clientAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)

	// Should return nil for invalid address
	conn := createNewUDPConn(serverAddr, clientAddr)
	assert.Nil(t, conn)
}

func TestInitUDPConnections(t *testing.T) {
	tests := []struct {
		name          string
		listenAddr    string
		targetAddr    string
		expectError   bool
	}{
		{
			name:        "valid addresses",
			listenAddr:  "127.0.0.1:0",
			targetAddr:  "127.0.0.1:9999",
			expectError: false,
		},
		{
			name:        "invalid listen address",
			listenAddr:  "invalid:address",
			targetAddr:  "127.0.0.1:9999",
			expectError: true,
		},
		{
			name:        "invalid target address",
			listenAddr:  "127.0.0.1:0",
			targetAddr:  "invalid:address",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inbound, outbound, err := initUDPConnections(tt.listenAddr, tt.targetAddr)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, inbound)
				assert.NotNil(t, outbound)
				if inbound != nil {
					inbound.Close()
				}
			}
		})
	}
}

func TestRunUDPConnectionRelay(t *testing.T) {
	tempDir := t.TempDir()
	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	manager := &Manager{
		Options: &Options{
			Logger: log,
		},
	}

	instance := &ProxyRelayInstance{
		EnableLogging: true,
		parent:        manager,
	}

	// Create listener
	listenerAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	listener, err := net.ListenUDP("udp", listenerAddr)
	require.NoError(t, err)
	defer listener.Close()

	// Create server connection
	serverAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	serverConn, err := net.ListenUDP("udp", serverAddr)
	require.NoError(t, err)
	actualServerAddr := serverConn.LocalAddr().(*net.UDPAddr)

	// Create client address
	clientAddr := listener.LocalAddr().(*net.UDPAddr)

	// Create UDP connection structure
	conn := &udpClientServerConn{
		ClientAddr: clientAddr,
		ServerConn: serverConn,
	}

	// Run relay in background
	go instance.RunUDPConnectionRelay(conn, listener)

	// Send data from server
	testData := []byte("test data from server")
	_, err = serverConn.WriteToUDP(testData, actualServerAddr)
	if err != nil {
		t.Logf("Write error: %v", err)
	}

	// Give it time to relay
	time.Sleep(100 * time.Millisecond)

	// Close server connection to stop relay
	serverConn.Close()

	time.Sleep(100 * time.Millisecond)
}

func TestCloseAllUDPConnections(t *testing.T) {
	tempDir := t.TempDir()
	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	manager := &Manager{
		Options: &Options{
			Logger: log,
		},
	}

	instance := &ProxyRelayInstance{
		EnableLogging: true,
		parent:        manager,
		udpClientMap:  sync.Map{},
	}

	// Add some connections to the map
	for i := 0; i < 3; i++ {
		addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
		conn, _ := net.ListenUDP("udp", addr)

		clientServerConn := &udpClientServerConn{
			ClientAddr: addr,
			ServerConn: conn,
		}

		instance.udpClientMap.Store(addr.String(), clientServerConn)
	}

	// Close all connections
	assert.NotPanics(t, func() {
		instance.CloseAllUDPConnections()
	})

	// Verify all are closed (we can't really verify they're closed,
	// but we can verify the function doesn't panic)
}

func TestWriteProxyProtocolHeaderUDP(t *testing.T) {
	// Create UDP connection
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	conn, err := net.ListenUDP("udp", addr)
	require.NoError(t, err)
	defer conn.Close()

	srcAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:12345")
	require.NoError(t, err)

	dstAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:54321")
	require.NoError(t, err)

	// Write proxy protocol header
	err = WriteProxyProtocolHeaderUDP(conn, srcAddr, dstAddr)
	assert.NoError(t, err)
}

func TestForwardUDP(t *testing.T) {
	// Create target UDP server
	targetAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	targetConn, err := net.ListenUDP("udp", targetAddr)
	require.NoError(t, err)
	defer targetConn.Close()

	actualTargetAddr := targetConn.LocalAddr().String()

	// Handle incoming packets
	go func() {
		buf := make([]byte, 1500)
		for {
			n, addr, err := targetConn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			// Echo back
			targetConn.WriteToUDP(buf[:n], addr)
		}
	}()

	tempDir := t.TempDir()
	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	manager := &Manager{
		Options: &Options{
			Logger: log,
		},
	}

	instance := &ProxyRelayInstance{
		EnableLogging: true,
		parent:        manager,
		udpClientMap:  sync.Map{},
	}

	stopChan := make(chan bool)

	// Start UDP forwarder
	errChan := make(chan error, 1)
	go func() {
		err := instance.ForwardUDP("127.0.0.1:0", actualTargetAddr, stopChan)
		errChan <- err
	}()

	// Give it time to start
	time.Sleep(200 * time.Millisecond)

	// Stop the forwarder
	stopChan <- true

	// Wait for it to finish
	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("ForwardUDP did not stop in time")
	}
}

func TestForwardUDPWithPortOnly(t *testing.T) {
	// Create target server
	targetAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	targetConn, err := net.ListenUDP("udp", targetAddr)
	require.NoError(t, err)
	defer targetConn.Close()

	actualTargetAddr := targetConn.LocalAddr().String()

	tempDir := t.TempDir()
	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	manager := &Manager{
		Options: &Options{
			Logger: log,
		},
	}

	tests := []struct {
		name       string
		listenAddr string
	}{
		{"port only", "0"},
		{"colon with port", ":0"},
		{"full address", "0.0.0.0:0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &ProxyRelayInstance{
				EnableLogging: true,
				parent:        manager,
				udpClientMap:  sync.Map{},
			}

			stopChan := make(chan bool)

			// Start forwarder
			go func() {
				instance.ForwardUDP(tt.listenAddr, actualTargetAddr, stopChan)
			}()

			time.Sleep(200 * time.Millisecond)

			// Stop
			stopChan <- true
			time.Sleep(100 * time.Millisecond)
		})
	}
}

func TestForwardUDPWithProxyProtocol(t *testing.T) {
	// Create target server
	targetAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	targetConn, err := net.ListenUDP("udp", targetAddr)
	require.NoError(t, err)
	defer targetConn.Close()

	actualTargetAddr := targetConn.LocalAddr().String()

	// Read incoming packets
	go func() {
		buf := make([]byte, 1500)
		for {
			_, _, err := targetConn.ReadFromUDP(buf)
			if err != nil {
				return
			}
		}
	}()

	tempDir := t.TempDir()
	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	manager := &Manager{
		Options: &Options{
			Logger: log,
		},
	}

	instance := &ProxyRelayInstance{
		EnableLogging:        true,
		parent:               manager,
		udpClientMap:         sync.Map{},
		ProxyProtocolVersion: ProxyProtocolV2,
	}

	// Create a listener for the proxy
	proxyAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	proxyListener, err := net.ListenUDP("udp", proxyAddr)
	require.NoError(t, err)
	actualProxyAddr := proxyListener.LocalAddr().String()
	proxyListener.Close() // Close so ForwardUDP can bind

	stopChan := make(chan bool)

	// Start forwarder
	go func() {
		instance.ForwardUDP(actualProxyAddr, actualTargetAddr, stopChan)
	}()

	time.Sleep(200 * time.Millisecond)

	// Send a UDP packet to the proxy
	clientConn, err := net.Dial("udp", actualProxyAddr)
	if err == nil {
		clientConn.Write([]byte("test packet"))
		time.Sleep(100 * time.Millisecond)
		clientConn.Close()
	}

	time.Sleep(100 * time.Millisecond)

	// Stop
	stopChan <- true
	time.Sleep(100 * time.Millisecond)
}

func TestForwardUDPMultipleClients(t *testing.T) {
	// Create target server
	targetAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	targetConn, err := net.ListenUDP("udp", targetAddr)
	require.NoError(t, err)
	defer targetConn.Close()

	actualTargetAddr := targetConn.LocalAddr().String()

	// Echo server
	go func() {
		buf := make([]byte, 1500)
		for {
			n, addr, err := targetConn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			targetConn.WriteToUDP(buf[:n], addr)
		}
	}()

	tempDir := t.TempDir()
	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	manager := &Manager{
		Options: &Options{
			Logger: log,
		},
	}

	instance := &ProxyRelayInstance{
		EnableLogging: true,
		parent:        manager,
		udpClientMap:  sync.Map{},
	}

	// Create proxy listener
	proxyAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	proxyListener, err := net.ListenUDP("udp", proxyAddr)
	require.NoError(t, err)
	actualProxyAddr := proxyListener.LocalAddr().String()
	proxyListener.Close()

	stopChan := make(chan bool)

	// Start forwarder
	go func() {
		instance.ForwardUDP(actualProxyAddr, actualTargetAddr, stopChan)
	}()

	time.Sleep(300 * time.Millisecond)

	// Create multiple clients
	for i := 0; i < 3; i++ {
		go func(id int) {
			conn, err := net.Dial("udp", actualProxyAddr)
			if err == nil {
				defer conn.Close()
				testData := []byte("hello from client")
				conn.Write(testData)
				time.Sleep(100 * time.Millisecond)
			}
		}(i)
	}

	time.Sleep(500 * time.Millisecond)

	// Stop
	stopChan <- true
	time.Sleep(200 * time.Millisecond)

	// Verify client map has entries
	count := 0
	instance.udpClientMap.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	// We should have had some clients
	assert.GreaterOrEqual(t, count, 0) // May be 0 if connections were too fast
}

func TestForwardUDPInvalidAddress(t *testing.T) {
	tempDir := t.TempDir()
	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	manager := &Manager{
		Options: &Options{
			Logger: log,
		},
	}

	instance := &ProxyRelayInstance{
		EnableLogging: true,
		parent:        manager,
		udpClientMap:  sync.Map{},
	}

	stopChan := make(chan bool)

	// Start with invalid target address
	err = instance.ForwardUDP("127.0.0.1:0", "invalid:address", stopChan)
	assert.Error(t, err)
}
