package utils_test

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"imuslab.com/zoraxy/mod/utils"

	"github.com/stretchr/testify/assert"
)

// Test SendTextResponse
func TestSendTextResponse(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{"simple message", "Hello, World!", "Hello, World!"},
		{"empty message", "", ""},
		{"multiline message", "Line1\nLine2\nLine3", "Line1\nLine2\nLine3"},
		{"special characters", "!@#$%^&*()", "!@#$%^&*()"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			utils.SendTextResponse(w, tt.message)

			assert.Equal(t, tt.expected, w.Body.String())
			assert.Equal(t, 200, w.Code)
		})
	}
}

// Test SendJSONResponse
func TestSendJSONResponse(t *testing.T) {
	tests := []struct {
		name        string
		jsonString  string
		expected    string
		contentType string
	}{
		{"valid json object", `{"key":"value"}`, `{"key":"value"}`, "application/json"},
		{"valid json array", `["item1","item2"]`, `["item1","item2"]`, "application/json"},
		{"empty json", `{}`, `{}`, "application/json"},
		{"json with quotes", `{"name":"John \"Doe\""}`, `{"name":"John \"Doe\""}`, "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			utils.SendJSONResponse(w, tt.jsonString)

			assert.Equal(t, tt.expected, w.Body.String())
			assert.Equal(t, tt.contentType, w.Header().Get("Content-Type"))
			assert.Equal(t, 200, w.Code)
		})
	}
}

// Test SendErrorResponse
func TestSendErrorResponse(t *testing.T) {
	tests := []struct {
		name        string
		errorMsg    string
		expected    string
		contentType string
	}{
		{"simple error", "Something went wrong", `{"error":"Something went wrong"}`, "application/json"},
		{"empty error", "", `{"error":""}`, "application/json"},
		{"error with special chars", "Error: 100%", `{"error":"Error: 100%"}`, "application/json"},
		{"error with space", "File not found", `{"error":"File not found"}`, "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			utils.SendErrorResponse(w, tt.errorMsg)

			assert.Equal(t, tt.expected, w.Body.String())
			assert.Equal(t, tt.contentType, w.Header().Get("Content-Type"))
			assert.Equal(t, 200, w.Code)
		})
	}
}

// Test SendOK
func TestSendOK(t *testing.T) {
	w := httptest.NewRecorder()
	utils.SendOK(w)

	assert.Equal(t, `"OK"`, w.Body.String())
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, 200, w.Code)
}

// Test GetPara
func TestGetPara(t *testing.T) {
	tests := []struct {
		name      string
		queryStr  string
		key       string
		expected  string
		expectErr bool
	}{
		{"valid parameter", "key=value", "key", "value", false},
		{"missing parameter", "", "key", "", true},
		{"empty value", "key=", "key", "", true},
		{"multiple values", "key=val1&key=val2", "key", "val1", false}, // Gets first value
		{"url encoded value", "key=hello%20world", "key", "hello world", false},
		{"url encoded special chars", "key=%21%40%23%24%25", "key", "!@#$%", false},
		{"numeric value", "id=12345", "id", "12345", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?"+tt.queryStr, nil)
			result, err := utils.GetPara(req, tt.key)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// Test GetBool
func TestGetBool(t *testing.T) {
	tests := []struct {
		name      string
		queryStr  string
		key       string
		expected  bool
		expectErr bool
	}{
		{"true value - 1", "enabled=1", "enabled", true, false},
		{"true value - true", "enabled=true", "enabled", true, false},
		{"true value - TRUE", "enabled=TRUE", "enabled", true, false},
		{"true value - on", "enabled=on", "enabled", true, false},
		{"true value - ON", "enabled=ON", "enabled", true, false},
		{"false value - 0", "enabled=0", "enabled", false, false},
		{"false value - false", "enabled=false", "enabled", false, false},
		{"false value - FALSE", "enabled=FALSE", "enabled", false, false},
		{"false value - off", "enabled=off", "enabled", false, false},
		{"false value - OFF", "enabled=OFF", "enabled", false, false},
		{"invalid boolean - yes", "enabled=yes", "enabled", false, true},
		{"invalid boolean - no", "enabled=no", "enabled", false, true},
		{"invalid boolean - 2", "enabled=2", "enabled", false, true},
		{"invalid boolean - random", "enabled=random", "enabled", false, true},
		{"missing parameter", "", "enabled", false, true},
		{"whitespace handling", "enabled=%20true%20", "enabled", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?"+tt.queryStr, nil)
			result, err := utils.GetBool(req, tt.key)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// Test PostPara
func TestPostPara(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		key         string
		expected    string
		expectErr   bool
	}{
		{"valid form parameter", "application/x-www-form-urlencoded", "key=value", "key", "value", false},
		{"missing parameter", "application/x-www-form-urlencoded", "", "key", "", true},
		{"empty value", "application/x-www-form-urlencoded", "key=", "key", "", true},
		{"url encoded value", "application/x-www-form-urlencoded", "key=hello%20world", "key", "hello world", false},
		{"multiple values", "application/x-www-form-urlencoded", "key=val1&key=val2", "key", "val1", false},
		{"special characters", "application/x-www-form-urlencoded", "key=!@#$%25", "key", "!@#$%", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			result, err := utils.PostPara(req, tt.key)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// Test PostDuration
func TestPostDuration(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		key         string
		expected    time.Duration
		expectErr   bool
	}{
		{"seconds", "application/x-www-form-urlencoded", "duration=5s", "duration", 5 * time.Second, false},
		{"minutes", "application/x-www-form-urlencoded", "duration=10m", "duration", 10 * time.Minute, false},
		{"hours", "application/x-www-form-urlencoded", "duration=2h", "duration", 2 * time.Hour, false},
		{"complex duration", "application/x-www-form-urlencoded", "duration=1h30m", "duration", 90 * time.Minute, false},
		{"milliseconds", "application/x-www-form-urlencoded", "duration=500ms", "duration", 500 * time.Millisecond, false},
		{"invalid duration", "application/x-www-form-urlencoded", "duration=invalid", "duration", 0, true},
		{"missing parameter", "application/x-www-form-urlencoded", "", "duration", 0, true},
		{"empty value", "application/x-www-form-urlencoded", "duration=", "duration", 0, true},
		{"numeric without unit", "application/x-www-form-urlencoded", "duration=5", "duration", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			result, err := utils.PostDuration(req, tt.key)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected, *result)
			}
		})
	}
}

// Test PostBool
func TestPostBool(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		key         string
		expected    bool
		expectErr   bool
	}{
		{"true value - 1", "application/x-www-form-urlencoded", "enabled=1", "enabled", true, false},
		{"true value - true", "application/x-www-form-urlencoded", "enabled=true", "enabled", true, false},
		{"true value - on", "application/x-www-form-urlencoded", "enabled=on", "enabled", true, false},
		{"false value - 0", "application/x-www-form-urlencoded", "enabled=0", "enabled", false, false},
		{"false value - false", "application/x-www-form-urlencoded", "enabled=false", "enabled", false, false},
		{"false value - off", "application/x-www-form-urlencoded", "enabled=off", "enabled", false, false},
		{"invalid boolean", "application/x-www-form-urlencoded", "enabled=yes", "enabled", false, true},
		{"missing parameter", "application/x-www-form-urlencoded", "", "enabled", false, true},
		{"case insensitive TRUE", "application/x-www-form-urlencoded", "enabled=TRUE", "enabled", true, false},
		{"whitespace handling", "application/x-www-form-urlencoded", "enabled= false ", "enabled", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			result, err := utils.PostBool(req, tt.key)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// Test PostInt
func TestPostInt(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		key         string
		expected    int
		expectErr   bool
	}{
		{"positive integer", "application/x-www-form-urlencoded", "count=42", "count", 42, false},
		{"negative integer", "application/x-www-form-urlencoded", "count=-10", "count", -10, false},
		{"zero", "application/x-www-form-urlencoded", "count=0", "count", 0, false},
		{"large number", "application/x-www-form-urlencoded", "count=999999", "count", 999999, false},
		{"invalid integer", "application/x-www-form-urlencoded", "count=abc", "count", 0, true},
		{"float value", "application/x-www-form-urlencoded", "count=3.14", "count", 0, true},
		{"missing parameter", "application/x-www-form-urlencoded", "", "count", 0, true},
		{"whitespace", "application/x-www-form-urlencoded", "count= 100 ", "count", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			result, err := utils.PostInt(req, tt.key)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// Test FileExists
func TestFileExists(t *testing.T) {
	// Use known paths
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"existing file", "/etc/hosts", true},  // Standard Linux file
		{"non-existing file", "/tmp/this_file_definitely_does_not_exist_12345.txt", false},
		{"empty path", "", false},
		{"directory as file", "/tmp", true}, // FileExists returns true for directories too
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.FileExists(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test IsDir
func TestIsDir(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"existing directory", "/tmp", true},
		{"non-existing path", "/this/path/does/not/exist", false},
		{"file not directory", "/etc/hosts", false},
		{"empty path", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsDir(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test TimeToString
func TestTimeToString(t *testing.T) {
	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			"standard time",
			time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC),
			"2023-12-25 15:30:45",
		},
		{
			"midnight",
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			"2024-01-01 00:00:00",
		},
		{
			"end of day",
			time.Date(2024, 6, 15, 23, 59, 59, 0, time.UTC),
			"2024-06-15 23:59:59",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.TimeToString(tt.time)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test StringInArray
func TestStringInArray(t *testing.T) {
	tests := []struct {
		name     string
		arr      []string
		str      string
		expected bool
	}{
		{"string exists", []string{"apple", "banana", "cherry"}, "banana", true},
		{"string not exists", []string{"apple", "banana", "cherry"}, "orange", false},
		{"empty array", []string{}, "test", false},
		{"empty string in array", []string{"", "test"}, "", true},
		{"case sensitive - match", []string{"Test", "TEST", "test"}, "test", true},
		{"case sensitive - no match", []string{"Test", "TEST"}, "test", false},
		{"single element match", []string{"only"}, "only", true},
		{"single element no match", []string{"only"}, "other", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.StringInArray(tt.arr, tt.str)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test StringInArrayIgnoreCase
func TestStringInArrayIgnoreCase(t *testing.T) {
	tests := []struct {
		name     string
		arr      []string
		str      string
		expected bool
	}{
		{"exact match", []string{"apple", "banana", "cherry"}, "banana", true},
		{"case insensitive match - uppercase", []string{"apple", "banana", "cherry"}, "BANANA", true},
		{"case insensitive match - mixed", []string{"Apple", "Banana", "Cherry"}, "bAnAnA", true},
		{"no match", []string{"apple", "banana", "cherry"}, "orange", false},
		{"empty array", []string{}, "test", false},
		{"empty string", []string{"", "test"}, "", true},
		{"all uppercase array", []string{"TEST", "DEMO", "SAMPLE"}, "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.StringInArrayIgnoreCase(tt.arr, tt.str)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test ValidateListeningAddress
func TestValidateListeningAddress(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		expected bool
	}{
		// Valid addresses
		{"port only with colon", ":8080", true},
		{"localhost with port", "127.0.0.1:8080", true},
		{"IPv4 with port", "192.168.1.1:3000", true},
		{"IPv6 with port", "[::1]:8080", true},
		{"IPv6 full with port", "[2001:db8::1]:8080", true},
		{"empty host with port", ":80", true},
		{"zero IP with port", "0.0.0.0:8080", true},

		// Invalid addresses
		{"port only without colon", "8080", false},
		{"no port", "192.168.1.1", false},
		{"invalid IP", "999.999.999.999:8080", false},
		{"hostname instead of IP", "localhost:8080", false},
		{"empty string", "", false},
		{"just colon", ":", true}, // Edge case: just colon with empty port
		{"non-numeric port", "192.168.1.1:abc", false},
		{"IPv4 without port", "192.168.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.ValidateListeningAddress(tt.address)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test edge cases for GET parameter extraction with multiple params
func TestGetParaMultipleParameters(t *testing.T) {
	req := httptest.NewRequest("GET", "/?name=John&age=30&city=NYC", nil)

	name, err := utils.GetPara(req, "name")
	assert.NoError(t, err)
	assert.Equal(t, "John", name)

	age, err := utils.GetPara(req, "age")
	assert.NoError(t, err)
	assert.Equal(t, "30", age)

	city, err := utils.GetPara(req, "city")
	assert.NoError(t, err)
	assert.Equal(t, "NYC", city)

	_, err = utils.GetPara(req, "missing")
	assert.Error(t, err)
}

// Test URL encoded special characters in parameters
func TestGetParaURLEncoding(t *testing.T) {
	tests := []struct {
		name     string
		encoded  string
		expected string
	}{
		{"spaces", "key=hello%20world", "hello world"},
		{"plus sign", "key=1%2B1", "1+1"},
		{"ampersand", "key=foo%26bar", "foo&bar"},
		{"equals sign", "key=a%3Db", "a=b"},
		{"unicode", "key=%E2%9C%93", "✓"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?"+tt.encoded, nil)
			result, err := utils.GetPara(req, "key")
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
