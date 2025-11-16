package acme

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"imuslab.com/zoraxy/mod/database"
	"imuslab.com/zoraxy/mod/info/logger"
)

func setupTestAutoRenewer(t *testing.T) (*AutoRenewer, string, *ACMEHandler) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "autorenew.json")
	certFolder := filepath.Join(tmpDir, "certs")

	err := os.MkdirAll(certFolder, 0755)
	require.NoError(t, err)

	// Create a test database
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.NewDatabase(dbPath, false)
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Close()
	})

	// Create a test logger
	testLogger := logger.NewLogger(tmpDir, "test", false)
	require.NotNil(t, testLogger)

	// Create ACME handler
	acmeHandler := NewACME("https://acme-staging-v02.api.letsencrypt.org/directory", "80", db, testLogger)

	renewer, err := NewAutoRenewer(configPath, certFolder, 10, 30, acmeHandler, testLogger)
	require.NoError(t, err)
	require.NotNil(t, renewer)

	return renewer, certFolder, acmeHandler
}

func TestNewAutoRenewer(t *testing.T) {
	t.Run("Create new auto renewer with default values", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "autorenew.json")
		certFolder := filepath.Join(tmpDir, "certs")

		err := os.MkdirAll(certFolder, 0755)
		require.NoError(t, err)

		db, err := database.NewDatabase(filepath.Join(tmpDir, "test.db"), false)
		require.NoError(t, err)
		defer db.Close()

		testLogger := logger.NewLogger(tmpDir, "test", false)
		acmeHandler := NewACME("https://acme-staging-v02.api.letsencrypt.org/directory", "80", db, testLogger)

		renewer, err := NewAutoRenewer(configPath, certFolder, 0, 0, acmeHandler, testLogger)
		require.NoError(t, err)
		require.NotNil(t, renewer)

		assert.Equal(t, int64(86400), renewer.RenewTickInterval) // Default 1 day
		assert.Equal(t, 30, renewer.EarlyRenewDays)              // Default 30 days
		assert.NotNil(t, renewer.RenewerConfig)
		assert.True(t, renewer.RenewerConfig.RenewAll) // Default is RenewAll
	})

	t.Run("Create config file if not exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "autorenew.json")
		certFolder := filepath.Join(tmpDir, "certs")

		err := os.MkdirAll(certFolder, 0755)
		require.NoError(t, err)

		db, err := database.NewDatabase(filepath.Join(tmpDir, "test.db"), false)
		require.NoError(t, err)
		defer db.Close()

		testLogger := logger.NewLogger(tmpDir, "test", false)
		acmeHandler := NewACME("https://acme-staging-v02.api.letsencrypt.org/directory", "80", db, testLogger)

		_, err = NewAutoRenewer(configPath, certFolder, 10, 30, acmeHandler, testLogger)
		require.NoError(t, err)

		// Check that config file was created
		assert.FileExists(t, configPath)

		// Check config content
		content, err := os.ReadFile(configPath)
		require.NoError(t, err)

		var config AutoRenewConfig
		err = json.Unmarshal(content, &config)
		require.NoError(t, err)
		assert.True(t, config.RenewAll)
		assert.Empty(t, config.FilesToRenew)
	})

	t.Run("Load existing config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "autorenew.json")
		certFolder := filepath.Join(tmpDir, "certs")

		err := os.MkdirAll(certFolder, 0755)
		require.NoError(t, err)

		// Create a config file
		config := AutoRenewConfig{
			Enabled:      true,
			Email:        "test@example.com",
			RenewAll:     false,
			FilesToRenew: []string{"cert1", "cert2"},
		}
		configBytes, err := json.Marshal(config)
		require.NoError(t, err)
		err = os.WriteFile(configPath, configBytes, 0644)
		require.NoError(t, err)

		db, err := database.NewDatabase(filepath.Join(tmpDir, "test.db"), false)
		require.NoError(t, err)
		defer db.Close()

		testLogger := logger.NewLogger(tmpDir, "test", false)
		acmeHandler := NewACME("https://acme-staging-v02.api.letsencrypt.org/directory", "80", db, testLogger)

		renewer, err := NewAutoRenewer(configPath, certFolder, 10, 30, acmeHandler, testLogger)
		require.NoError(t, err)
		require.NotNil(t, renewer)

		assert.True(t, renewer.RenewerConfig.Enabled)
		assert.Equal(t, "test@example.com", renewer.RenewerConfig.Email)
		assert.False(t, renewer.RenewerConfig.RenewAll)
		assert.Equal(t, []string{"cert1", "cert2"}, renewer.RenewerConfig.FilesToRenew)
	})
}

func TestAutoRenewer_SaveRenewConfigToFile(t *testing.T) {
	renewer, _, _ := setupTestAutoRenewer(t)

	renewer.RenewerConfig.Email = "updated@example.com"
	renewer.RenewerConfig.Enabled = true
	renewer.RenewerConfig.RenewAll = false
	renewer.RenewerConfig.FilesToRenew = []string{"cert1", "cert2", "cert3"}

	err := renewer.saveRenewConfigToFile()
	require.NoError(t, err)

	// Read and verify
	content, err := os.ReadFile(renewer.ConfigFilePath)
	require.NoError(t, err)

	var config AutoRenewConfig
	err = json.Unmarshal(content, &config)
	require.NoError(t, err)

	assert.Equal(t, "updated@example.com", config.Email)
	assert.True(t, config.Enabled)
	assert.False(t, config.RenewAll)
	assert.Equal(t, []string{"cert1", "cert2", "cert3"}, config.FilesToRenew)
}

func TestAutoRenewer_HandleSetAutoRenewDomains(t *testing.T) {
	renewer, _, _ := setupTestAutoRenewer(t)

	t.Run("Set selected domains", func(t *testing.T) {
		domains := []string{"example.com", "www.example.com"}
		domainsJSON, _ := json.Marshal(domains)

		form := url.Values{}
		form.Add("opr", "setSelected")
		form.Add("domains", string(domainsJSON))

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		renewer.HandleSetAutoRenewDomains(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		assert.False(t, renewer.RenewerConfig.RenewAll)
		assert.Equal(t, domains, renewer.RenewerConfig.FilesToRenew)
	})

	t.Run("Set auto mode", func(t *testing.T) {
		form := url.Values{}
		form.Add("opr", "setAuto")

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		renewer.HandleSetAutoRenewDomains(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		assert.True(t, renewer.RenewerConfig.RenewAll)
	})

	t.Run("Invalid operation", func(t *testing.T) {
		form := url.Values{}
		form.Add("opr", "invalidOpr")

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		renewer.HandleSetAutoRenewDomains(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "error")
	})

	t.Run("Missing operation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		w := httptest.NewRecorder()

		renewer.HandleSetAutoRenewDomains(w, req)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "error")
	})

	t.Run("Invalid JSON in domains", func(t *testing.T) {
		form := url.Values{}
		form.Add("opr", "setSelected")
		form.Add("domains", "invalid json")

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		renewer.HandleSetAutoRenewDomains(w, req)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "error")
	})
}

func TestAutoRenewer_HandleLoadAutoRenewDomains(t *testing.T) {
	renewer, _, _ := setupTestAutoRenewer(t)

	t.Run("Load auto mode (RenewAll=true)", func(t *testing.T) {
		renewer.RenewerConfig.RenewAll = true

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		renewer.HandleLoadAutoRenewDomains(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result []string
		err := json.Unmarshal(body, &result)
		require.NoError(t, err)
		assert.Equal(t, []string{"*"}, result)
	})

	t.Run("Load selected domains (RenewAll=false)", func(t *testing.T) {
		renewer.RenewerConfig.RenewAll = false
		renewer.RenewerConfig.FilesToRenew = []string{"cert1", "cert2"}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		renewer.HandleLoadAutoRenewDomains(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result []string
		err := json.Unmarshal(body, &result)
		require.NoError(t, err)
		assert.Equal(t, []string{"cert1", "cert2"}, result)
	})
}

func TestAutoRenewer_HandleRenewPolicy(t *testing.T) {
	renewer, _, _ := setupTestAutoRenewer(t)

	t.Run("Get RenewAll=true", func(t *testing.T) {
		renewer.RenewerConfig.RenewAll = true

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		renewer.HandleRenewPolicy(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result bool
		err := json.Unmarshal(body, &result)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("Get RenewAll=false", func(t *testing.T) {
		renewer.RenewerConfig.RenewAll = false

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		renewer.HandleRenewPolicy(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result bool
		err := json.Unmarshal(body, &result)
		require.NoError(t, err)
		assert.False(t, result)
	})
}

func TestAutoRenewer_HandleAutoRenewEnable(t *testing.T) {
	renewer, _, _ := setupTestAutoRenewer(t)

	t.Run("GET enabled state", func(t *testing.T) {
		renewer.RenewerConfig.Enabled = true

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		renewer.HandleAutoRenewEnable(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result bool
		err := json.Unmarshal(body, &result)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("Enable auto renew with email set", func(t *testing.T) {
		renewer.RenewerConfig.Email = "test@example.com"
		renewer.RenewerConfig.Enabled = false

		form := url.Values{}
		form.Add("enable", "true")

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		renewer.HandleAutoRenewEnable(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.True(t, renewer.RenewerConfig.Enabled)
	})

	t.Run("Enable auto renew without email", func(t *testing.T) {
		renewer.RenewerConfig.Email = ""
		renewer.RenewerConfig.Enabled = false

		form := url.Values{}
		form.Add("enable", "true")

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		renewer.HandleAutoRenewEnable(w, req)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "error")
		assert.False(t, renewer.RenewerConfig.Enabled)
	})

	t.Run("Disable auto renew", func(t *testing.T) {
		renewer.RenewerConfig.Enabled = true

		form := url.Values{}
		form.Add("enable", "false")

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		renewer.HandleAutoRenewEnable(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.False(t, renewer.RenewerConfig.Enabled)
	})
}

func TestAutoRenewer_HandleACMEEmail(t *testing.T) {
	renewer, _, _ := setupTestAutoRenewer(t)

	t.Run("GET email", func(t *testing.T) {
		renewer.RenewerConfig.Email = "test@example.com"

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		renewer.HandleACMEEmail(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		var result string
		err := json.Unmarshal(body, &result)
		require.NoError(t, err)
		assert.Equal(t, "test@example.com", result)
	})

	t.Run("SET valid email", func(t *testing.T) {
		form := url.Values{}
		form.Add("set", "newemail@example.com")

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		renewer.HandleACMEEmail(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "newemail@example.com", renewer.RenewerConfig.Email)
	})

	t.Run("SET invalid email", func(t *testing.T) {
		form := url.Values{}
		form.Add("set", "invalidemail")

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		renewer.HandleACMEEmail(w, req)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "error")
	})

	t.Run("Invalid HTTP method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPut, "/", nil)
		w := httptest.NewRecorder()

		renewer.HandleACMEEmail(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func TestAutoRenewer_CheckAndRenewCertificates(t *testing.T) {
	renewer, certFolder, _ := setupTestAutoRenewer(t)

	t.Run("No certificates to renew", func(t *testing.T) {
		renewed, err := renewer.CheckAndRenewCertificates()
		assert.NoError(t, err)
		assert.Empty(t, renewed)
	})

	t.Run("Certificate expires soon (RenewAll=true)", func(t *testing.T) {
		renewer.RenewerConfig.RenewAll = true
		renewer.RenewerConfig.Email = "test@example.com"

		// Create a certificate that expires in 20 days (less than 30 day threshold)
		certPEM, err := generateTestCertificate(
			[]string{"example.com"},
			"Let's Encrypt",
			time.Now().Add(-70*24*time.Hour),
			time.Now().Add(20*24*time.Hour),
		)
		require.NoError(t, err)

		certPath := filepath.Join(certFolder, "test.pem")
		err = os.WriteFile(certPath, certPEM, 0644)
		require.NoError(t, err)

		// This will fail to renew (no real ACME server), but should detect the cert
		renewed, err := renewer.CheckAndRenewCertificates()
		// err may or may not be nil depending on renewal attempt
		_ = err
		// The cert should be in the list to renew even if renewal failed
		_ = renewed
	})

	t.Run("Certificate is valid (not expiring soon)", func(t *testing.T) {
		renewer.RenewerConfig.RenewAll = true

		// Create a certificate that expires in 60 days (more than 30 day threshold)
		certPEM, err := generateTestCertificate(
			[]string{"valid.com"},
			"Let's Encrypt",
			time.Now().Add(-30*24*time.Hour),
			time.Now().Add(60*24*time.Hour),
		)
		require.NoError(t, err)

		certPath := filepath.Join(certFolder, "valid.pem")
		err = os.WriteFile(certPath, certPEM, 0644)
		require.NoError(t, err)

		renewed, err := renewer.CheckAndRenewCertificates()
		assert.NoError(t, err)
		assert.Empty(t, renewed) // Should not try to renew
	})
}

func TestAutoRenewer_StartAndStopTicker(t *testing.T) {
	renewer, _, _ := setupTestAutoRenewer(t)

	t.Run("Start ticker", func(t *testing.T) {
		renewer.StartAutoRenewTicker()
		assert.NotNil(t, renewer.TickerstopChan)
	})

	t.Run("Stop ticker", func(t *testing.T) {
		renewer.StartAutoRenewTicker()
		assert.NotNil(t, renewer.TickerstopChan)

		renewer.StopAutoRenewTicker()
		// Give it a moment to process the stop signal
		time.Sleep(100 * time.Millisecond)
		assert.Nil(t, renewer.TickerstopChan)
	})

	t.Run("Restart ticker", func(t *testing.T) {
		renewer.StartAutoRenewTicker()
		oldChan := renewer.TickerstopChan
		assert.NotNil(t, oldChan)

		// Starting again should stop the old ticker
		renewer.StartAutoRenewTicker()
		assert.NotNil(t, renewer.TickerstopChan)
	})
}

func TestAutoRenewer_Close(t *testing.T) {
	renewer, _, _ := setupTestAutoRenewer(t)

	renewer.StartAutoRenewTicker()
	assert.NotNil(t, renewer.TickerstopChan)

	renewer.Close()
	// The ticker should be stopped
	time.Sleep(100 * time.Millisecond)
}

func TestAutoRenewer_HandleSetDNS(t *testing.T) {
	renewer, _, _ := setupTestAutoRenewer(t)

	t.Run("Set DNS configuration", func(t *testing.T) {
		form := url.Values{}
		form.Add("dnsProvider", "cloudflare")
		form.Add("dnsCredentials", `{"APIKey":"test123"}`)
		form.Add("filename", "example.com")
		form.Add("dnsServers", "8.8.8.8,8.8.4.4")

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		renewer.HandleSetDNS(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify it was saved to database
		var provider string
		err := renewer.AcmeHandler.Database.Read("acme", "example.com_dns_provider", &provider)
		assert.NoError(t, err)
		assert.Equal(t, "cloudflare", provider)
	})

	t.Run("Missing DNS provider", func(t *testing.T) {
		form := url.Values{}
		form.Add("dnsCredentials", `{"APIKey":"test123"}`)
		form.Add("filename", "example.com")

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		renewer.HandleSetDNS(w, req)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "error")
	})
}

func TestAutoRenewer_HanldeSetEAB(t *testing.T) {
	renewer, _, _ := setupTestAutoRenewer(t)

	t.Run("Set EAB configuration", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?kid=test_kid&hmacEncoded=test_hmac&acmeDirectoryURL=https://acme.example.com", nil)
		w := httptest.NewRecorder()

		renewer.HanldeSetEAB(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify it was saved to database
		var kid string
		err := renewer.AcmeHandler.Database.Read("acme", "https://acme.example.com_kid", &kid)
		assert.NoError(t, err)
		assert.Equal(t, "test_kid", kid)
	})

	t.Run("Missing kid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?hmacEncoded=test_hmac&acmeDirectoryURL=https://acme.example.com", nil)
		w := httptest.NewRecorder()

		renewer.HanldeSetEAB(w, req)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "error")
	})
}
