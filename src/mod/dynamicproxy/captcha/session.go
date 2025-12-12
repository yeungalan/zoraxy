package captcha

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

/*
	session.go

	This file handles CAPTCHA session management.
	After a user successfully completes a CAPTCHA challenge,
	a session is created to allow them access for a period of time.
*/

// SessionManager manages verified CAPTCHA sessions
type SessionManager struct {
	sessions map[string]*Session
	mutex    sync.RWMutex
}

// Session represents a verified CAPTCHA session
type Session struct {
	Token      string
	IP         string
	Endpoint   string
	CreatedAt  time.Time
	ExpiresAt  time.Time
}

// NewSessionManager creates a new CAPTCHA session manager
func NewSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
	}

	// Start cleanup goroutine
	go sm.cleanupExpiredSessions()

	return sm
}

// CreateSession creates a new verified session
func (sm *SessionManager) CreateSession(ip, endpoint string, ttl int) (*Session, error) {
	token, err := generateSessionToken()
	if err != nil {
		return nil, err
	}

	session := &Session{
		Token:     token,
		IP:        ip,
		Endpoint:  endpoint,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Duration(ttl) * time.Second),
	}

	sm.mutex.Lock()
	sm.sessions[token] = session
	sm.mutex.Unlock()

	return session, nil
}

// ValidateSession checks if a session token is valid
func (sm *SessionManager) ValidateSession(token, ip, endpoint string) bool {
	if token == "" {
		return false
	}

	sm.mutex.RLock()
	session, exists := sm.sessions[token]
	sm.mutex.RUnlock()

	if !exists {
		return false
	}

	// Check if session has expired
	if time.Now().After(session.ExpiresAt) {
		sm.RemoveSession(token)
		return false
	}

	// Check if IP and endpoint match
	if session.IP != ip || session.Endpoint != endpoint {
		return false
	}

	return true
}

// RemoveSession removes a session
func (sm *SessionManager) RemoveSession(token string) {
	sm.mutex.Lock()
	delete(sm.sessions, token)
	sm.mutex.Unlock()
}

// cleanupExpiredSessions periodically removes expired sessions
func (sm *SessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sm.mutex.Lock()
		now := time.Now()
		for token, session := range sm.sessions {
			if now.After(session.ExpiresAt) {
				delete(sm.sessions, token)
			}
		}
		sm.mutex.Unlock()
	}
}

// generateSessionToken generates a cryptographically secure random token
func generateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
