package websocketproxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebSocketMessagePassing tests actual WebSocket message passing through the proxy
func TestWebSocketMessagePassing(t *testing.T) {
	tests := []struct {
		name        string
		messageType int
		message     string
	}{
		{
			name:        "Text message",
			messageType: websocket.TextMessage,
			message:     "Hello WebSocket",
		},
		{
			name:        "Binary message",
			messageType: websocket.BinaryMessage,
			message:     "Binary data",
		},
		{
			name:        "Empty message",
			messageType: websocket.TextMessage,
			message:     "",
		},
		{
			name:        "Large message",
			messageType: websocket.TextMessage,
			message:     strings.Repeat("A", 10000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup backend echo server
			backendUpgrader := &websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool { return true },
			}

			backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := backendUpgrader.Upgrade(w, r, nil)
				if err != nil {
					t.Logf("Backend upgrade error: %v", err)
					return
				}
				defer conn.Close()

				// Echo messages back
				for {
					msgType, msg, err := conn.ReadMessage()
					if err != nil {
						return
					}
					err = conn.WriteMessage(msgType, msg)
					if err != nil {
						return
					}
				}
			}))
			defer backendServer.Close()

			// Setup proxy
			backendURL, _ := url.Parse(strings.Replace(backendServer.URL, "http", "ws", 1))
			proxy := NewProxy(backendURL, Options{
				SkipOriginCheck: true,
			})

			proxyServer := httptest.NewServer(proxy)
			defer proxyServer.Close()

			// Connect client to proxy
			proxyWsURL := strings.Replace(proxyServer.URL, "http", "ws", 1)
			conn, _, err := websocket.DefaultDialer.Dial(proxyWsURL, nil)
			require.NoError(t, err)
			defer conn.Close()

			// Send message
			err = conn.WriteMessage(tt.messageType, []byte(tt.message))
			require.NoError(t, err)

			// Receive echo
			msgType, msg, err := conn.ReadMessage()
			require.NoError(t, err)
			assert.Equal(t, tt.messageType, msgType)
			assert.Equal(t, tt.message, string(msg))
		})
	}
}

// TestWebSocketCloseHandling tests WebSocket close frame handling
func TestWebSocketCloseHandling(t *testing.T) {
	tests := []struct {
		name      string
		closeCode int
		closeText string
	}{
		{
			name:      "Normal closure",
			closeCode: websocket.CloseNormalClosure,
			closeText: "Goodbye",
		},
		{
			name:      "Going away",
			closeCode: websocket.CloseGoingAway,
			closeText: "Server going away",
		},
		{
			name:      "Protocol error",
			closeCode: websocket.CloseProtocolError,
			closeText: "Protocol error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup backend server that closes connection
			backendUpgrader := &websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool { return true },
			}

			backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := backendUpgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer conn.Close()

				// Wait for a message, then close with specific code
				_, _, err = conn.ReadMessage()
				if err == nil {
					closeMsg := websocket.FormatCloseMessage(tt.closeCode, tt.closeText)
					conn.WriteMessage(websocket.CloseMessage, closeMsg)
				}
			}))
			defer backendServer.Close()

			// Setup proxy
			backendURL, _ := url.Parse(strings.Replace(backendServer.URL, "http", "ws", 1))
			proxy := NewProxy(backendURL, Options{
				SkipOriginCheck: true,
			})

			proxyServer := httptest.NewServer(proxy)
			defer proxyServer.Close()

			// Connect client to proxy
			proxyWsURL := strings.Replace(proxyServer.URL, "http", "ws", 1)
			conn, _, err := websocket.DefaultDialer.Dial(proxyWsURL, nil)
			require.NoError(t, err)
			defer conn.Close()

			// Send a message to trigger close
			err = conn.WriteMessage(websocket.TextMessage, []byte("trigger close"))
			require.NoError(t, err)

			// Should receive close message
			_, _, err = conn.ReadMessage()
			if err != nil {
				closeErr, ok := err.(*websocket.CloseError)
				if ok {
					assert.Equal(t, tt.closeCode, closeErr.Code)
				}
			}
		})
	}
}

// TestWebSocketBidirectionalCommunication tests bidirectional message flow
func TestWebSocketBidirectionalCommunication(t *testing.T) {
	// Setup backend server
	backendUpgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	messageCount := 0
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := backendUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Echo messages back with a counter
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			messageCount++
			response := fmt.Sprintf("%s - count: %d", string(msg), messageCount)
			err = conn.WriteMessage(msgType, []byte(response))
			if err != nil {
				return
			}
		}
	}))
	defer backendServer.Close()

	// Setup proxy
	backendURL, _ := url.Parse(strings.Replace(backendServer.URL, "http", "ws", 1))
	proxy := NewProxy(backendURL, Options{
		SkipOriginCheck: true,
	})

	proxyServer := httptest.NewServer(proxy)
	defer proxyServer.Close()

	// Connect client to proxy
	proxyWsURL := strings.Replace(proxyServer.URL, "http", "ws", 1)
	conn, _, err := websocket.DefaultDialer.Dial(proxyWsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Send multiple messages
	messages := []string{"message1", "message2", "message3"}
	for i, msg := range messages {
		err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
		require.NoError(t, err)

		// Receive response
		_, response, err := conn.ReadMessage()
		require.NoError(t, err)
		assert.Contains(t, string(response), msg)
		assert.Contains(t, string(response), fmt.Sprintf("count: %d", i+1))
	}
}

// TestWebSocketProtocolNegotiation tests subprotocol negotiation
func TestWebSocketProtocolNegotiation(t *testing.T) {
	supportedProtocols := []string{"chat", "superchat"}

	// Setup backend server with protocol support
	backendUpgrader := &websocket.Upgrader{
		CheckOrigin:  func(r *http.Request) bool { return true },
		Subprotocols: supportedProtocols,
	}

	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := backendUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Keep connection open for a bit
		time.Sleep(100 * time.Millisecond)
	}))
	defer backendServer.Close()

	// Setup proxy
	backendURL, _ := url.Parse(strings.Replace(backendServer.URL, "http", "ws", 1))
	proxy := NewProxy(backendURL, Options{
		SkipOriginCheck: true,
	})
	proxy.Upgrader = backendUpgrader

	proxyServer := httptest.NewServer(proxy)
	defer proxyServer.Close()

	// Connect with subprotocol
	proxyWsURL := strings.Replace(proxyServer.URL, "http", "ws", 1)
	dialer := websocket.Dialer{
		Subprotocols: []string{"chat", "unsupported"},
	}
	conn, resp, err := dialer.Dial(proxyWsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Check that the supported protocol was selected
	selectedProtocol := resp.Header.Get("Sec-WebSocket-Protocol")
	assert.Equal(t, "chat", selectedProtocol)
}

// TestWebSocketWithCookies tests cookie forwarding through proxy
func TestWebSocketWithCookies(t *testing.T) {
	receivedCookies := ""

	// Setup backend server that captures cookies
	backendUpgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCookies = r.Header.Get("Cookie")
		conn, err := backendUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		conn.Close()
	}))
	defer backendServer.Close()

	// Setup proxy
	backendURL, _ := url.Parse(strings.Replace(backendServer.URL, "http", "ws", 1))
	proxy := NewProxy(backendURL, Options{
		SkipOriginCheck: true,
	})

	proxyServer := httptest.NewServer(proxy)
	defer proxyServer.Close()

	// Connect with cookies
	proxyWsURL := strings.Replace(proxyServer.URL, "http", "ws", 1)
	headers := http.Header{}
	headers.Add("Cookie", "session=abc123; user=john")
	conn, _, err := websocket.DefaultDialer.Dial(proxyWsURL, headers)
	require.NoError(t, err)
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)
	assert.Contains(t, receivedCookies, "session=abc123")
}

// TestWebSocketConnectionError tests error handling during WebSocket operations
func TestWebSocketConnectionError(t *testing.T) {
	// Setup backend server that immediately closes after upgrade
	backendUpgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := backendUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Immediately close the connection
		conn.Close()
	}))
	defer backendServer.Close()

	// Setup proxy
	backendURL, _ := url.Parse(strings.Replace(backendServer.URL, "http", "ws", 1))
	proxy := NewProxy(backendURL, Options{
		SkipOriginCheck: true,
	})

	proxyServer := httptest.NewServer(proxy)
	defer proxyServer.Close()

	// Connect client to proxy
	proxyWsURL := strings.Replace(proxyServer.URL, "http", "ws", 1)
	conn, _, err := websocket.DefaultDialer.Dial(proxyWsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Try to send message - should fail since backend closed
	time.Sleep(100 * time.Millisecond)
	err = conn.WriteMessage(websocket.TextMessage, []byte("test"))
	if err == nil {
		// If write succeeds, read should fail
		_, _, err = conn.ReadMessage()
	}
	assert.Error(t, err)
}

// TestBackendHandshakeFailure tests handling of backend handshake failures
func TestBackendHandshakeFailure(t *testing.T) {
	// Setup backend server that rejects WebSocket upgrade
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "WebSocket upgrade not allowed", http.StatusForbidden)
	}))
	defer backendServer.Close()

	// Setup proxy
	backendURL, _ := url.Parse(strings.Replace(backendServer.URL, "http", "ws", 1))
	proxy := NewProxy(backendURL, Options{
		SkipOriginCheck: true,
	})

	// Create a test request
	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	rw := httptest.NewRecorder()
	proxy.ServeHTTP(rw, req)

	// Should get forbidden status from backend
	assert.Equal(t, http.StatusForbidden, rw.Code)
}

// TestWebSocketMultipleClients tests multiple concurrent client connections
func TestWebSocketMultipleClients(t *testing.T) {
	// Setup backend echo server
	backendUpgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := backendUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Echo messages back
		for {
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			err = conn.WriteMessage(msgType, msg)
			if err != nil {
				return
			}
		}
	}))
	defer backendServer.Close()

	// Setup proxy
	backendURL, _ := url.Parse(strings.Replace(backendServer.URL, "http", "ws", 1))
	proxy := NewProxy(backendURL, Options{
		SkipOriginCheck: true,
	})

	proxyServer := httptest.NewServer(proxy)
	defer proxyServer.Close()

	// Connect multiple clients simultaneously
	numClients := 5
	done := make(chan bool, numClients)

	for i := 0; i < numClients; i++ {
		go func(clientID int) {
			proxyWsURL := strings.Replace(proxyServer.URL, "http", "ws", 1)
			conn, _, err := websocket.DefaultDialer.Dial(proxyWsURL, nil)
			if err != nil {
				t.Logf("Client %d dial error: %v", clientID, err)
				done <- false
				return
			}
			defer conn.Close()

			message := fmt.Sprintf("Client %d message", clientID)
			err = conn.WriteMessage(websocket.TextMessage, []byte(message))
			if err != nil {
				t.Logf("Client %d write error: %v", clientID, err)
				done <- false
				return
			}

			_, response, err := conn.ReadMessage()
			if err != nil {
				t.Logf("Client %d read error: %v", clientID, err)
				done <- false
				return
			}

			if string(response) != message {
				t.Logf("Client %d: expected %s, got %s", clientID, message, string(response))
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// Wait for all clients to finish
	successCount := 0
	for i := 0; i < numClients; i++ {
		if <-done {
			successCount++
		}
	}

	assert.Equal(t, numClients, successCount, "All clients should complete successfully")
}
