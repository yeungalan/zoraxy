package captcha

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

/*
	verifier.go

	This file handles CAPTCHA response verification
	for both Cloudflare Turnstile and Google reCAPTCHA.
*/

const (
	CloudflareTurnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	GoogleRecaptchaVerifyURL     = "https://www.google.com/recaptcha/api/siteverify"
)

// CloudflareResponse represents the response from Cloudflare Turnstile verification
type CloudflareResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts,omitempty"`
	Hostname    string   `json:"hostname,omitempty"`
	ErrorCodes  []string `json:"error-codes,omitempty"`
	Action      string   `json:"action,omitempty"`
	CData       string   `json:"cdata,omitempty"`
}

// GoogleRecaptchaResponse represents the response from Google reCAPTCHA verification
type GoogleRecaptchaResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts,omitempty"`
	Hostname    string   `json:"hostname,omitempty"`
	ErrorCodes  []string `json:"error-codes,omitempty"`
	Score       float64  `json:"score,omitempty"`       // reCAPTCHA v3 only
	Action      string   `json:"action,omitempty"`      // reCAPTCHA v3 only
}

// VerifyCloudflare verifies a Cloudflare Turnstile response
func VerifyCloudflare(secretKey, response, remoteIP string) (bool, error) {
	if secretKey == "" || response == "" {
		return false, errors.New("secret key and response are required")
	}

	// Prepare form data
	data := url.Values{}
	data.Set("secret", secretKey)
	data.Set("response", response)
	if remoteIP != "" {
		data.Set("remoteip", remoteIP)
	}

	// Make request to Cloudflare
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm(CloudflareTurnstileVerifyURL, data)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	// Parse response
	var cfResp CloudflareResponse
	if err := json.Unmarshal(body, &cfResp); err != nil {
		return false, err
	}

	if !cfResp.Success {
		return false, errors.New("cloudflare verification failed: " + strings.Join(cfResp.ErrorCodes, ", "))
	}

	return true, nil
}

// VerifyGoogle verifies a Google reCAPTCHA response
func VerifyGoogle(secretKey, response, remoteIP string) (bool, error) {
	if secretKey == "" || response == "" {
		return false, errors.New("secret key and response are required")
	}

	// Prepare form data
	data := url.Values{}
	data.Set("secret", secretKey)
	data.Set("response", response)
	if remoteIP != "" {
		data.Set("remoteip", remoteIP)
	}

	// Make request to Google
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm(GoogleRecaptchaVerifyURL, data)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	// Parse response
	var gResp GoogleRecaptchaResponse
	if err := json.Unmarshal(body, &gResp); err != nil {
		return false, err
	}

	if !gResp.Success {
		return false, errors.New("google recaptcha verification failed: " + strings.Join(gResp.ErrorCodes, ", "))
	}

	// For reCAPTCHA v3, you might want to check the score
	// A score of 0.5 is generally considered a good threshold
	// For now, we'll accept any successful response
	// Users can adjust this based on their security requirements

	return true, nil
}
