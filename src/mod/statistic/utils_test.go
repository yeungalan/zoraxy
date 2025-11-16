package statistic_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"imuslab.com/zoraxy/mod/statistic"
)

func TestIsBeforeToday(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Yesterday",
			input:    time.Now().AddDate(0, 0, -1).Format("2006_01_02"),
			expected: true,
		},
		{
			name:     "Today",
			input:    time.Now().Format("2006_01_02"),
			expected: true,
		},
		{
			name:     "Tomorrow",
			input:    time.Now().AddDate(0, 0, 1).Format("2006_01_02"),
			expected: false,
		},
		{
			name:     "Last Week",
			input:    time.Now().AddDate(0, 0, -7).Format("2006_01_02"),
			expected: true,
		},
		{
			name:     "Invalid Date Format",
			input:    "invalid-date",
			expected: false,
		},
		{
			name:     "Empty String",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := statistic.IsBeforeToday(tt.input)
			assert.Equal(t, tt.expected, result, "Test case: %s", tt.name)
		})
	}
}

func TestIsValidIPAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Valid IPv4",
			input:    "192.168.1.1",
			expected: true,
		},
		{
			name:     "Valid IPv4 Loopback",
			input:    "127.0.0.1",
			expected: true,
		},
		{
			name:     "Valid IPv4 Public",
			input:    "8.8.8.8",
			expected: true,
		},
		{
			name:     "Valid IPv6",
			input:    "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			expected: true,
		},
		{
			name:     "Valid IPv6 Short",
			input:    "::1",
			expected: true,
		},
		{
			name:     "Valid IPv6 Compressed",
			input:    "2001:db8::1",
			expected: true,
		},
		{
			name:     "Invalid IP - Letters",
			input:    "abc.def.ghi.jkl",
			expected: false,
		},
		{
			name:     "Invalid IP - Out of Range",
			input:    "256.256.256.256",
			expected: false,
		},
		{
			name:     "Invalid IP - Empty",
			input:    "",
			expected: false,
		},
		{
			name:     "Invalid IP - Hostname",
			input:    "example.com",
			expected: false,
		},
		{
			name:     "Invalid IP - Incomplete",
			input:    "192.168.1",
			expected: false,
		},
		{
			name:     "Invalid IP - Special Characters",
			input:    "192.168.1.1/24",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := statistic.IsValidIPAddress(tt.input)
			assert.Equal(t, tt.expected, result, "Test case: %s", tt.name)
		})
	}
}
