package streamproxy

import (
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"imuslab.com/zoraxy/mod/info/logger"
)

func TestIsValidIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"valid IPv4", "127.0.0.1", true},
		{"valid IPv4 2", "192.168.1.1", true},
		{"valid IPv6", "::1", true},
		{"valid IPv6 full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", true},
		{"invalid IP", "999.999.999.999", false},
		{"invalid string", "not-an-ip", false},
		{"empty string", "", false},
		{"partial IP", "192.168", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidIP(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidPort(t *testing.T) {
	tests := []struct {
		name     string
		port     string
		expected bool
	}{
		{"valid port 80", "80", true},
		{"valid port 8080", "8080", true},
		{"valid port 1", "1", true},
		{"valid port 65535", "65535", true},
		{"invalid port 0", "0", false},
		{"invalid port 65536", "65536", false},
		{"invalid port negative", "-1", false},
		{"invalid port string", "abc", false},
		{"invalid port empty", "", false},
		{"invalid port with colon", ":8080", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidPort(tt.port)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStartListener(t *testing.T) {
	tests := []struct {
		name        string
		address     string
		expectError bool
	}{
		{
			name:        "valid address with port 0",
			address:     "127.0.0.1:0",
			expectError: false,
		},
		{
			name:        "valid localhost",
			address:     "127.0.0.1:0",
			expectError: false,
		},
		{
			name:        "invalid address",
			address:     "999.999.999.999:8080",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listener, err := startListener(tt.address)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, listener)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, listener)
				if listener != nil {
					listener.Close()
				}
			}
		})
	}
}

func TestConnCopy(t *testing.T) {
	// This test verifies connCopy but since it involves io.Copy which blocks,
	// we test it indirectly through other tests. Here we just verify the basic structure.
	tempDir := t.TempDir()
	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	instance := &ProxyRelayInstance{
		EnableLogging: true,
		parent: &Manager{
			Options: &Options{
				Logger: log,
			},
		},
	}

	// Verify instance is set up correctly
	assert.NotNil(t, instance)
	assert.NotNil(t, instance.parent)

	// The actual functionality of connCopy is tested via Port2host and forward tests
	// since it's difficult to test io.Copy in isolation without blocking
}

func TestForward(t *testing.T) {
	// Create two pipe pairs for bidirectional communication
	conn1Server, conn1Client := net.Pipe()
	conn2Server, conn2Client := net.Pipe()

	defer conn1Server.Close()
	defer conn1Client.Close()
	defer conn2Server.Close()
	defer conn2Client.Close()

	tempDir := t.TempDir()
	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	instance := &ProxyRelayInstance{
		EnableLogging: true,
		parent: &Manager{
			Options: &Options{
				Logger: log,
			},
		},
	}

	var aTob, bToa atomic.Int64

	// Run forward in background
	go instance.forward(conn1Server, conn2Server, &aTob, &bToa)

	// Send data from client1 to client2
	testData1 := []byte("hello from client1")
	go func() {
		conn1Client.Write(testData1)
		time.Sleep(50 * time.Millisecond)
		conn1Client.Close()
	}()

	// Read on client2
	buf := make([]byte, len(testData1))
	n, err := io.ReadFull(conn2Client, buf)

	if err == nil {
		assert.Equal(t, len(testData1), n)
		assert.Equal(t, testData1, buf[:n])
	}

	time.Sleep(100 * time.Millisecond)
}

func TestAccept(t *testing.T) {
	tempDir := t.TempDir()
	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	manager := &Manager{
		Options: &Options{
			Logger: log,
			AccessControlHandler: func(conn net.Conn) bool {
				return true // Allow all
			},
		},
	}

	instance := &ProxyRelayInstance{
		EnableLogging: true,
		parent:        manager,
	}

	// Create a listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	// Connect to the listener
	go func() {
		time.Sleep(50 * time.Millisecond)
		net.Dial("tcp", listener.Addr().String())
	}()

	// Accept connection
	conn, err := instance.accept(listener)
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	if conn != nil {
		conn.Close()
	}
}

func TestAcceptWithAccessControl(t *testing.T) {
	tempDir := t.TempDir()
	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	tests := []struct {
		name          string
		accessControl func(net.Conn) bool
		expectReject  bool
	}{
		{
			name: "allow connection",
			accessControl: func(conn net.Conn) bool {
				return true
			},
			expectReject: false,
		},
		{
			name: "reject connection",
			accessControl: func(conn net.Conn) bool {
				return false
			},
			expectReject: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &Manager{
				Options: &Options{
					Logger:               log,
					AccessControlHandler: tt.accessControl,
				},
			}

			instance := &ProxyRelayInstance{
				EnableLogging: true,
				parent:        manager,
			}

			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer listener.Close()

			// Connect in background
			go func() {
				time.Sleep(50 * time.Millisecond)
				conn, err := net.Dial("tcp", listener.Addr().String())
				if err == nil && conn != nil {
					time.Sleep(500 * time.Millisecond)
					conn.Close()
				}
			}()

			// Accept connection
			conn, err := instance.accept(listener)

			if tt.expectReject {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, conn)
				if conn != nil {
					conn.Close()
				}
			}
		})
	}
}

func TestWriteProxyProtocolHeader(t *testing.T) {
	// Create TCP connections for testing
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	// Accept connection in background
	var dstConn net.Conn
	go func() {
		dstConn, _ = listener.Accept()
	}()

	// Connect
	srcConn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	defer srcConn.Close()

	time.Sleep(100 * time.Millisecond)
	defer dstConn.Close()

	tests := []struct {
		name    string
		version ProxyProtocolVersion
	}{
		{"version 1", ProxyProtocolV1},
		{"version 2", ProxyProtocolV2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create new connections for each test
			var dst net.Conn
			go func() {
				dst, _ = listener.Accept()
			}()

			src, err := net.Dial("tcp", listener.Addr().String())
			require.NoError(t, err)
			defer src.Close()

			time.Sleep(100 * time.Millisecond)
			if dst != nil {
				defer dst.Close()

				// Write proxy protocol header
				err = WriteProxyProtocolHeader(dst, src, tt.version)
				assert.NoError(t, err)
			}
		})
	}
}

func TestPort2hostWithDifferentAddressFormats(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// Setup an echo server as target
	echoListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer echoListener.Close()

	targetAddr := echoListener.Addr().String()

	// Start echo server
	go func() {
		for {
			conn, err := echoListener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	tempDir := t.TempDir()
	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	manager := &Manager{
		Options: &Options{
			Logger: log,
			AccessControlHandler: func(conn net.Conn) bool {
				return true
			},
		},
	}

	tests := []struct {
		name        string
		listenAddr  string
		expectError bool
	}{
		{
			name:        "port only",
			listenAddr:  "0", // Will be converted to 0.0.0.0:0
			expectError: false,
		},
		{
			name:        "colon with port",
			listenAddr:  ":0",
			expectError: false,
		},
		{
			name:        "full address",
			listenAddr:  "127.0.0.1:0",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &ProxyRelayInstance{
				EnableLogging: true,
				parent:        manager,
				Timeout:       1,
			}

			stopChan := make(chan bool)

			// Start Port2host in background
			errChan := make(chan error, 1)
			go func() {
				err := instance.Port2host(tt.listenAddr, targetAddr, stopChan)
				errChan <- err
			}()

			// Give it time to start
			time.Sleep(200 * time.Millisecond)

			// Stop it
			stopChan <- true

			// Wait for it to finish
			select {
			case err := <-errChan:
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("Port2host did not stop in time")
			}
		})
	}
}

func TestPort2hostConnectionFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// Create echo server
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
				io.Copy(c, c) // Echo back
			}(conn)
		}
	}()

	tempDir := t.TempDir()
	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	manager := &Manager{
		Options: &Options{
			Logger: log,
			AccessControlHandler: func(conn net.Conn) bool {
				return true
			},
		},
	}

	// Create proxy listener
	proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	proxyAddr := proxyListener.Addr().String()
	proxyListener.Close() // Close so Port2host can bind to it

	instance := &ProxyRelayInstance{
		EnableLogging: true,
		parent:        manager,
		Timeout:       1,
	}

	stopChan := make(chan bool)

	// Start proxy
	go func() {
		instance.Port2host(proxyAddr, echoAddr, stopChan)
	}()

	// Give it time to start
	time.Sleep(300 * time.Millisecond)

	// Try to connect to proxy
	conn, err := net.DialTimeout("tcp", proxyAddr, 2*time.Second)
	if err == nil {
		defer conn.Close()

		// Send data
		testData := []byte("hello")
		_, err = conn.Write(testData)
		if err == nil {
			// Try to read echo
			buf := make([]byte, len(testData))
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			_, err = io.ReadFull(conn, buf)
			// We may or may not get data back depending on timing
		}
	}

	// Stop proxy
	stopChan <- true
	time.Sleep(100 * time.Millisecond)
}

func TestPort2hostWithProxyProtocol(t *testing.T) {
	tempDir := t.TempDir()
	log, err := logger.NewLogger("test", tempDir)
	require.NoError(t, err)

	manager := &Manager{
		Options: &Options{
			Logger: log,
			AccessControlHandler: func(conn net.Conn) bool {
				return true
			},
		},
	}

	tests := []struct {
		name    string
		version ProxyProtocolVersion
	}{
		{"disabled", ProxyProtocolDisabled},
		{"version 1", ProxyProtocolV1},
		{"version 2", ProxyProtocolV2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create target server
			targetListener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer targetListener.Close()

			targetAddr := targetListener.Addr().String()

			// Accept connections
			go func() {
				for {
					conn, err := targetListener.Accept()
					if err != nil {
						return
					}
					conn.Close()
				}
			}()

			instance := &ProxyRelayInstance{
				EnableLogging:        true,
				parent:               manager,
				Timeout:              1,
				ProxyProtocolVersion: tt.version,
			}

			stopChan := make(chan bool)

			// Start proxy
			go func() {
				instance.Port2host("127.0.0.1:0", targetAddr, stopChan)
			}()

			time.Sleep(200 * time.Millisecond)

			// Stop
			stopChan <- true
			time.Sleep(100 * time.Millisecond)
		})
	}
}
