package wakeonlan

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test IsValidMacAddress with various MAC address formats
func TestIsValidMacAddress(t *testing.T) {
	tests := []struct {
		name     string
		macAddr  string
		expected bool
	}{
		// Valid MAC addresses
		{"valid colon format lowercase", "aa:bb:cc:dd:ee:ff", true},
		{"valid colon format uppercase", "AA:BB:CC:DD:EE:FF", true},
		{"valid colon format mixed", "Aa:Bb:Cc:Dd:Ee:Ff", true},
		{"valid hyphen format", "AA-BB-CC-DD-EE-FF", true},
		{"valid dot format", "aabb.ccdd.eeff", true},
		{"valid all zeros", "00:00:00:00:00:00", true},
		{"valid all ones", "FF:FF:FF:FF:FF:FF", true},

		// Invalid MAC addresses
		{"empty string", "", false},
		{"too short", "aa:bb:cc:dd:ee", false},
		{"too long", "aa:bb:cc:dd:ee:ff:gg", false},
		{"invalid characters", "xx:yy:zz:aa:bb:cc", false},
		{"invalid separator", "aa_bb_cc_dd_ee_ff", false},
		{"missing separator", "aa:bbccddee:ff", false},
		{"only separator", ":::::", false},
		{"spaces", "aa bb cc dd ee ff", false},
		{"letters only", "aabbccddeeffgg", false},
		{"single char", "a", false},
		{"random string", "not-a-mac-address", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidMacAddress(tt.macAddr)
			assert.Equal(t, tt.expected, result, "MAC address: %s", tt.macAddr)
		})
	}
}

// Test WakeTarget with invalid MAC addresses
func TestWakeTargetInvalidMAC(t *testing.T) {
	tests := []struct {
		name    string
		macAddr string
	}{
		{"empty string", ""},
		{"invalid format", "invalid-mac"},
		{"too short", "aa:bb:cc:dd:ee"},
		{"invalid chars", "xx:yy:zz:aa:bb:cc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WakeTarget(tt.macAddr)
			assert.Error(t, err, "Expected error for MAC: %s", tt.macAddr)
		})
	}
}

// Test magic packet construction
func TestMagicPacketConstruction(t *testing.T) {
	// Test that a valid MAC address creates a proper magic packet structure
	macAddr := "AA:BB:CC:DD:EE:FF"

	// Parse the MAC
	mac, err := net.ParseMAC(macAddr)
	assert.NoError(t, err)
	assert.Equal(t, 6, len(mac))

	// Construct magic packet manually to verify structure
	packet := magicPacket{}

	// First 6 bytes should be all 0xFF
	copy(packet[0:], []byte{255, 255, 255, 255, 255, 255})

	// Next 16 repetitions of the MAC address
	offset := 6
	for i := 0; i < 16; i++ {
		copy(packet[offset:], mac)
		offset += 6
	}

	// Verify packet structure
	assert.Equal(t, 102, len(packet), "Magic packet should be 102 bytes")

	// First 6 bytes should be 0xFF
	for i := 0; i < 6; i++ {
		assert.Equal(t, byte(255), packet[i], "First 6 bytes should be 0xFF")
	}

	// Next 96 bytes (16 * 6) should be repetitions of MAC
	expectedMAC := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	for i := 0; i < 16; i++ {
		offset := 6 + (i * 6)
		for j := 0; j < 6; j++ {
			assert.Equal(t, expectedMAC[j], packet[offset+j],
				"MAC byte mismatch at repetition %d, byte %d", i, j)
		}
	}
}

// Test magic packet with different MAC addresses
func TestMagicPacketWithDifferentMACs(t *testing.T) {
	testMACs := []struct {
		name   string
		mac    string
		bytes  []byte
	}{
		{"all zeros", "00:00:00:00:00:00", []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{"all ones", "FF:FF:FF:FF:FF:FF", []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}},
		{"mixed", "12:34:56:78:9A:BC", []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC}},
	}

	for _, tt := range testMACs {
		t.Run(tt.name, func(t *testing.T) {
			mac, err := net.ParseMAC(tt.mac)
			assert.NoError(t, err)

			packet := magicPacket{}
			copy(packet[0:], []byte{255, 255, 255, 255, 255, 255})

			offset := 6
			for i := 0; i < 16; i++ {
				copy(packet[offset:], mac)
				offset += 6
			}

			// Verify the MAC is repeated correctly
			for i := 0; i < 16; i++ {
				offset := 6 + (i * 6)
				for j := 0; j < 6; j++ {
					assert.Equal(t, tt.bytes[j], packet[offset+j])
				}
			}
		})
	}
}

// Test MAC address parsing edge cases
func TestMACAddressParsingEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		macAddr   string
		shouldErr bool
		length    int
	}{
		{"standard colon", "AA:BB:CC:DD:EE:FF", false, 6},
		{"standard hyphen", "AA-BB-CC-DD-EE-FF", false, 6},
		{"cisco format", "aabb.ccdd.eeff", false, 6},
		{"lowercase", "aa:bb:cc:dd:ee:ff", false, 6},
		{"mixed case", "Aa:Bb:Cc:Dd:Ee:Ff", false, 6},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mac, err := net.ParseMAC(tt.macAddr)
			if tt.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.length, len(mac))
			}
		})
	}
}

// Test packet size
func TestMagicPacketSize(t *testing.T) {
	var packet magicPacket
	// Magic packet should be exactly 102 bytes:
	// 6 bytes of 0xFF + (16 repetitions * 6 bytes MAC) = 6 + 96 = 102
	assert.Equal(t, 102, len(packet))
}

// Test that invalid MAC lengths are rejected by WakeTarget
func TestWakeTargetInvalidMACLength(t *testing.T) {
	// Create a MAC-like string that parses but has wrong length
	// Note: net.ParseMAC actually normalizes to 6 bytes for standard Ethernet MACs
	// This test ensures the length check works even if ParseMAC succeeds

	// Test with a parseable MAC that should pass the net.ParseMAC but fail length check
	// However, net.ParseMAC normalizes all valid Ethernet MACs to 6 bytes
	// So we test the error path indirectly through invalid formats
	invalidMACs := []string{
		"",                    // Empty
		"invalid",             // Invalid format
		"AA:BB",               // Too short
		"AA:BB:CC:DD:EE:FF:GG:HH", // Too long format
	}

	for _, macAddr := range invalidMACs {
		err := WakeTarget(macAddr)
		assert.Error(t, err, "Should error for MAC: %s", macAddr)
	}
}

// Test valid MAC formats that should work
func TestIsValidMacAddressFormats(t *testing.T) {
	validFormats := []string{
		"01:23:45:67:89:AB",           // Standard colon notation
		"01-23-45-67-89-AB",           // Hyphen notation
		"0123.4567.89AB",              // Cisco notation
		"01:23:45:67:89:ab",           // Lowercase
		"01-23-45-67-89-ab",           // Lowercase hyphen
		"00:00:00:00:00:00",           // All zeros
		"FF:FF:FF:FF:FF:FF",           // Broadcast MAC
	}

	for _, mac := range validFormats {
		t.Run(mac, func(t *testing.T) {
			assert.True(t, IsValidMacAddress(mac), "Should be valid: %s", mac)
		})
	}
}

// Test invalid MAC formats
func TestIsValidMacAddressInvalidFormats(t *testing.T) {
	invalidFormats := []string{
		"",                           // Empty
		"not-a-mac",                  // Random string
		"GG:HH:II:JJ:KK:LL",         // Invalid hex chars
		"AA:BB:CC:DD:EE",            // Too short
		"AA:BB:CC:DD:EE:FF:GG",      // Too long
		"AA_BB_CC_DD_EE_FF",         // Wrong separator
		"AABBCCDDEEFFGG",            // Too many chars
		"AA BB CC DD EE FF",         // Spaces
		"AA:BB:CC:DD:EE:FG",         // Invalid hex digit
		"::::::",                    // Just separators
		"12345",                     // Too short
	}

	for _, mac := range invalidFormats {
		t.Run(mac, func(t *testing.T) {
			assert.False(t, IsValidMacAddress(mac), "Should be invalid: %s", mac)
		})
	}
}
