package dynamicproxy

import (
	"net/http"
	"net/url"
	"strings"
)

// handleCaptchaCheck checks if captcha is enabled and if the client has a valid session
// Returns true if the request has been handled (captcha challenge sent), false otherwise
func (h *ProxyHandler) handleCaptchaCheck(w http.ResponseWriter, r *http.Request) bool {
	// Check if captcha is enabled
	if h.Parent.Option.CaptchaManager == nil || !h.Parent.Option.CaptchaManager.IsEnabled() {
		return false
	}

	// Skip captcha for special paths
	path := r.URL.Path
	if strings.HasPrefix(path, "/.well-known/") ||
		strings.HasPrefix(path, "/.zoraxy/") ||
		strings.HasPrefix(path, "/api/captcha/") {
		return false
	}

	// Check if client has a valid captcha session cookie
	if h.Parent.Option.CaptchaManager.HasValidSessionFromRequest(r) {
		return false
	}

	// Redirect to captcha challenge page
	originalURL := r.URL.String()
	if r.Host != "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		originalURL = scheme + "://" + r.Host + r.URL.String()
	}

	challengeURL := "/.zoraxy/captcha/challenge?redirect=" + url.QueryEscape(originalURL)
	http.Redirect(w, r, challengeURL, http.StatusTemporaryRedirect)

	return true
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	// Remove port if present
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}
