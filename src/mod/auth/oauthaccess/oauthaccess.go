package oauthaccess

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"imuslab.com/zoraxy/mod/database"
	"imuslab.com/zoraxy/mod/info/logger"
)

// OAuthProvider represents the OAuth provider configuration
type OAuthProvider struct {
	ID                 string   `json:"id"`                   // Unique identifier
	Name               string   `json:"name"`                 // Display name
	Enabled            bool     `json:"enabled"`              // Whether this provider is enabled
	ClientID           string   `json:"client_id"`            // OAuth client ID
	ClientSecret       string   `json:"client_secret"`        // OAuth client secret
	AuthURL            string   `json:"auth_url"`             // Authorization endpoint
	TokenURL           string   `json:"token_url"`            // Token endpoint
	UserInfoURL        string   `json:"userinfo_url"`         // User info endpoint
	Scopes             []string `json:"scopes"`               // OAuth scopes
	RedirectURL        string   `json:"redirect_url"`         // Callback URL
	AllowedDomains     []string `json:"allowed_domains"`      // Allowed email domains (empty = all)
	AllowedEmails      []string `json:"allowed_emails"`       // Allowed specific emails (empty = all)
	SessionDuration    int      `json:"session_duration"`     // Session duration in seconds
	UseStateCookie     bool     `json:"use_state_cookie"`     // Use cookie for state validation (more secure)
	AllowRefresh       bool     `json:"allow_refresh"`        // Allow token refresh
	RequireEmailVerify bool     `json:"require_email_verify"` // Require verified email
}

// OAuthSession represents an authenticated OAuth session
type OAuthSession struct {
	SessionID     string    `json:"session_id"`
	ProviderID    string    `json:"provider_id"`
	UserEmail     string    `json:"user_email"`
	UserID        string    `json:"user_id"`
	UserName      string    `json:"user_name"`
	AccessToken   string    `json:"access_token"`
	RefreshToken  string    `json:"refresh_token,omitempty"`
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
	LastAccessedAt time.Time `json:"last_accessed_at"`
}

// OAuthState stores temporary OAuth state for CSRF protection
type OAuthState struct {
	State        string
	ProviderID   string
	OriginalURL  string
	CreatedAt    time.Time
	CodeVerifier string // For PKCE
}

// Manager handles OAuth Access authentication
type Manager struct {
	db        *database.Database
	logger    *logger.Logger
	providers map[string]*OAuthProvider
	sessions  map[string]*OAuthSession // Key: session ID
	states    map[string]*OAuthState   // Key: state token
	mu        sync.RWMutex
	client    *http.Client
}

// UserInfo represents the user information from OAuth provider
type UserInfo struct {
	ID            string `json:"id,omitempty"`
	Sub           string `json:"sub,omitempty"` // OIDC standard
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name,omitempty"`
	Picture       string `json:"picture,omitempty"`
}

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

// NewManager creates a new OAuth Access manager
func NewManager(db *database.Database, logger *logger.Logger) (*Manager, error) {
	m := &Manager{
		db:        db,
		logger:    logger,
		providers: make(map[string]*OAuthProvider),
		sessions:  make(map[string]*OAuthSession),
		states:    make(map[string]*OAuthState),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Load providers from database
	if err := m.loadProviders(); err != nil {
		logger.PrintAndLog("oauth-access", "Failed to load providers, starting fresh", err)
	}

	// Start cleanup goroutines
	go m.cleanupExpiredSessions()
	go m.cleanupExpiredStates()

	return m, nil
}

// loadProviders loads OAuth providers from database
func (m *Manager) loadProviders() error {
	var providers []*OAuthProvider
	err := m.db.Read("oauth_access", "providers", &providers)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range providers {
		m.providers[p.ID] = p
	}

	return nil
}

// saveProviders saves OAuth providers to database
func (m *Manager) saveProviders() error {
	m.mu.RLock()
	providers := make([]*OAuthProvider, 0, len(m.providers))
	for _, p := range m.providers {
		providers = append(providers, p)
	}
	m.mu.RUnlock()

	return m.db.Write("oauth_access", "providers", providers)
}

// AddProvider adds or updates an OAuth provider
func (m *Manager) AddProvider(provider *OAuthProvider) error {
	if provider.ID == "" {
		return errors.New("provider ID is required")
	}
	if provider.ClientID == "" || provider.ClientSecret == "" {
		return errors.New("client ID and secret are required")
	}
	if provider.AuthURL == "" || provider.TokenURL == "" {
		return errors.New("auth URL and token URL are required")
	}

	// Set defaults
	if provider.SessionDuration == 0 {
		provider.SessionDuration = 3600 // 1 hour default
	}
	if len(provider.Scopes) == 0 {
		provider.Scopes = []string{"openid", "email", "profile"}
	}

	m.mu.Lock()
	m.providers[provider.ID] = provider
	m.mu.Unlock()

	return m.saveProviders()
}

// GetProvider returns a provider by ID
func (m *Manager) GetProvider(id string) (*OAuthProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	provider, ok := m.providers[id]
	if !ok {
		return nil, errors.New("provider not found")
	}

	// Return a copy with secret hidden
	p := *provider
	p.ClientSecret = ""
	return &p, nil
}

// ListProviders returns all providers (with secrets hidden)
func (m *Manager) ListProviders() []*OAuthProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providers := make([]*OAuthProvider, 0, len(m.providers))
	for _, p := range m.providers {
		provider := *p
		provider.ClientSecret = "" // Hide secret
		providers = append(providers, &provider)
	}

	return providers
}

// DeleteProvider deletes a provider
func (m *Manager) DeleteProvider(id string) error {
	m.mu.Lock()
	delete(m.providers, id)
	m.mu.Unlock()

	return m.saveProviders()
}

// GetEnabledProvider returns the first enabled provider, or nil if none
func (m *Manager) GetEnabledProvider() *OAuthProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.providers {
		if p.Enabled {
			return p
		}
	}
	return nil
}

// generateState generates a random state token
func (m *Manager) generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// generateSessionID generates a random session ID
func (m *Manager) generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// createState creates a new OAuth state for CSRF protection
func (m *Manager) createState(providerID, originalURL string) string {
	state := m.generateState()

	m.mu.Lock()
	m.states[state] = &OAuthState{
		State:       state,
		ProviderID:  providerID,
		OriginalURL: originalURL,
		CreatedAt:   time.Now(),
	}
	m.mu.Unlock()

	return state
}

// getState retrieves and removes an OAuth state
func (m *Manager) getState(state string) (*OAuthState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.states[state]
	if !ok {
		return nil, errors.New("invalid or expired state")
	}

	// Check if state is expired (10 minutes)
	if time.Since(s.CreatedAt) > 10*time.Minute {
		delete(m.states, state)
		return nil, errors.New("state expired")
	}

	// Remove state after use
	delete(m.states, state)

	return s, nil
}

// GetAuthorizationURL generates the OAuth authorization URL
func (m *Manager) GetAuthorizationURL(providerID, originalURL string) (string, error) {
	m.mu.RLock()
	provider, ok := m.providers[providerID]
	m.mu.RUnlock()

	if !ok || !provider.Enabled {
		return "", errors.New("provider not found or disabled")
	}

	state := m.createState(providerID, originalURL)

	params := url.Values{}
	params.Set("client_id", provider.ClientID)
	params.Set("redirect_uri", provider.RedirectURL)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(provider.Scopes, " "))
	params.Set("state", state)

	authURL := provider.AuthURL + "?" + params.Encode()
	return authURL, nil
}

// ExchangeCodeForToken exchanges authorization code for access token
func (m *Manager) ExchangeCodeForToken(code, state string) (*OAuthSession, error) {
	// Validate state
	stateObj, err := m.getState(state)
	if err != nil {
		return nil, fmt.Errorf("invalid state: %w", err)
	}

	m.mu.RLock()
	provider, ok := m.providers[stateObj.ProviderID]
	m.mu.RUnlock()

	if !ok {
		return nil, errors.New("provider not found")
	}

	// Exchange code for token
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", provider.RedirectURL)
	data.Set("client_id", provider.ClientID)
	data.Set("client_secret", provider.ClientSecret)

	// Create request with JSON accept header (GitHub and some providers need this)
	req, err := http.NewRequest("POST", provider.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Try to decode the response as JSON first
	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		// If JSON parsing fails, try URL-encoded format (GitHub uses this by default)
		values, parseErr := url.ParseQuery(string(body))
		if parseErr != nil {
			return nil, fmt.Errorf("failed to decode token response as JSON or form-encoded: %w. Response body: %s", err, string(body))
		}

		// Parse form-encoded response
		tokenResp.AccessToken = values.Get("access_token")
		tokenResp.TokenType = values.Get("token_type")
		tokenResp.RefreshToken = values.Get("refresh_token")
		tokenResp.Scope = values.Get("scope")

		if tokenResp.AccessToken == "" {
			return nil, fmt.Errorf("no access token in response. Response body: %s", string(body))
		}
	}

	// Get user info
	userInfo, err := m.getUserInfo(provider, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Validate user
	if err := m.validateUser(provider, userInfo); err != nil {
		return nil, err
	}

	// Create session
	sessionID := m.generateSessionID()
	expiresAt := time.Now().Add(time.Duration(provider.SessionDuration) * time.Second)

	session := &OAuthSession{
		SessionID:      sessionID,
		ProviderID:     provider.ID,
		UserEmail:      userInfo.Email,
		UserID:         getUserID(userInfo),
		UserName:       userInfo.Name,
		AccessToken:    tokenResp.AccessToken,
		RefreshToken:   tokenResp.RefreshToken,
		ExpiresAt:      expiresAt,
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	return session, nil
}

// getUserInfo retrieves user information from the OAuth provider
func (m *Manager) getUserInfo(provider *OAuthProvider, accessToken string) (*UserInfo, error) {
	if provider.UserInfoURL == "" {
		return nil, errors.New("user info URL not configured")
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", provider.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user info: %s", string(body))
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// getUserID extracts user ID from UserInfo (handles both 'id' and 'sub' fields)
func getUserID(userInfo *UserInfo) string {
	if userInfo.ID != "" {
		return userInfo.ID
	}
	if userInfo.Sub != "" {
		return userInfo.Sub
	}
	return userInfo.Email // Fallback to email
}

// validateUser validates if the user is allowed access
func (m *Manager) validateUser(provider *OAuthProvider, userInfo *UserInfo) error {
	if provider.RequireEmailVerify && !userInfo.EmailVerified {
		return errors.New("email not verified")
	}

	// Check allowed domains
	if len(provider.AllowedDomains) > 0 {
		emailDomain := getEmailDomain(userInfo.Email)
		allowed := false
		for _, domain := range provider.AllowedDomains {
			if strings.EqualFold(emailDomain, domain) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("email domain %s not allowed", emailDomain)
		}
	}

	// Check allowed emails
	if len(provider.AllowedEmails) > 0 {
		allowed := false
		for _, email := range provider.AllowedEmails {
			if strings.EqualFold(userInfo.Email, email) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("email %s not allowed", userInfo.Email)
		}
	}

	return nil
}

// getEmailDomain extracts the domain from an email address
func getEmailDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

// GetSession retrieves a session by ID
func (m *Manager) GetSession(sessionID string) (*OAuthSession, error) {
	m.mu.RLock()
	session, ok := m.sessions[sessionID]
	m.mu.RUnlock()

	if !ok {
		return nil, errors.New("session not found")
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		m.DeleteSession(sessionID)
		return nil, errors.New("session expired")
	}

	// Update last accessed time
	m.mu.Lock()
	session.LastAccessedAt = time.Now()
	m.mu.Unlock()

	return session, nil
}

// DeleteSession deletes a session
func (m *Manager) DeleteSession(sessionID string) {
	m.mu.Lock()
	delete(m.sessions, sessionID)
	m.mu.Unlock()
}

// cleanupExpiredSessions periodically removes expired sessions
func (m *Manager) cleanupExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for id, session := range m.sessions {
			if now.After(session.ExpiresAt) {
				delete(m.sessions, id)
			}
		}
		m.mu.Unlock()
	}
}

// cleanupExpiredStates periodically removes expired OAuth states
func (m *Manager) cleanupExpiredStates() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for state, s := range m.states {
			if now.Sub(s.CreatedAt) > 10*time.Minute {
				delete(m.states, state)
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

// Close cleanly shuts down the manager
func (m *Manager) Close() {
	// Save any pending data if needed
}
