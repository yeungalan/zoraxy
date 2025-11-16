package streamproxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"imuslab.com/zoraxy/mod/info/logger"
)

func setupTestHTTPManager(t *testing.T) *Manager {
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

	return manager
}

func createPostRequest(params map[string]string) *http.Request {
	form := url.Values{}
	for key, value := range params {
		form.Add(key, value)
	}

	req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return req
}

func TestHandleAddProxyConfig(t *testing.T) {
	manager := setupTestHTTPManager(t)

	tests := []struct {
		name           string
		params         map[string]string
		expectedStatus int
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "valid TCP config",
			params: map[string]string{
				"name":       "test-proxy",
				"listenAddr": "127.0.0.1:8001",
				"proxyAddr":  "127.0.0.1:8002",
				"useTCP":     "true",
				"timeout":    "60",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var uuid string
				err := json.Unmarshal(w.Body.Bytes(), &uuid)
				assert.NoError(t, err)
				assert.NotEmpty(t, uuid)

				// Verify config was created
				cfg, err := manager.GetConfigByUUID(uuid)
				assert.NoError(t, err)
				assert.Equal(t, "test-proxy", cfg.Name)
				assert.True(t, cfg.UseTCP)
			},
		},
		{
			name: "valid UDP config",
			params: map[string]string{
				"name":       "udp-proxy",
				"listenAddr": "127.0.0.1:8003",
				"proxyAddr":  "127.0.0.1:8004",
				"useUDP":     "true",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var uuid string
				json.Unmarshal(w.Body.Bytes(), &uuid)
				cfg, _ := manager.GetConfigByUUID(uuid)
				assert.True(t, cfg.UseUDP)
			},
		},
		{
			name: "with proxy protocol v1",
			params: map[string]string{
				"name":                 "proxy-v1",
				"listenAddr":           "127.0.0.1:8005",
				"proxyAddr":            "127.0.0.1:8006",
				"useTCP":               "true",
				"proxyProtocolVersion": "1",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var uuid string
				json.Unmarshal(w.Body.Bytes(), &uuid)
				cfg, _ := manager.GetConfigByUUID(uuid)
				assert.Equal(t, ProxyProtocolV1, cfg.ProxyProtocolVersion)
			},
		},
		{
			name: "with proxy protocol v2",
			params: map[string]string{
				"name":                 "proxy-v2",
				"listenAddr":           "127.0.0.1:8007",
				"proxyAddr":            "127.0.0.1:8008",
				"useTCP":               "true",
				"proxyProtocolVersion": "2",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var uuid string
				json.Unmarshal(w.Body.Bytes(), &uuid)
				cfg, _ := manager.GetConfigByUUID(uuid)
				assert.Equal(t, ProxyProtocolV2, cfg.ProxyProtocolVersion)
			},
		},
		{
			name: "with logging enabled",
			params: map[string]string{
				"name":          "logging-proxy",
				"listenAddr":    "127.0.0.1:8009",
				"proxyAddr":     "127.0.0.1:8010",
				"useTCP":        "true",
				"enableLogging": "true",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var uuid string
				json.Unmarshal(w.Body.Bytes(), &uuid)
				cfg, _ := manager.GetConfigByUUID(uuid)
				assert.True(t, cfg.EnableLogging)
			},
		},
		{
			name: "missing name",
			params: map[string]string{
				"listenAddr": "127.0.0.1:8011",
				"proxyAddr":  "127.0.0.1:8012",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "error")
			},
		},
		{
			name: "missing listenAddr",
			params: map[string]string{
				"name":      "test",
				"proxyAddr": "127.0.0.1:8012",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "error")
			},
		},
		{
			name: "missing proxyAddr",
			params: map[string]string{
				"name":       "test",
				"listenAddr": "127.0.0.1:8011",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "error")
			},
		},
		{
			name: "invalid timeout",
			params: map[string]string{
				"name":       "test",
				"listenAddr": "127.0.0.1:8011",
				"proxyAddr":  "127.0.0.1:8012",
				"timeout":    "invalid",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createPostRequest(tt.params)
			w := httptest.NewRecorder()

			manager.HandleAddProxyConfig(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w)
		})
	}
}

func TestHandleEditProxyConfigs(t *testing.T) {
	manager := setupTestHTTPManager(t)

	// Create a config to edit
	uuid := manager.NewConfig(&ProxyRelayOptions{
		Name:          "original",
		ListeningAddr: "127.0.0.1:8100",
		ProxyAddr:     "127.0.0.1:8101",
		Timeout:       30,
		UseTCP:        true,
	})

	tests := []struct {
		name           string
		params         map[string]string
		expectedStatus int
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "edit name",
			params: map[string]string{
				"uuid":   uuid,
				"name":   "updated-name",
				"useTCP": "true",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				cfg, _ := manager.GetConfigByUUID(uuid)
				assert.Equal(t, "updated-name", cfg.Name)
			},
		},
		{
			name: "edit listen address",
			params: map[string]string{
				"uuid":       uuid,
				"listenAddr": "127.0.0.1:9999",
				"useTCP":     "true",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				cfg, _ := manager.GetConfigByUUID(uuid)
				assert.Equal(t, "127.0.0.1:9999", cfg.ListeningAddress)
			},
		},
		{
			name: "edit proxy address",
			params: map[string]string{
				"uuid":      uuid,
				"proxyAddr": "127.0.0.1:8888",
				"useTCP":    "true",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				cfg, _ := manager.GetConfigByUUID(uuid)
				assert.Equal(t, "127.0.0.1:8888", cfg.ProxyTargetAddr)
			},
		},
		{
			name: "enable UDP",
			params: map[string]string{
				"uuid":   uuid,
				"useTCP": "true",
				"useUDP": "true",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				cfg, _ := manager.GetConfigByUUID(uuid)
				assert.True(t, cfg.UseUDP)
			},
		},
		{
			name: "update timeout",
			params: map[string]string{
				"uuid":    uuid,
				"timeout": "120",
				"useTCP":  "true",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				cfg, _ := manager.GetConfigByUUID(uuid)
				assert.Equal(t, 120, cfg.Timeout)
			},
		},
		{
			name: "missing uuid",
			params: map[string]string{
				"name": "test",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "error")
			},
		},
		{
			name: "invalid uuid",
			params: map[string]string{
				"uuid": "non-existent",
				"name": "test",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "error")
			},
		},
		{
			name: "invalid timeout",
			params: map[string]string{
				"uuid":    uuid,
				"timeout": "invalid",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createPostRequest(tt.params)
			w := httptest.NewRecorder()

			manager.HandleEditProxyConfigs(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w)
		})
	}
}

func TestHandleListConfigs(t *testing.T) {
	manager := setupTestHTTPManager(t)

	// Create some configs
	manager.NewConfig(&ProxyRelayOptions{
		Name:          "config1",
		ListeningAddr: "127.0.0.1:8200",
		ProxyAddr:     "127.0.0.1:8201",
		UseTCP:        true,
	})

	manager.NewConfig(&ProxyRelayOptions{
		Name:          "config2",
		ListeningAddr: "127.0.0.1:8202",
		ProxyAddr:     "127.0.0.1:8203",
		UseUDP:        true,
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	manager.HandleListConfigs(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var configs []*ProxyRelayInstance
	err := json.Unmarshal(w.Body.Bytes(), &configs)
	assert.NoError(t, err)
	assert.Len(t, configs, 2)
}

func TestHandleStartProxy(t *testing.T) {
	manager := setupTestHTTPManager(t)

	// Create a config
	uuid := manager.NewConfig(&ProxyRelayOptions{
		Name:          "start-test",
		ListeningAddr: "127.0.0.1:0",
		ProxyAddr:     "127.0.0.1:8301",
		Timeout:       1,
		UseTCP:        true,
	})

	tests := []struct {
		name           string
		params         map[string]string
		expectedStatus int
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "start valid proxy",
			params: map[string]string{
				"uuid": uuid,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				cfg, _ := manager.GetConfigByUUID(uuid)
				assert.True(t, cfg.Running)
			},
		},
		{
			name:           "missing uuid",
			params:         map[string]string{},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "error")
			},
		},
		{
			name: "invalid uuid",
			params: map[string]string{
				"uuid": "non-existent",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createPostRequest(tt.params)
			w := httptest.NewRecorder()

			manager.HandleStartProxy(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w)
		})
	}

	// Cleanup - stop the proxy
	cfg, _ := manager.GetConfigByUUID(uuid)
	if cfg.IsRunning() {
		cfg.Stop()
	}
}

func TestHandleStopProxy(t *testing.T) {
	manager := setupTestHTTPManager(t)

	// Create and start a config
	uuid := manager.NewConfig(&ProxyRelayOptions{
		Name:          "stop-test",
		ListeningAddr: "127.0.0.1:0",
		ProxyAddr:     "127.0.0.1:8401",
		Timeout:       1,
		UseTCP:        true,
	})

	cfg, _ := manager.GetConfigByUUID(uuid)
	cfg.Start()

	tests := []struct {
		name           string
		params         map[string]string
		expectedStatus int
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "stop running proxy",
			params: map[string]string{
				"uuid": uuid,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				cfg, _ := manager.GetConfigByUUID(uuid)
				assert.False(t, cfg.Running)
			},
		},
		{
			name:           "missing uuid",
			params:         map[string]string{},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "error")
			},
		},
		{
			name: "invalid uuid",
			params: map[string]string{
				"uuid": "non-existent",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createPostRequest(tt.params)
			w := httptest.NewRecorder()

			manager.HandleStopProxy(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w)
		})
	}
}

func TestHandleStopProxyNotRunning(t *testing.T) {
	manager := setupTestHTTPManager(t)

	// Create a config but don't start it
	uuid := manager.NewConfig(&ProxyRelayOptions{
		Name:          "not-running",
		ListeningAddr: "127.0.0.1:8500",
		ProxyAddr:     "127.0.0.1:8501",
		UseTCP:        true,
	})

	req := createPostRequest(map[string]string{"uuid": uuid})
	w := httptest.NewRecorder()

	manager.HandleStopProxy(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestHandleRemoveProxy(t *testing.T) {
	manager := setupTestHTTPManager(t)

	// Create a config
	uuid := manager.NewConfig(&ProxyRelayOptions{
		Name:          "remove-test",
		ListeningAddr: "127.0.0.1:8600",
		ProxyAddr:     "127.0.0.1:8601",
		UseTCP:        true,
	})

	tests := []struct {
		name           string
		params         map[string]string
		startProxy     bool
		expectedStatus int
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "remove stopped proxy",
			params: map[string]string{
				"uuid": uuid,
			},
			startProxy:     false,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				_, err := manager.GetConfigByUUID(uuid)
				assert.Error(t, err)
			},
		},
		{
			name:           "missing uuid",
			params:         map[string]string{},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "error")
			},
		},
		{
			name: "invalid uuid",
			params: map[string]string{
				"uuid": "non-existent",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createPostRequest(tt.params)
			w := httptest.NewRecorder()

			manager.HandleRemoveProxy(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w)
		})
	}
}

func TestHandleRemoveProxyRunning(t *testing.T) {
	manager := setupTestHTTPManager(t)

	// Create and start a config
	uuid := manager.NewConfig(&ProxyRelayOptions{
		Name:          "running-remove",
		ListeningAddr: "127.0.0.1:0",
		ProxyAddr:     "127.0.0.1:8701",
		Timeout:       1,
		UseTCP:        true,
	})

	cfg, _ := manager.GetConfigByUUID(uuid)
	cfg.Start()

	req := createPostRequest(map[string]string{"uuid": uuid})
	w := httptest.NewRecorder()

	manager.HandleRemoveProxy(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "error")

	// Cleanup
	cfg.Stop()
}

func TestHandleGetProxyStatus(t *testing.T) {
	manager := setupTestHTTPManager(t)

	// Create a config
	uuid := manager.NewConfig(&ProxyRelayOptions{
		Name:          "status-test",
		ListeningAddr: "127.0.0.1:8800",
		ProxyAddr:     "127.0.0.1:8801",
		UseTCP:        true,
		Timeout:       60,
	})

	tests := []struct {
		name           string
		params         map[string]string
		expectedStatus int
		checkResponse  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "get valid proxy status",
			params: map[string]string{
				"uuid": uuid,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var cfg ProxyRelayInstance
				err := json.Unmarshal(w.Body.Bytes(), &cfg)
				assert.NoError(t, err)
				assert.Equal(t, "status-test", cfg.Name)
				assert.Equal(t, uuid, cfg.UUID)
			},
		},
		{
			name:           "missing uuid",
			params:         map[string]string{},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "error")
			},
		},
		{
			name: "invalid uuid",
			params: map[string]string{
				"uuid": "non-existent",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Contains(t, w.Body.String(), "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use GET request for this handler
			req := httptest.NewRequest("GET", "/?uuid="+tt.params["uuid"], nil)
			w := httptest.NewRecorder()

			manager.HandleGetProxyStatus(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w)
		})
	}
}
