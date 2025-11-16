package pathrule

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"imuslab.com/zoraxy/mod/utils"
)

// Test NewPathRuleHandler
func TestNewPathRuleHandler(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathrule_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	options := &Options{
		Enabled:      true,
		ConfigFolder: tmpDir,
	}

	handler := NewPathRuleHandler(options)
	assert.NotNil(t, handler)
	assert.Equal(t, options, handler.Options)
	assert.Equal(t, 0, len(handler.BlockingPaths))
	assert.True(t, utils.FileExists(tmpDir))
}

// Test NewPathRuleHandler creates folder if not exists
func TestNewPathRuleHandlerCreatesFolder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathrule_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	newFolder := filepath.Join(tmpDir, "config")
	options := &Options{
		Enabled:      true,
		ConfigFolder: newFolder,
	}

	handler := NewPathRuleHandler(options)
	assert.NotNil(t, handler)
	assert.True(t, utils.FileExists(newFolder))
}

// Test ListBlockingPath
func TestListBlockingPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathrule_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	handler := NewPathRuleHandler(&Options{
		Enabled:      true,
		ConfigFolder: tmpDir,
	})

	// Empty list
	paths := handler.ListBlockingPath()
	assert.Equal(t, 0, len(paths))

	// Add some paths
	blocker1 := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/test",
		Enabled:      true,
	}
	blocker2 := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/admin",
		Enabled:      true,
	}

	handler.BlockingPaths = []*BlockingPath{blocker1, blocker2}
	paths = handler.ListBlockingPath()
	assert.Equal(t, 2, len(paths))
}

// Test GetPathBlockerFromMatchingPath
func TestGetPathBlockerFromMatchingPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathrule_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	handler := NewPathRuleHandler(&Options{
		Enabled:      true,
		ConfigFolder: tmpDir,
	})

	blocker1 := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/test",
		Enabled:      true,
	}
	blocker2 := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/admin/",
		Enabled:      true,
	}

	handler.BlockingPaths = []*BlockingPath{blocker1, blocker2}

	// Exact match
	result := handler.GetPathBlockerFromMatchingPath("/test")
	assert.NotNil(t, result)
	assert.Equal(t, "/test", result.MatchingPath)

	// Match with trailing slash normalized
	result = handler.GetPathBlockerFromMatchingPath("/test/")
	assert.NotNil(t, result)
	assert.Equal(t, "/test", result.MatchingPath)

	// Match admin path
	result = handler.GetPathBlockerFromMatchingPath("/admin")
	assert.NotNil(t, result)
	assert.Equal(t, "/admin/", result.MatchingPath)

	// Non-existent path
	result = handler.GetPathBlockerFromMatchingPath("/nonexistent")
	assert.Nil(t, result)
}

// Test GetPathBlockerFromUUID
func TestGetPathBlockerFromUUID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathrule_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	handler := NewPathRuleHandler(&Options{
		Enabled:      true,
		ConfigFolder: tmpDir,
	})

	uuid1 := uuid.New().String()
	uuid2 := uuid.New().String()

	blocker1 := &BlockingPath{
		UUID:         uuid1,
		MatchingPath: "/test",
		Enabled:      true,
	}
	blocker2 := &BlockingPath{
		UUID:         uuid2,
		MatchingPath: "/admin",
		Enabled:      true,
	}

	handler.BlockingPaths = []*BlockingPath{blocker1, blocker2}

	// Find by UUID
	result := handler.GetPathBlockerFromUUID(uuid1)
	assert.NotNil(t, result)
	assert.Equal(t, "/test", result.MatchingPath)

	result = handler.GetPathBlockerFromUUID(uuid2)
	assert.NotNil(t, result)
	assert.Equal(t, "/admin", result.MatchingPath)

	// Non-existent UUID
	result = handler.GetPathBlockerFromUUID(uuid.New().String())
	assert.Nil(t, result)
}

// Test AddBlockingPath
func TestAddBlockingPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathrule_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	handler := NewPathRuleHandler(&Options{
		Enabled:      true,
		ConfigFolder: tmpDir,
	})

	blocker := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/test",
		ExactMatch:   true,
		StatusCode:   404,
		Enabled:      true,
	}

	// Add blocker
	err = handler.AddBlockingPath(blocker)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(handler.BlockingPaths))

	// Verify file was created
	configFile := filepath.Join(tmpDir, blocker.UUID)
	assert.True(t, utils.FileExists(configFile))

	// Try to add duplicate (same path)
	blocker2 := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/test",
		ExactMatch:   false,
		StatusCode:   403,
		Enabled:      true,
	}

	err = handler.AddBlockingPath(blocker2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	assert.Equal(t, 1, len(handler.BlockingPaths))
}

// Test AddBlockingPath with trailing slash normalization
func TestAddBlockingPathTrailingSlash(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathrule_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	handler := NewPathRuleHandler(&Options{
		Enabled:      true,
		ConfigFolder: tmpDir,
	})

	blocker1 := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/test/",
		Enabled:      true,
	}

	err = handler.AddBlockingPath(blocker1)
	assert.NoError(t, err)

	// Try to add with same path but different trailing slash
	blocker2 := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/test",
		Enabled:      true,
	}

	err = handler.AddBlockingPath(blocker2)
	assert.Error(t, err)
}

// Test RemoveBlockingPathByUUID
func TestRemoveBlockingPathByUUID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathrule_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	handler := NewPathRuleHandler(&Options{
		Enabled:      true,
		ConfigFolder: tmpDir,
	})

	uuid1 := uuid.New().String()
	uuid2 := uuid.New().String()

	blocker1 := &BlockingPath{
		UUID:         uuid1,
		MatchingPath: "/test",
		Enabled:      true,
	}
	blocker2 := &BlockingPath{
		UUID:         uuid2,
		MatchingPath: "/admin",
		Enabled:      true,
	}

	handler.AddBlockingPath(blocker1)
	handler.AddBlockingPath(blocker2)
	assert.Equal(t, 2, len(handler.BlockingPaths))

	// Remove first blocker
	err = handler.RemoveBlockingPathByUUID(uuid1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(handler.BlockingPaths))
	assert.Equal(t, uuid2, handler.BlockingPaths[0].UUID)

	// Verify file was removed
	configFile := filepath.Join(tmpDir, uuid1)
	assert.False(t, utils.FileExists(configFile))

	// Try to remove non-existent UUID
	err = handler.RemoveBlockingPathByUUID(uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not exists")
}

// Test GetMatchingBlockers with exact match
func TestGetMatchingBlockersExactMatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathrule_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	handler := NewPathRuleHandler(&Options{
		Enabled:      true,
		ConfigFolder: tmpDir,
	})

	blocker1 := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/test",
		ExactMatch:   true,
		Enabled:      true,
	}

	handler.BlockingPaths = []*BlockingPath{blocker1}

	// Exact match
	matchers, longest := handler.GetMatchingBlockers("/test")
	assert.Equal(t, 1, len(matchers))
	assert.Equal(t, blocker1, longest)

	// With trailing slash
	matchers, longest = handler.GetMatchingBlockers("/test/")
	assert.Equal(t, 1, len(matchers))
	assert.Equal(t, blocker1, longest)

	// No match for prefix when exact match is required
	matchers, longest = handler.GetMatchingBlockers("/test/subpath")
	assert.Equal(t, 0, len(matchers))
	assert.Nil(t, longest)
}

// Test GetMatchingBlockers with prefix match
func TestGetMatchingBlockersPrefixMatch(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathrule_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	handler := NewPathRuleHandler(&Options{
		Enabled:      true,
		ConfigFolder: tmpDir,
	})

	blocker1 := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/admin",
		ExactMatch:   false,
		Enabled:      true,
	}

	handler.BlockingPaths = []*BlockingPath{blocker1}

	// Exact match
	matchers, longest := handler.GetMatchingBlockers("/admin")
	assert.Equal(t, 1, len(matchers))
	assert.Equal(t, blocker1, longest)

	// Prefix match
	matchers, longest = handler.GetMatchingBlockers("/admin/users")
	assert.Equal(t, 1, len(matchers))
	assert.Equal(t, blocker1, longest)

	matchers, longest = handler.GetMatchingBlockers("/admin/users/edit")
	assert.Equal(t, 1, len(matchers))
	assert.Equal(t, blocker1, longest)

	// No match
	matchers, longest = handler.GetMatchingBlockers("/public")
	assert.Equal(t, 0, len(matchers))
	assert.Nil(t, longest)
}

// Test GetMatchingBlockers case sensitivity
func TestGetMatchingBlockersCaseSensitivity(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathrule_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	handler := NewPathRuleHandler(&Options{
		Enabled:      true,
		ConfigFolder: tmpDir,
	})

	blocker1 := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/Admin",
		ExactMatch:   true,
		Enabled:      true,
		CaseSenitive: true,
	}
	blocker2 := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/Public",
		ExactMatch:   true,
		Enabled:      true,
		CaseSenitive: false,
	}

	handler.BlockingPaths = []*BlockingPath{blocker1, blocker2}

	// Case sensitive - exact match only
	matchers, longest := handler.GetMatchingBlockers("/Admin")
	assert.Equal(t, 1, len(matchers))
	assert.Equal(t, blocker1, longest)

	// Case sensitive - no match for different case
	matchers, _ = handler.GetMatchingBlockers("/admin")
	assert.Equal(t, 0, len(matchers))

	// Case insensitive - matches regardless of case
	matchers, longest = handler.GetMatchingBlockers("/public")
	assert.Equal(t, 1, len(matchers))
	assert.Equal(t, blocker2, longest)

	matchers, longest = handler.GetMatchingBlockers("/PUBLIC")
	assert.Equal(t, 1, len(matchers))
	assert.Equal(t, blocker2, longest)
}

// Test GetMatchingBlockers with disabled blocker
func TestGetMatchingBlockersDisabled(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathrule_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	handler := NewPathRuleHandler(&Options{
		Enabled:      true,
		ConfigFolder: tmpDir,
	})

	blocker1 := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/test",
		ExactMatch:   true,
		Enabled:      false, // Disabled
	}

	handler.BlockingPaths = []*BlockingPath{blocker1}

	// Should not match disabled blocker
	matchers, longest := handler.GetMatchingBlockers("/test")
	assert.Equal(t, 0, len(matchers))
	assert.Nil(t, longest)
}

// Test GetMatchingBlockers longest prefix match
func TestGetMatchingBlockersLongestPrefix(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathrule_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	handler := NewPathRuleHandler(&Options{
		Enabled:      true,
		ConfigFolder: tmpDir,
	})

	blocker1 := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/admin",
		ExactMatch:   false,
		Enabled:      true,
	}
	blocker2 := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/admin/users",
		ExactMatch:   false,
		Enabled:      true,
	}

	handler.BlockingPaths = []*BlockingPath{blocker1, blocker2}

	// Should match both but return longest
	matchers, longest := handler.GetMatchingBlockers("/admin/users/edit")
	assert.Equal(t, 2, len(matchers))
	assert.Equal(t, blocker2, longest) // Longest match
	assert.Equal(t, "/admin/users", longest.MatchingPath)
}

// Test SaveBlockerToFile
func TestSaveBlockerToFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathrule_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	handler := NewPathRuleHandler(&Options{
		Enabled:      true,
		ConfigFolder: tmpDir,
	})

	blocker := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/test",
		ExactMatch:   true,
		StatusCode:   404,
		Enabled:      true,
	}

	err = handler.SaveBlockerToFile(blocker)
	assert.NoError(t, err)

	configFile := filepath.Join(tmpDir, blocker.UUID)
	assert.True(t, utils.FileExists(configFile))

	// Verify file content
	data, err := os.ReadFile(configFile)
	assert.NoError(t, err)
	assert.Contains(t, string(data), blocker.UUID)
	assert.Contains(t, string(data), "/test")
}

// Test RemoveBlockerFromFile
func TestRemoveBlockerFromFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathrule_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	handler := NewPathRuleHandler(&Options{
		Enabled:      true,
		ConfigFolder: tmpDir,
	})

	testUUID := uuid.New().String()
	blocker := &BlockingPath{
		UUID:         testUUID,
		MatchingPath: "/test",
		Enabled:      true,
	}

	// Create the file first
	err = handler.SaveBlockerToFile(blocker)
	assert.NoError(t, err)

	configFile := filepath.Join(tmpDir, testUUID)
	assert.True(t, utils.FileExists(configFile))

	// Remove the file
	err = handler.RemoveBlockerFromFile(testUUID)
	assert.NoError(t, err)
	assert.False(t, utils.FileExists(configFile))

	// Try to remove non-existent file
	err = handler.RemoveBlockerFromFile(uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// Test GetMatchingBlockers with multiple overlapping rules
func TestGetMatchingBlockersMultipleRules(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathrule_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	handler := NewPathRuleHandler(&Options{
		Enabled:      true,
		ConfigFolder: tmpDir,
	})

	blocker1 := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/api",
		ExactMatch:   false,
		Enabled:      true,
	}
	blocker2 := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/api/v1",
		ExactMatch:   false,
		Enabled:      true,
	}
	blocker3 := &BlockingPath{
		UUID:         uuid.New().String(),
		MatchingPath: "/api/v1/users",
		ExactMatch:   true,
		Enabled:      true,
	}

	handler.BlockingPaths = []*BlockingPath{blocker1, blocker2, blocker3}

	// Test exact match with overlapping prefix rules
	matchers, longest := handler.GetMatchingBlockers("/api/v1/users")
	assert.Equal(t, 3, len(matchers)) // All three match
	assert.Equal(t, blocker3, longest) // Longest is the exact match

	// Test prefix match with two overlapping rules
	matchers, longest = handler.GetMatchingBlockers("/api/v1/posts")
	assert.Equal(t, 2, len(matchers)) // blocker1 and blocker2 match
	assert.Equal(t, blocker2, longest) // Longest prefix

	// Test single prefix match
	matchers, longest = handler.GetMatchingBlockers("/api/v2/items")
	assert.Equal(t, 1, len(matchers)) // Only blocker1 matches
	assert.Equal(t, blocker1, longest)
}

// Test BlockingPath with custom headers
func TestBlockingPathCustomHeaders(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pathrule_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	handler := NewPathRuleHandler(&Options{
		Enabled:      true,
		ConfigFolder: tmpDir,
	})

	customHeaders := http.Header{}
	customHeaders.Set("X-Custom-Header", "test-value")
	customHeaders.Set("Content-Type", "application/json")

	blocker := &BlockingPath{
		UUID:          uuid.New().String(),
		MatchingPath:  "/api",
		ExactMatch:    true,
		StatusCode:    403,
		CustomHeaders: customHeaders,
		CustomHTML:    []byte(`{"error": "forbidden"}`),
		Enabled:       true,
	}

	err = handler.AddBlockingPath(blocker)
	assert.NoError(t, err)

	// Retrieve and verify
	retrieved := handler.GetPathBlockerFromMatchingPath("/api")
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test-value", retrieved.CustomHeaders.Get("X-Custom-Header"))
	assert.Equal(t, "application/json", retrieved.CustomHeaders.Get("Content-Type"))
	assert.Equal(t, `{"error": "forbidden"}`, string(retrieved.CustomHTML))
}
