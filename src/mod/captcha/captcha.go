package captcha

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"imuslab.com/zoraxy/mod/database"
)

// CaptchaProvider represents the type of captcha provider
type CaptchaProvider string

const (
	ProviderGoogleRecaptcha   CaptchaProvider = "google_recaptcha"
	ProviderCloudflareTurnstile CaptchaProvider = "cloudflare_turnstile"
	ProviderDisabled          CaptchaProvider = "disabled"

	// Cookie name for captcha session
	CaptchaSessionCookie = "zoraxy_captcha_session"
)

// CaptchaConfig stores the configuration for captcha providers
type CaptchaConfig struct {
	Enabled    bool            `json:"enabled"`
	Provider   CaptchaProvider `json:"provider"`
	SiteKey    string          `json:"site_key"`
	SecretKey  string          `json:"secret_key"`
	Threshold  float64         `json:"threshold,omitempty"` // For reCAPTCHA v3 score threshold (0.0-1.0)
	ExpiryTime int             `json:"expiry_time"`         // Session expiry in seconds (default: 3600)
}

// CaptchaSession stores validated captcha sessions
type CaptchaSession struct {
	SessionID  string
	ValidUntil time.Time
	Provider   CaptchaProvider
}

// Manager handles captcha verification and session management
type Manager struct {
	config   *CaptchaConfig
	db       *database.Database
	sessions map[string]*CaptchaSession // Key: Session ID (from cookie)
	mu       sync.RWMutex
	client   *http.Client
}

// GoogleRecaptchaResponse represents the response from Google reCAPTCHA API
type GoogleRecaptchaResponse struct {
	Success     bool     `json:"success"`
	Score       float64  `json:"score,omitempty"`       // v3 only
	Action      string   `json:"action,omitempty"`      // v3 only
	ChallengeTs string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes,omitempty"`
}

// CloudflareTurnstileResponse represents the response from Cloudflare Turnstile API
type CloudflareTurnstileResponse struct {
	Success     bool     `json:"success"`
	ChallengeTs string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes,omitempty"`
	Action      string   `json:"action,omitempty"`
	Cdata       string   `json:"cdata,omitempty"`
}

// NewManager creates a new captcha manager
func NewManager(db *database.Database) (*Manager, error) {
	m := &Manager{
		db:       db,
		sessions: make(map[string]*CaptchaSession),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Load config from database
	if err := m.LoadConfig(); err != nil {
		// If config doesn't exist, create default
		m.config = &CaptchaConfig{
			Enabled:    false,
			Provider:   ProviderDisabled,
			ExpiryTime: 3600, // 1 hour default
			Threshold:  0.5,  // Default reCAPTCHA v3 threshold
		}
		if err := m.SaveConfig(); err != nil {
			return nil, err
		}
	}

	// Start cleanup goroutine
	go m.cleanupExpiredSessions()

	return m, nil
}

// LoadConfig loads captcha configuration from database
func (m *Manager) LoadConfig() error {
	var config CaptchaConfig
	err := m.db.Read("captcha", "config", &config)
	if err != nil {
		return err
	}
	m.config = &config
	return nil
}

// SaveConfig saves captcha configuration to database
func (m *Manager) SaveConfig() error {
	return m.db.Write("captcha", "config", m.config)
}

// GetConfig returns the current captcha configuration (with secret key hidden)
func (m *Manager) GetConfig() CaptchaConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config := *m.config
	// Hide secret key for security
	config.SecretKey = ""
	return config
}

// UpdateConfig updates the captcha configuration
func (m *Manager) UpdateConfig(config *CaptchaConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate configuration
	if config.Enabled {
		if config.Provider == ProviderDisabled {
			return errors.New("provider must be specified when captcha is enabled")
		}
		if config.SiteKey == "" || config.SecretKey == "" {
			return errors.New("site key and secret key are required")
		}
		if config.Provider == ProviderGoogleRecaptcha && (config.Threshold < 0 || config.Threshold > 1) {
			return errors.New("threshold must be between 0.0 and 1.0")
		}
	}

	if config.ExpiryTime <= 0 {
		config.ExpiryTime = 3600 // Default 1 hour
	}

	m.config = config
	return m.SaveConfig()
}

// IsEnabled returns whether captcha is enabled
func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Enabled
}

// GetProvider returns the current captcha provider
func (m *Manager) GetProvider() CaptchaProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Provider
}

// GetSiteKey returns the site key for the frontend
func (m *Manager) GetSiteKey() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.SiteKey
}

// VerifyToken verifies a captcha token from the client and returns a session ID
func (m *Manager) VerifyToken(token string, clientIP string) (string, error) {
	m.mu.RLock()
	provider := m.config.Provider
	secretKey := m.config.SecretKey
	threshold := m.config.Threshold
	m.mu.RUnlock()

	var success bool
	var err error

	switch provider {
	case ProviderGoogleRecaptcha:
		success, err = m.verifyGoogleRecaptcha(token, clientIP, secretKey, threshold)
	case ProviderCloudflareTurnstile:
		success, err = m.verifyCloudfllareTurnstile(token, clientIP, secretKey)
	default:
		return "", errors.New("invalid captcha provider")
	}

	if err != nil {
		return "", err
	}

	if success {
		// Create session and return session ID
		sessionID := m.createSession(provider)
		return sessionID, nil
	}

	return "", errors.New("captcha verification failed")
}

// verifyGoogleRecaptcha verifies a Google reCAPTCHA token
func (m *Manager) verifyGoogleRecaptcha(token, clientIP, secretKey string, threshold float64) (bool, error) {
	verifyURL := "https://www.google.com/recaptcha/api/siteverify"

	data := url.Values{}
	data.Set("secret", secretKey)
	data.Set("response", token)
	data.Set("remoteip", clientIP)

	resp, err := m.client.PostForm(verifyURL, data)
	if err != nil {
		return false, fmt.Errorf("failed to verify recaptcha: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read recaptcha response: %w", err)
	}

	var result GoogleRecaptchaResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("failed to parse recaptcha response: %w", err)
	}

	// For reCAPTCHA v3, check the score
	if result.Score > 0 {
		return result.Success && result.Score >= threshold, nil
	}

	// For reCAPTCHA v2
	return result.Success, nil
}

// verifyCloudfllareTurnstile verifies a Cloudflare Turnstile token
func (m *Manager) verifyCloudfllareTurnstile(token, clientIP, secretKey string) (bool, error) {
	verifyURL := "https://challenges.cloudflare.com/turnstile/v0/siteverify"

	data := url.Values{}
	data.Set("secret", secretKey)
	data.Set("response", token)
	data.Set("remoteip", clientIP)

	resp, err := m.client.PostForm(verifyURL, data)
	if err != nil {
		return false, fmt.Errorf("failed to verify turnstile: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read turnstile response: %w", err)
	}

	var result CloudflareTurnstileResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("failed to parse turnstile response: %w", err)
	}

	return result.Success, nil
}

// generateSessionID generates a random session ID
func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// createSession creates a validated captcha session and returns the session ID
func (m *Manager) createSession(provider CaptchaProvider) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	sessionID := generateSessionID()
	m.sessions[sessionID] = &CaptchaSession{
		SessionID:  sessionID,
		ValidUntil: time.Now().Add(time.Duration(m.config.ExpiryTime) * time.Second),
		Provider:   provider,
	}

	return sessionID
}

// HasValidSession checks if a session ID has a valid captcha session
func (m *Manager) HasValidSession(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return false
	}

	return time.Now().Before(session.ValidUntil)
}

// HasValidSessionFromRequest checks if the request has a valid captcha session cookie
func (m *Manager) HasValidSessionFromRequest(r *http.Request) bool {
	cookie, err := r.Cookie(CaptchaSessionCookie)
	if err != nil {
		return false
	}

	return m.HasValidSession(cookie.Value)
}

// InvalidateSession removes a captcha session for a session ID
func (m *Manager) InvalidateSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, sessionID)
}

// cleanupExpiredSessions periodically removes expired sessions
func (m *Manager) cleanupExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for sessionID, session := range m.sessions {
			if now.After(session.ValidUntil) {
				delete(m.sessions, sessionID)
			}
		}
		m.mu.Unlock()
	}
}

// GetSessionCount returns the number of active sessions
func (m *Manager) GetSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// Close cleanly shuts down the captcha manager
func (m *Manager) Close() {
	// Nothing to clean up currently
}
