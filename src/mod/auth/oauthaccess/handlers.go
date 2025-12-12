package oauthaccess

import (
	"encoding/json"
	"net/http"
	"time"
)

const (
	SessionCookieName = "zoraxy_oauth_session"
)

// HandleListProviders returns all configured OAuth providers
func (m *Manager) HandleListProviders(w http.ResponseWriter, r *http.Request) {
	providers := m.ListProviders()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers)
}

// HandleGetProvider returns a specific OAuth provider
func (m *Manager) HandleGetProvider(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Provider ID is required", http.StatusBadRequest)
		return
	}

	provider, err := m.GetProvider(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(provider)
}

// HandleAddProvider adds or updates an OAuth provider
func (m *Manager) HandleAddProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var provider OAuthProvider
	if err := json.NewDecoder(r.Body).Decode(&provider); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := m.AddProvider(&provider); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Provider added successfully",
	})
}

// HandleDeleteProvider deletes an OAuth provider
func (m *Manager) HandleDeleteProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Provider ID is required", http.StatusBadRequest)
		return
	}

	if err := m.DeleteProvider(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Provider deleted successfully",
	})
}

// HandleLogin initiates the OAuth login flow
func (m *Manager) HandleLogin(w http.ResponseWriter, r *http.Request) {
	providerID := r.URL.Query().Get("provider")
	if providerID == "" {
		// Use the first enabled provider
		provider := m.GetEnabledProvider()
		if provider == nil {
			http.Error(w, "No OAuth provider configured", http.StatusServiceUnavailable)
			return
		}
		providerID = provider.ID
	}

	// Get the original URL to redirect back to after authentication
	originalURL := r.URL.Query().Get("redirect")
	if originalURL == "" {
		originalURL = r.Referer()
		if originalURL == "" {
			originalURL = "/"
		}
	}

	// Generate authorization URL
	authURL, err := m.GetAuthorizationURL(providerID, originalURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect to OAuth provider
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// HandleCallback handles the OAuth callback
func (m *Manager) HandleCallback(w http.ResponseWriter, r *http.Request) {
	// Get authorization code and state from query parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		http.Error(w, "Missing code or state parameter", http.StatusBadRequest)
		return
	}

	// Exchange code for token and create session
	session, err := m.ExchangeCodeForToken(code, state)
	if err != nil {
		m.logger.PrintAndLog("oauth-access", "Failed to exchange code for token", err)
		http.Error(w, "Authentication failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    session.SessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		Expires:  session.ExpiresAt,
	})

	// Get the original URL from state
	m.mu.RLock()
	stateObj, ok := m.states[state]
	m.mu.RUnlock()

	redirectURL := "/"
	if ok && stateObj.OriginalURL != "" {
		redirectURL = stateObj.OriginalURL
	}

	// Redirect to original URL
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// HandleLogout logs out the user
func (m *Manager) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Get session cookie
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil {
		// Delete session
		m.DeleteSession(cookie.Value)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	// Redirect to home or specified URL
	redirectURL := r.URL.Query().Get("redirect")
	if redirectURL == "" {
		redirectURL = "/"
	}

	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// HandleCheckSession checks if the user has a valid session
func (m *Manager) HandleCheckSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	session, err := m.GetSession(cookie.Value)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": true,
		"user_email":    session.UserEmail,
		"user_name":     session.UserName,
		"expires_at":    session.ExpiresAt,
	})
}

// HandleGetStats returns statistics about OAuth Access usage
func (m *Manager) HandleGetStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"active_sessions": m.GetSessionCount(),
		"providers_count": len(m.ListProviders()),
	})
}

// RequireAuth is a middleware that requires OAuth authentication
func (m *Manager) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil {
			// No session cookie, redirect to login
			loginURL := "/.zoraxy/oauth/login?redirect=" + r.URL.String()
			http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
			return
		}

		_, err = m.GetSession(cookie.Value)
		if err != nil {
			// Invalid or expired session, redirect to login
			loginURL := "/.zoraxy/oauth/login?redirect=" + r.URL.String()
			http.Redirect(w, r, loginURL, http.StatusTemporaryRedirect)
			return
		}

		// Add user info to request context if needed
		// session, _ := m.GetSession(cookie.Value)
		// context.WithValue(r.Context(), "user_email", session.UserEmail)

		next(w, r)
	}
}

// CheckAuth checks if a request has valid OAuth authentication
// Returns the session if authenticated, nil otherwise
func (m *Manager) CheckAuth(r *http.Request) *OAuthSession {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil
	}

	session, err := m.GetSession(cookie.Value)
	if err != nil {
		return nil
	}

	return session
}
