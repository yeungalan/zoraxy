package utils_test

import (
	"os"
	"path/filepath"
	"testing"

	"imuslab.com/zoraxy/mod/utils"

	"github.com/stretchr/testify/assert"
)

func TestSizeStringToBytes(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"1024", 1024, false},
		{"1k", 1024, false},
		{"1K", 1024, false},
		{"2kb", 2 * 1024, false},
		{"1M", 1024 * 1024, false},
		{"3mb", 3 * 1024 * 1024, false},
		{"1g", 1024 * 1024 * 1024, false},
		{"2gb", 2 * 1024 * 1024 * 1024, false},
		{"", 0, false},
		{"  5mb  ", 5 * 1024 * 1024, false},
		{"invalid", 0, true},
		{"1tb", 1099511627776, false},
		{"1.5mb", int64(1.5 * 1024 * 1024), false},
		{"1pb", 1125899906842624, false},
		{"2.5kb", int64(2.5 * 1024), false},
		{"0", 0, false},
		{"0kb", 0, false},
		{"1t", 1099511627776, false},
		{"1p", 1125899906842624, false},
	}

	for _, tt := range tests {
		got, err := utils.SizeStringToBytes(tt.input)
		if tt.hasError {
			assert.Error(t, err, "input: %s", tt.input)
		} else {
			assert.NoError(t, err, "input: %s", tt.input)
			assert.Equal(t, tt.expected, got, "input: %s", tt.input)
		}
	}
}

func TestStringToInt64(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  int64
		expectErr bool
	}{
		{"positive number", "12345", 12345, false},
		{"negative number", "-12345", -12345, false},
		{"zero", "0", 0, false},
		{"large number", "9223372036854775807", 9223372036854775807, false}, // Max int64
		{"invalid string", "abc", -1, true},
		{"float value", "3.14", -1, true},
		{"empty string", "", -1, true},
		{"mixed alphanumeric", "123abc", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := utils.StringToInt64(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestInt64ToString(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{"positive number", 12345, "12345"},
		{"negative number", -12345, "-12345"},
		{"zero", 0, "0"},
		{"large number", 9223372036854775807, "9223372036854775807"},
		{"small negative", -1, "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.Int64ToString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBytesToHumanReadable(t *testing.T) {
	tests := []struct {
		name     string
		input    int64
		expected string
	}{
		{"bytes", 512, "512 Bytes"},
		{"exactly 1 KB", 1024, "1.00 KB"},
		{"kilobytes", 2048, "2.00 KB"},
		{"exactly 1 MB", 1024 * 1024, "1.00 MB"},
		{"megabytes", 5 * 1024 * 1024, "5.00 MB"},
		{"exactly 1 GB", 1024 * 1024 * 1024, "1.00 GB"},
		{"gigabytes", 3 * 1024 * 1024 * 1024, "3.00 GB"},
		{"terabytes", 2 * 1024 * 1024 * 1024 * 1024, "2.00 TB"},
		{"zero bytes", 0, "0 Bytes"},
		{"fractional KB", 1536, "1.50 KB"},
		{"fractional MB", int64(2.5 * 1024 * 1024), "2.50 MB"},
		{"large TB", 10 * 1024 * 1024 * 1024 * 1024, "10.00 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.BytesToHumanReadable(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplaceSpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple filename", "file.txt", "file_txt"},
		{"hash symbol", "file#1.txt", "file%pound%1_txt"},
		{"ampersand", "file&data.txt", "file%amp%data_txt"},
		{"curly braces", "file{test}.txt", "file%left_cur%test%right_cur%_txt"},
		{"backslash", "path\\file.txt", "path%backslash%file_txt"},
		{"angle brackets", "file<1>.txt", "file%left_ang%1%right_ang%_txt"},
		{"asterisk", "file*.txt", "file%aster%_txt"},
		{"question mark", "file?.txt", "file%quest%_txt"},
		{"space", "my file.txt", "my%space%file_txt"},
		{"dollar sign", "file$100.txt", "file%dollar%100_txt"},
		{"exclamation", "file!.txt", "file%exclan%_txt"},
		{"quotes", `file"test"'s.txt`, "file%dou_q%test%dou_q%%sin_q%s_txt"},
		{"colon", "file:data.txt", "file%colon%data_txt"},
		{"at symbol", "file@test.txt", "file%at%test_txt"},
		{"plus", "file+1.txt", "file%plus%1_txt"},
		{"backtick", "file`cmd`.txt", "file%backtick%cmd%backtick%_txt"},
		{"pipe", "file|data.txt", "file%pipe%data_txt"},
		{"equals", "file=1.txt", "file%equal%1_txt"},
		{"dot replacement", "file.name.txt", "file_name_txt"},
		{"slash replacement", "path/to/file.txt", "path-to-file_txt"},
		{"multiple special chars", "test#file & data.txt", "test%pound%file%space%%amp%%space%data_txt"},
		{"all special chars", `#&{}\<>*? $!'":@+` + "`|=./", "%pound%%amp%%left_cur%%right_cur%%backslash%%left_ang%%right_ang%%aster%%quest%%space%%dollar%%exclan%%sin_q%%dou_q%%colon%%at%%plus%%backtick%%pipe%%equal%_-"},
		{"no special chars", "filename", "filename"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.ReplaceSpecialCharacters(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestZipFiles(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "ziptest")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	zipPath := filepath.Join(tmpDir, "test.zip")

	err = os.WriteFile(file1, []byte("Content of file 1"), 0644)
	assert.NoError(t, err)

	err = os.WriteFile(file2, []byte("Content of file 2"), 0644)
	assert.NoError(t, err)

	// Test successful zip creation
	t.Run("successful zip creation", func(t *testing.T) {
		err := utils.ZipFiles(zipPath, file1, file2)
		assert.NoError(t, err)
		assert.True(t, utils.FileExists(zipPath))

		// Verify zip file is not empty
		info, err := os.Stat(zipPath)
		assert.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0))
	})

	// Test with non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		nonExistentZip := filepath.Join(tmpDir, "fail.zip")
		err := utils.ZipFiles(nonExistentZip, filepath.Join(tmpDir, "nonexistent.txt"))
		assert.Error(t, err)
	})

	// Test with single file
	t.Run("single file zip", func(t *testing.T) {
		singleZip := filepath.Join(tmpDir, "single.zip")
		err := utils.ZipFiles(singleZip, file1)
		assert.NoError(t, err)
		assert.True(t, utils.FileExists(singleZip))
	})

	// Test with empty file list
	t.Run("empty file list", func(t *testing.T) {
		emptyZip := filepath.Join(tmpDir, "empty.zip")
		err := utils.ZipFiles(emptyZip)
		assert.NoError(t, err)
		assert.True(t, utils.FileExists(emptyZip))
	})
}

func TestSizeStringToBytesEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  int64
		expectErr bool
	}{
		{"just unit", "kb", 0, true},       // No number
		{"negative size", "-5mb", int64(-5 * 1024 * 1024), false},
		{"very small decimal", "0.001mb", 1048, false},
		{"leading zeros", "00005kb", 5 * 1024, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := utils.SizeStringToBytes(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
