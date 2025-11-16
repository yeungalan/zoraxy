package utils_test

import (
	"net/http/httptest"
	"testing"

	"imuslab.com/zoraxy/mod/utils"

	"github.com/stretchr/testify/assert"
)

func TestSendHTMLResponse(t *testing.T) {
	tests := []struct {
		name        string
		htmlContent string
		expected    string
		contentType string
	}{
		{"simple HTML", "<html><body>Hello</body></html>", "<html><body>Hello</body></html>", "text/html"},
		{"empty HTML", "", "", "text/html"},
		{"HTML with special chars", "<div>&copy; 2024</div>", "<div>&copy; 2024</div>", "text/html"},
		{"complex HTML", `<!DOCTYPE html><html><head><title>Test</title></head><body><p>Test paragraph</p></body></html>`, `<!DOCTYPE html><html><head><title>Test</title></head><body><p>Test paragraph</p></body></html>`, "text/html"},
		{"HTML with inline CSS", `<style>body{color:red;}</style><p>Styled</p>`, `<style>body{color:red;}</style><p>Styled</p>`, "text/html"},
		{"HTML with script", `<script>alert('test');</script>`, `<script>alert('test');</script>`, "text/html"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			utils.SendHTMLResponse(w, tt.htmlContent)

			assert.Equal(t, tt.expected, w.Body.String())
			assert.Equal(t, tt.contentType, w.Header().Get("Content-Type"))
			assert.Equal(t, 200, w.Code)
		})
	}
}
