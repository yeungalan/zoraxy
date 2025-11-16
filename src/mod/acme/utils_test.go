package acme

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to generate a test certificate
func generateTestCertificate(domains []string, issuerOrg string, notBefore, notAfter time.Time) ([]byte, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: domains[0],
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              domains,
		Issuer: pkix.Name{
			Organization: []string{issuerOrg},
		},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	return certPEM, nil
}

func TestExtractIssuerName(t *testing.T) {
	tests := []struct {
		name        string
		issuerOrg   string
		expectError bool
		expected    string
	}{
		{
			name:        "Let's Encrypt certificate",
			issuerOrg:   "Let's Encrypt",
			expectError: false,
			expected:    "Let's Encrypt",
		},
		{
			name:        "Buypass certificate",
			issuerOrg:   "Buypass",
			expectError: false,
			expected:    "Buypass",
		},
		{
			name:        "ZeroSSL certificate",
			issuerOrg:   "ZeroSSL",
			expectError: false,
			expected:    "ZeroSSL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			certPEM, err := generateTestCertificate(
				[]string{"example.com"},
				tt.issuerOrg,
				time.Now(),
				time.Now().Add(90*24*time.Hour),
			)
			require.NoError(t, err)

			issuer, err := ExtractIssuerName(certPEM)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, issuer)
			}
		})
	}
}

func TestExtractIssuerName_InvalidCertificate(t *testing.T) {
	tests := []struct {
		name    string
		certPEM []byte
	}{
		{
			name:    "Invalid PEM data",
			certPEM: []byte("invalid pem data"),
		},
		{
			name:    "Empty PEM",
			certPEM: []byte(""),
		},
		{
			name: "Non-certificate PEM",
			certPEM: []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0Z+
-----END RSA PRIVATE KEY-----`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExtractIssuerName(tt.certPEM)
			assert.Error(t, err)
		})
	}
}

func TestExtractIssuerName_NoOrganization(t *testing.T) {
	// Generate a certificate without organization
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "example.com",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(90 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"example.com"},
		Issuer: pkix.Name{
			// No organization
			CommonName: "Test CA",
		},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})

	_, err = ExtractIssuerName(certPEM)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "org section")
}

func TestExtractDomains(t *testing.T) {
	tests := []struct {
		name           string
		domains        []string
		expectedDomains []string
	}{
		{
			name:           "Single domain",
			domains:        []string{"example.com"},
			expectedDomains: []string{"example.com"},
		},
		{
			name:           "Multiple domains",
			domains:        []string{"example.com", "www.example.com", "api.example.com"},
			expectedDomains: []string{"example.com", "www.example.com", "api.example.com"},
		},
		{
			name:           "Wildcard domain",
			domains:        []string{"*.example.com", "example.com"},
			expectedDomains: []string{"*.example.com", "example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			certPEM, err := generateTestCertificate(
				tt.domains,
				"Let's Encrypt",
				time.Now(),
				time.Now().Add(90*24*time.Hour),
			)
			require.NoError(t, err)

			extractedDomains, err := ExtractDomains(certPEM)
			assert.NoError(t, err)
			assert.ElementsMatch(t, tt.expectedDomains, extractedDomains)
		})
	}
}

func TestExtractDomains_InvalidCertificate(t *testing.T) {
	tests := []struct {
		name    string
		certPEM []byte
	}{
		{
			name:    "Invalid PEM data",
			certPEM: []byte("invalid pem data"),
		},
		{
			name:    "Empty data",
			certPEM: []byte(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ExtractDomains(tt.certPEM)
			assert.Error(t, err)
		})
	}
}

func TestExtractIssuerNameFromPEM(t *testing.T) {
	// Create a temporary test certificate file
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "test.pem")

	certPEM, err := generateTestCertificate(
		[]string{"example.com"},
		"Let's Encrypt",
		time.Now(),
		time.Now().Add(90*24*time.Hour),
	)
	require.NoError(t, err)

	err = os.WriteFile(certPath, certPEM, 0644)
	require.NoError(t, err)

	issuer, err := ExtractIssuerNameFromPEM(certPath)
	assert.NoError(t, err)
	assert.Equal(t, "Let's Encrypt", issuer)
}

func TestExtractIssuerNameFromPEM_FileNotFound(t *testing.T) {
	_, err := ExtractIssuerNameFromPEM("/nonexistent/path/cert.pem")
	assert.Error(t, err)
}

func TestExtractDomainsFromPEM(t *testing.T) {
	// Create a temporary test certificate file
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "test.pem")

	expectedDomains := []string{"example.com", "www.example.com"}
	certPEM, err := generateTestCertificate(
		expectedDomains,
		"Let's Encrypt",
		time.Now(),
		time.Now().Add(90*24*time.Hour),
	)
	require.NoError(t, err)

	err = os.WriteFile(certPath, certPEM, 0644)
	require.NoError(t, err)

	domains, err := ExtractDomainsFromPEM(certPath)
	assert.NoError(t, err)
	assert.ElementsMatch(t, expectedDomains, domains)
}

func TestExtractDomainsFromPEM_FileNotFound(t *testing.T) {
	_, err := ExtractDomainsFromPEM("/nonexistent/path/cert.pem")
	assert.Error(t, err)
}

func TestCertIsExpired(t *testing.T) {
	tests := []struct {
		name       string
		notBefore  time.Time
		notAfter   time.Time
		shouldExpire bool
	}{
		{
			name:       "Valid certificate",
			notBefore:  time.Now().Add(-30 * 24 * time.Hour),
			notAfter:   time.Now().Add(60 * 24 * time.Hour),
			shouldExpire: false,
		},
		{
			name:       "Expired certificate",
			notBefore:  time.Now().Add(-90 * 24 * time.Hour),
			notAfter:   time.Now().Add(-1 * time.Hour),
			shouldExpire: true,
		},
		{
			name:       "Just expired certificate",
			notBefore:  time.Now().Add(-90 * 24 * time.Hour),
			notAfter:   time.Now().Add(-1 * time.Second),
			shouldExpire: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			certPEM, err := generateTestCertificate(
				[]string{"example.com"},
				"Let's Encrypt",
				tt.notBefore,
				tt.notAfter,
			)
			require.NoError(t, err)

			isExpired := CertIsExpired(certPEM)
			assert.Equal(t, tt.shouldExpire, isExpired)
		})
	}
}

func TestCertIsExpired_InvalidCertificate(t *testing.T) {
	// Should return false for invalid certificates (can't determine expiry)
	isExpired := CertIsExpired([]byte("invalid certificate"))
	assert.False(t, isExpired)
}

func TestCertExpireSoon(t *testing.T) {
	tests := []struct {
		name          string
		notBefore     time.Time
		notAfter      time.Time
		numberOfDays  int
		shouldExpireSoon bool
	}{
		{
			name:          "Certificate expires in 10 days, checking 30 days",
			notBefore:     time.Now().Add(-80 * 24 * time.Hour),
			notAfter:      time.Now().Add(10 * 24 * time.Hour),
			numberOfDays:  30,
			shouldExpireSoon: true,
		},
		{
			name:          "Certificate expires in 60 days, checking 30 days",
			notBefore:     time.Now().Add(-30 * 24 * time.Hour),
			notAfter:      time.Now().Add(60 * 24 * time.Hour),
			numberOfDays:  30,
			shouldExpireSoon: false,
		},
		{
			name:          "Certificate expires in exactly 30 days, checking 30 days",
			notBefore:     time.Now().Add(-60 * 24 * time.Hour),
			notAfter:      time.Now().Add(30 * 24 * time.Hour),
			numberOfDays:  30,
			shouldExpireSoon: true,
		},
		{
			name:          "Already expired certificate",
			notBefore:     time.Now().Add(-90 * 24 * time.Hour),
			notAfter:      time.Now().Add(-1 * time.Hour),
			numberOfDays:  30,
			shouldExpireSoon: true,
		},
		{
			name:          "Certificate expires in 5 days, checking 7 days",
			notBefore:     time.Now().Add(-85 * 24 * time.Hour),
			notAfter:      time.Now().Add(5 * 24 * time.Hour),
			numberOfDays:  7,
			shouldExpireSoon: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			certPEM, err := generateTestCertificate(
				[]string{"example.com"},
				"Let's Encrypt",
				tt.notBefore,
				tt.notAfter,
			)
			require.NoError(t, err)

			expiresSoon := CertExpireSoon(certPEM, tt.numberOfDays)
			assert.Equal(t, tt.shouldExpireSoon, expiresSoon, "Expected expiresSoon to be %v", tt.shouldExpireSoon)
		})
	}
}

func TestCertExpireSoon_InvalidCertificate(t *testing.T) {
	// Should return false for invalid certificates (can't determine expiry)
	expiresSoon := CertExpireSoon([]byte("invalid certificate"), 30)
	assert.False(t, expiresSoon)
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		str      string
		expected bool
	}{
		{
			name:     "String exists in slice",
			slice:    []string{"apple", "banana", "cherry"},
			str:      "banana",
			expected: true,
		},
		{
			name:     "String does not exist in slice",
			slice:    []string{"apple", "banana", "cherry"},
			str:      "orange",
			expected: false,
		},
		{
			name:     "Empty slice",
			slice:    []string{},
			str:      "apple",
			expected: false,
		},
		{
			name:     "Empty string in slice",
			slice:    []string{"apple", "", "cherry"},
			str:      "",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.str)
			assert.Equal(t, tt.expected, result)
		})
	}
}
