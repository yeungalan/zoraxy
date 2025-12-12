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

	// Get client IP for captcha provider verification
	clientIP := getClientIP(r)

	// Verify the token and get session ID
	sessionID, err := m.VerifyToken(req.Token, clientIP)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     CaptchaSessionCookie,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   m.config.ExpiryTime,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
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
	hasValidSession := m.HasValidSessionFromRequest(r)

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
<html lang="en-US">
<head>
    <meta charset="UTF-8">
    <meta http-equiv="X-UA-Compatible" content="IE=Edge">
    <meta name="robots" content="noindex,nofollow">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Just a moment...</title>
    ` + providerScript + `
    <style>
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        html {
            line-height: 1.15;
            -webkit-text-size-adjust: 100%;
            color: #313131;
            font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, "Noto Sans", sans-serif, "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol", "Noto Color Emoji";
        }

        body {
            display: flex;
            flex-direction: column;
            min-height: 100vh;
            background-color: #fff;
        }

        .main-wrapper {
            flex: 1;
            display: flex;
            align-items: center;
        }

        .main-content {
            margin: 8rem auto;
            padding: 0 1.5rem;
            max-width: 60rem;
            width: 100%;
        }

        @media (max-width: 720px) {
            .main-content {
                margin-top: 4rem;
            }
        }

        .h2 {
            line-height: 2.25rem;
            font-size: 1.5rem;
            font-weight: 500;
            margin-bottom: 2rem;
        }

        @media (max-width: 720px) {
            .h2 {
                line-height: 1.5rem;
                font-size: 1.25rem;
            }
        }

        .captcha-container {
            display: flex;
            justify-content: center;
            margin: 2rem 0;
        }

        .loading {
            display: none;
            text-align: center;
            margin-top: 2rem;
        }

        .loading.active {
            display: block;
        }

        .spinner {
            border: 3px solid #f3f3f3;
            border-top: 3px solid #555;
            border-radius: 50%;
            width: 40px;
            height: 40px;
            animation: spin 1s linear infinite;
            margin: 0 auto 1rem;
        }

        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }

        .error {
            display: none;
            background-color: #fee;
            border: 1px solid #fcc;
            color: #c00;
            padding: 1rem;
            border-radius: 4px;
            margin-top: 1rem;
        }

        .error.active {
            display: block;
        }

        .footer {
            text-align: center;
            padding: 2rem 1.5rem;
            color: #999;
            font-size: 0.875rem;
        }

        @media (prefers-color-scheme: dark) {
            html {
                color: #d9d9d9;
            }

            body {
                background-color: #222;
            }

            .spinner {
                border-color: #555;
                border-top-color: #999;
            }

            .error {
                background-color: #3a1a1a;
                border-color: #5a2a2a;
                color: #faa;
            }
        }
    </style>
</head>
<body>
    <div class="main-wrapper">
        <div class="main-content">
            <div class="h2">Checking your browser before accessing the site...</div>

            <noscript>
                <div class="error active">Enable JavaScript and cookies to continue</div>
            </noscript>

            <div class="captcha-container">
                ` + providerWidget + `
            </div>

            <div class="loading" id="loading">
                <div class="spinner"></div>
                <div>Verifying...</div>
            </div>

            <div class="error" id="error">
                Verification failed. Please try again.
            </div>
        </div>
    </div>

    <div class="footer">
        Protected by Zoraxy
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
