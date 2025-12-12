package captcha

import (
	"encoding/json"
	"net/http"
	"strings"
)

// HandleGetConfig returns the captcha configuration (with secret key hidden)
func (m *Manager) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	config := m.GetConfig()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// HandleUpdateConfig updates the captcha configuration
func (m *Manager) HandleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var config CaptchaConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := m.UpdateConfig(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Captcha configuration updated successfully",
	})
}

// HandleVerify handles captcha verification requests
func (m *Manager) HandleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		http.Error(w, "Token is required", http.StatusBadRequest)
		return
	}

	// Get client IP
	clientIP := getClientIP(r)

	// Verify the token
	success, err := m.VerifyToken(req.Token, clientIP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": success,
	})
}

// HandleChallengePage serves the captcha challenge page
func (m *Manager) HandleChallengePage(w http.ResponseWriter, r *http.Request) {
	provider := m.GetProvider()
	siteKey := m.GetSiteKey()

	// Get the original URL they were trying to access
	originalURL := r.URL.Query().Get("redirect")
	if originalURL == "" {
		originalURL = "/"
	}

	// Generate the challenge page HTML
	html := generateChallengePage(provider, siteKey, originalURL)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// HandleCheckSession checks if a client has a valid captcha session
func (m *Manager) HandleCheckSession(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	hasValidSession := m.HasValidSession(clientIP)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": hasValidSession,
	})
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

// generateChallengePage generates the HTML for the captcha challenge page
func generateChallengePage(provider CaptchaProvider, siteKey string, redirectURL string) string {
	var providerScript, providerWidget string

	// Escape the redirect URL for safe embedding in JavaScript
	escapedRedirectURL := strings.Replace(redirectURL, `\`, `\\`, -1)
	escapedRedirectURL = strings.Replace(escapedRedirectURL, `"`, `\"`, -1)
	escapedRedirectURL = strings.Replace(escapedRedirectURL, "\n", `\n`, -1)
	escapedRedirectURL = strings.Replace(escapedRedirectURL, "\r", `\r`, -1)

	switch provider {
	case ProviderGoogleRecaptcha:
		providerScript = `<script src="https://www.google.com/recaptcha/api.js" async defer></script>`
		providerWidget = `<div class="g-recaptcha" data-sitekey="` + siteKey + `" data-callback="onCaptchaSuccess"></div>`
	case ProviderCloudflareTurnstile:
		providerScript = `<script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>`
		providerWidget = `<div class="cf-turnstile" data-sitekey="` + siteKey + `" data-callback="onCaptchaSuccess"></div>`
	default:
		return "<html><body><h1>Captcha not configured</h1></body></html>"
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Security Verification - Zoraxy</title>
    ` + providerScript + `
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }

        .container {
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            max-width: 500px;
            width: 100%;
            padding: 40px;
            text-align: center;
        }

        .logo {
            width: 80px;
            height: 80px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            border-radius: 50%;
            margin: 0 auto 24px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 36px;
            color: white;
            font-weight: bold;
        }

        h1 {
            color: #1a202c;
            font-size: 28px;
            margin-bottom: 12px;
            font-weight: 700;
        }

        p {
            color: #718096;
            font-size: 16px;
            line-height: 1.6;
            margin-bottom: 32px;
        }

        .captcha-container {
            display: flex;
            justify-content: center;
            margin: 32px 0;
            min-height: 78px;
        }

        .loading {
            display: none;
            color: #667eea;
            font-size: 16px;
            margin-top: 20px;
        }

        .loading.active {
            display: block;
        }

        .spinner {
            border: 3px solid #f3f3f3;
            border-top: 3px solid #667eea;
            border-radius: 50%;
            width: 40px;
            height: 40px;
            animation: spin 1s linear infinite;
            margin: 0 auto 12px;
        }

        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }

        .footer {
            margin-top: 32px;
            padding-top: 24px;
            border-top: 1px solid #e2e8f0;
            color: #a0aec0;
            font-size: 14px;
        }

        .error {
            display: none;
            background: #fff5f5;
            border: 1px solid #fc8181;
            color: #c53030;
            padding: 12px 16px;
            border-radius: 8px;
            margin-top: 20px;
            font-size: 14px;
        }

        .error.active {
            display: block;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo">Z</div>
        <h1>Security Verification</h1>
        <p>Please complete the security check below to continue to your destination.</p>

        <div class="captcha-container">
            ` + providerWidget + `
        </div>

        <div class="loading" id="loading">
            <div class="spinner"></div>
            <p>Verifying...</p>
        </div>

        <div class="error" id="error">
            Verification failed. Please try again.
        </div>

        <div class="footer">
            <p>Protected by Zoraxy</p>
        </div>
    </div>

    <script>
        const redirectUrl = "` + escapedRedirectURL + `";

        function onCaptchaSuccess(token) {
            document.getElementById('loading').classList.add('active');
            document.getElementById('error').classList.remove('active');

            // Send verification request to backend
            fetch('/api/captcha/verify', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ token: token })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    // Redirect to original URL
                    window.location.href = redirectUrl;
                } else {
                    document.getElementById('loading').classList.remove('active');
                    document.getElementById('error').classList.add('active');
                    // Reset captcha
                    if (typeof grecaptcha !== 'undefined') {
                        grecaptcha.reset();
                    } else if (typeof turnstile !== 'undefined') {
                        turnstile.reset();
                    }
                }
            })
            .catch(error => {
                console.error('Error:', error);
                document.getElementById('loading').classList.remove('active');
                document.getElementById('error').classList.add('active');
                // Reset captcha
                if (typeof grecaptcha !== 'undefined') {
                    grecaptcha.reset();
                } else if (typeof turnstile !== 'undefined') {
                    turnstile.reset();
                }
            });
        }
    </script>
</body>
</html>`
}
