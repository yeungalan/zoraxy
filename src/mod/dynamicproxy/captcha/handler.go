package captcha

import (
	_ "embed"
	"encoding/json"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

/*
	handler.go

	This file handles CAPTCHA challenge presentation and verification
*/

//go:embed templates/challenge.html
var challengeTemplate string

const (
	CaptchaCookieName = "zoraxy_captcha_session"
	DefaultSessionTTL = 3600 // 1 hour
)

// Handler handles CAPTCHA challenges
type Handler struct {
	SessionManager *SessionManager
	template       *template.Template
}

// NewHandler creates a new CAPTCHA handler
func NewHandler() (*Handler, error) {
	tmpl, err := template.New("challenge").Parse(challengeTemplate)
	if err != nil {
		return nil, err
	}

	return &Handler{
		SessionManager: NewSessionManager(),
		template:       tmpl,
	}, nil
}

// ChallengeData holds data for rendering the challenge page
type ChallengeData struct {
	Provider    string
	SiteKey     string
	VerifyURL   string
	RedirectURL string
}

// ShowChallenge displays the CAPTCHA challenge page
func (h *Handler) ShowChallenge(w http.ResponseWriter, r *http.Request, provider int, siteKey string, redirectURL string) {
	// Determine provider type
	providerName := "cloudflare"
	if provider == 1 { // Google reCAPTCHA
		providerName = "google"
	}

	// Create verify URL (same host, special endpoint)
	verifyURL := "/.zoraxy/captcha/verify"

	data := ChallengeData{
		Provider:    providerName,
		SiteKey:     siteKey,
		VerifyURL:   verifyURL,
		RedirectURL: redirectURL,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if err := h.template.Execute(w, data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// VerifyResponse handles CAPTCHA verification
type VerifyRequest struct {
	CaptchaResponse string `json:"captcha_response"`
	RedirectURL     string `json:"redirect_url"`
}

type VerifyResponse struct {
	Success     bool   `json:"success"`
	RedirectURL string `json:"redirect_url,omitempty"`
	Error       string `json:"error,omitempty"`
}

// HandleVerify processes CAPTCHA verification requests
func (h *Handler) HandleVerify(w http.ResponseWriter, r *http.Request, provider int, secretKey string, endpoint string, sessionTTL int) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		sendJSONError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	captchaResponse := r.FormValue("captcha_response")
	redirectURL := r.FormValue("redirect_url")

	if captchaResponse == "" {
		sendJSONError(w, "CAPTCHA response is required", http.StatusBadRequest)
		return
	}

	// Get client IP
	clientIP := getClientIP(r)

	// Verify CAPTCHA with provider
	var verified bool
	var err error

	if provider == 0 { // Cloudflare Turnstile
		verified, err = VerifyCloudflare(secretKey, captchaResponse, clientIP)
	} else { // Google reCAPTCHA
		verified, err = VerifyGoogle(secretKey, captchaResponse, clientIP)
	}

	if err != nil || !verified {
		errorMsg := "Verification failed"
		if err != nil {
			errorMsg = err.Error()
		}
		sendJSONError(w, errorMsg, http.StatusForbidden)
		return
	}

	// Create session
	if sessionTTL <= 0 {
		sessionTTL = DefaultSessionTTL
	}

	session, err := h.SessionManager.CreateSession(clientIP, endpoint, sessionTTL)
	if err != nil {
		sendJSONError(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	cookie := &http.Cookie{
		Name:     CaptchaCookieName,
		Value:    session.Token,
		Path:     "/",
		MaxAge:   sessionTTL,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	// Send success response
	response := VerifyResponse{
		Success:     true,
		RedirectURL: redirectURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CheckSession validates if a request has a valid CAPTCHA session
func (h *Handler) CheckSession(r *http.Request, endpoint string) bool {
	cookie, err := r.Cookie(CaptchaCookieName)
	if err != nil {
		return false
	}

	clientIP := getClientIP(r)
	return h.SessionManager.ValidateSession(cookie.Value, clientIP, endpoint)
}

// sendJSONError sends a JSON error response
func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(VerifyResponse{
		Success: false,
		Error:   message,
	})
}

// getClientIP extracts the real client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// ShouldBypassCaptcha checks if a request path should bypass CAPTCHA
func ShouldBypassCaptcha(requestPath string, bypassPaths []string) bool {
	// Always bypass CAPTCHA verification endpoint
	if strings.HasPrefix(requestPath, "/.zoraxy/captcha/") {
		return true
	}

	// Check user-defined bypass paths
	for _, bypassPath := range bypassPaths {
		if strings.HasPrefix(requestPath, bypassPath) {
			return true
		}
	}

	return false
}

// GetRedirectURL constructs the redirect URL after verification
func GetRedirectURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	// Get the original URL
	originalURL := r.URL.String()
	if originalURL == "" {
		originalURL = "/"
	}

	// URL encode it
	return url.QueryEscape(originalURL)
}
