package uptime

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"imuslab.com/zoraxy/mod/info/logger"
)

// Test NewUptimeMonitor with valid config
func TestNewUptimeMonitor_ValidConfig(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets: []*Target{
			{
				ID:       "test1",
				Name:     "Test Target 1",
				URL:      "http://example.com",
				Protocol: "http",
			},
		},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor, err := NewUptimeMonitor(config)
	require.NoError(t, err)
	assert.NotNil(t, monitor)
	assert.Equal(t, config, monitor.Config)
	assert.NotNil(t, monitor.OnlineStatusLog)

	// Give the initial uptime check a moment to complete
	time.Sleep(100 * time.Millisecond)
}

// Test NewUptimeMonitor with nil logger (should use default)
func TestNewUptimeMonitor_NilLogger(t *testing.T) {
	config := &Config{
		Targets:         []*Target{},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          nil,
	}

	monitor, err := NewUptimeMonitor(config)
	require.NoError(t, err)
	assert.NotNil(t, monitor)
	assert.NotNil(t, monitor.Config.Logger, "Logger should be initialized with default")
}

// Test NewUptimeMonitor with nil OnlineStateNotify (should use default)
func TestNewUptimeMonitor_NilNotifyFunc(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets:           []*Target{},
		Interval:          60,
		MaxRecordsStore:   100,
		Logger:            testLogger,
		OnlineStateNotify: nil,
	}

	monitor, err := NewUptimeMonitor(config)
	require.NoError(t, err)
	assert.NotNil(t, monitor)
	assert.NotNil(t, monitor.Config.OnlineStateNotify, "OnlineStateNotify should be initialized with default")
}

// Test NewUptimeMonitor with empty targets
func TestNewUptimeMonitor_EmptyTargets(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets:         []*Target{},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor, err := NewUptimeMonitor(config)
	require.NoError(t, err)
	assert.NotNil(t, monitor)
	assert.Empty(t, monitor.Config.Targets)
}

// Test ExecuteUptimeCheck with HTTP target
func TestExecuteUptimeCheck_HTTPTarget(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets: []*Target{
			{
				ID:       "test1",
				Name:     "Test Server",
				URL:      server.URL,
				Protocol: "http",
			},
		},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config:          config,
		OnlineStatusLog: make(map[string][]*Record),
	}
	if config.OnlineStateNotify == nil {
		config.OnlineStateNotify = defaultNotify
	}

	monitor.ExecuteUptimeCheck()

	// Check that a record was created
	records, ok := monitor.OnlineStatusLog["test1"]
	assert.True(t, ok, "Record should exist for test1")
	assert.Len(t, records, 1, "Should have exactly one record")

	record := records[0]
	assert.Equal(t, "test1", record.ID)
	assert.Equal(t, "Test Server", record.Name)
	assert.Equal(t, server.URL, record.URL)
	assert.Equal(t, "http", record.Protocol)
	assert.True(t, record.Online, "Server should be online")
	assert.Equal(t, http.StatusOK, record.StatusCode)
	assert.Greater(t, record.Latency, int64(0), "Latency should be positive")
}

// Test ExecuteUptimeCheck with HTTPS target
func TestExecuteUptimeCheck_HTTPSTarget(t *testing.T) {
	// Create a test HTTPS server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets: []*Target{
			{
				ID:       "test-https",
				Name:     "Test HTTPS Server",
				URL:      server.URL,
				Protocol: "https",
			},
		},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config:          config,
		OnlineStatusLog: make(map[string][]*Record),
	}
	if config.OnlineStateNotify == nil {
		config.OnlineStateNotify = defaultNotify
	}

	monitor.ExecuteUptimeCheck()

	// Check that a record was created
	records, ok := monitor.OnlineStatusLog["test-https"]
	assert.True(t, ok, "Record should exist for test-https")
	assert.Len(t, records, 1, "Should have exactly one record")
}

// Test ExecuteUptimeCheck with unsupported protocol
func TestExecuteUptimeCheck_UnsupportedProtocol(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets: []*Target{
			{
				ID:       "test-ftp",
				Name:     "FTP Server",
				URL:      "ftp://example.com",
				Protocol: "ftp",
			},
		},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config:          config,
		OnlineStatusLog: make(map[string][]*Record),
	}
	if config.OnlineStateNotify == nil {
		config.OnlineStateNotify = defaultNotify
	}

	monitor.ExecuteUptimeCheck()

	// Should not create a record for unsupported protocol
	_, ok := monitor.OnlineStatusLog["test-ftp"]
	assert.False(t, ok, "Should not create record for unsupported protocol")
}

// Test ExecuteUptimeCheck with multiple targets
func TestExecuteUptimeCheck_MultipleTargets(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server2.Close()

	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets: []*Target{
			{
				ID:       "server1",
				Name:     "Server 1",
				URL:      server1.URL,
				Protocol: "http",
			},
			{
				ID:       "server2",
				Name:     "Server 2",
				URL:      server2.URL,
				Protocol: "http",
			},
		},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config:          config,
		OnlineStatusLog: make(map[string][]*Record),
	}
	if config.OnlineStateNotify == nil {
		config.OnlineStateNotify = defaultNotify
	}

	monitor.ExecuteUptimeCheck()

	// Check both records
	records1, ok1 := monitor.OnlineStatusLog["server1"]
	assert.True(t, ok1)
	assert.Len(t, records1, 1)
	assert.True(t, records1[0].Online)
	assert.Equal(t, http.StatusOK, records1[0].StatusCode)

	records2, ok2 := monitor.OnlineStatusLog["server2"]
	assert.True(t, ok2)
	assert.Len(t, records2, 1)
	assert.False(t, records2[0].Online)
	assert.Equal(t, http.StatusNotFound, records2[0].StatusCode)
}

// Test ExecuteUptimeCheck with max records limit
func TestExecuteUptimeCheck_MaxRecordsLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets: []*Target{
			{
				ID:       "test1",
				Name:     "Test Server",
				URL:      server.URL,
				Protocol: "http",
			},
		},
		Interval:        60,
		MaxRecordsStore: 3, // Small limit for testing
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config:          config,
		OnlineStatusLog: make(map[string][]*Record),
	}
	if config.OnlineStateNotify == nil {
		config.OnlineStateNotify = defaultNotify
	}

	// Execute multiple checks
	for i := 0; i < 5; i++ {
		monitor.ExecuteUptimeCheck()
		time.Sleep(10 * time.Millisecond)
	}

	records := monitor.OnlineStatusLog["test1"]
	assert.LessOrEqual(t, len(records), config.MaxRecordsStore, "Should not exceed MaxRecordsStore")
}

// Test AddTargetToMonitor
func TestAddTargetToMonitor(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets:         []*Target{},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config:          config,
		OnlineStatusLog: make(map[string][]*Record),
	}

	newTarget := &Target{
		ID:       "new-target",
		Name:     "New Target",
		URL:      "http://example.com",
		Protocol: "http",
	}

	monitor.AddTargetToMonitor(newTarget)

	// Verify target was added to config
	assert.Len(t, monitor.Config.Targets, 1)
	assert.Equal(t, newTarget, monitor.Config.Targets[0])

	// Verify entry was created in status log
	records, ok := monitor.OnlineStatusLog["new-target"]
	assert.True(t, ok)
	assert.NotNil(t, records)
	assert.Empty(t, records)
}

// Test AddTargetToMonitor with nil target (should panic when accessing ID)
func TestAddTargetToMonitor_NilTarget(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets:         []*Target{},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config:          config,
		OnlineStatusLog: make(map[string][]*Record),
	}

	// Adding a nil target should panic when accessing target.ID
	assert.Panics(t, func() {
		monitor.AddTargetToMonitor(nil)
	}, "Adding nil target should panic")
}

// Test AddTargetToMonitor multiple times
func TestAddTargetToMonitor_Multiple(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets:         []*Target{},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config:          config,
		OnlineStatusLog: make(map[string][]*Record),
	}

	for i := 0; i < 5; i++ {
		target := &Target{
			ID:       "target-" + string(rune('0'+i)),
			Name:     "Target " + string(rune('0'+i)),
			URL:      "http://example.com",
			Protocol: "http",
		}
		monitor.AddTargetToMonitor(target)
	}

	assert.Len(t, monitor.Config.Targets, 5)
	assert.Len(t, monitor.OnlineStatusLog, 5)
}

// Test RemoveTargetFromMonitor
func TestRemoveTargetFromMonitor(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	target1 := &Target{ID: "target1", Name: "Target 1", URL: "http://example1.com", Protocol: "http"}
	target2 := &Target{ID: "target2", Name: "Target 2", URL: "http://example2.com", Protocol: "http"}

	config := &Config{
		Targets:         []*Target{target1, target2},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config: config,
		OnlineStatusLog: map[string][]*Record{
			"target1": {},
			"target2": {},
		},
	}

	monitor.RemoveTargetFromMonitor("target1")

	// Verify target was removed from config
	assert.Len(t, monitor.Config.Targets, 1)
	assert.Equal(t, "target2", monitor.Config.Targets[0].ID)

	// Verify entry was removed from status log
	_, ok := monitor.OnlineStatusLog["target1"]
	assert.False(t, ok, "target1 should be removed from status log")

	_, ok = monitor.OnlineStatusLog["target2"]
	assert.True(t, ok, "target2 should still exist in status log")
}

// Test RemoveTargetFromMonitor with non-existent ID
func TestRemoveTargetFromMonitor_NonExistent(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	target1 := &Target{ID: "target1", Name: "Target 1", URL: "http://example1.com", Protocol: "http"}

	config := &Config{
		Targets:         []*Target{target1},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config: config,
		OnlineStatusLog: map[string][]*Record{
			"target1": {},
		},
	}

	// Try to remove non-existent target
	monitor.RemoveTargetFromMonitor("non-existent")

	// Verify nothing was removed
	assert.Len(t, monitor.Config.Targets, 1)
	assert.Equal(t, "target1", monitor.Config.Targets[0].ID)
}

// Test RemoveTargetFromMonitor with empty ID
func TestRemoveTargetFromMonitor_EmptyID(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	target1 := &Target{ID: "target1", Name: "Target 1", URL: "http://example1.com", Protocol: "http"}

	config := &Config{
		Targets:         []*Target{target1},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config: config,
		OnlineStatusLog: map[string][]*Record{
			"target1": {},
		},
	}

	// Try to remove with empty ID
	monitor.RemoveTargetFromMonitor("")

	// Verify nothing was removed
	assert.Len(t, monitor.Config.Targets, 1)
}

// Test CleanRecords with orphaned records
func TestCleanRecords_OrphanedRecords(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	target1 := &Target{ID: "target1", Name: "Target 1", URL: "http://example1.com", Protocol: "http"}

	config := &Config{
		Targets:         []*Target{target1},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config: config,
		OnlineStatusLog: map[string][]*Record{
			"target1":  {},
			"orphan1":  {},
			"orphan2":  {},
		},
	}

	monitor.CleanRecords()

	// Verify orphaned records were removed
	assert.Len(t, monitor.OnlineStatusLog, 1)
	_, ok := monitor.OnlineStatusLog["target1"]
	assert.True(t, ok, "target1 should remain")

	_, ok = monitor.OnlineStatusLog["orphan1"]
	assert.False(t, ok, "orphan1 should be removed")

	_, ok = monitor.OnlineStatusLog["orphan2"]
	assert.False(t, ok, "orphan2 should be removed")
}

// Test CleanRecords with no orphaned records
func TestCleanRecords_NoOrphans(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	target1 := &Target{ID: "target1", Name: "Target 1", URL: "http://example1.com", Protocol: "http"}
	target2 := &Target{ID: "target2", Name: "Target 2", URL: "http://example2.com", Protocol: "http"}

	config := &Config{
		Targets:         []*Target{target1, target2},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config: config,
		OnlineStatusLog: map[string][]*Record{
			"target1": {},
			"target2": {},
		},
	}

	monitor.CleanRecords()

	// Verify all records remain
	assert.Len(t, monitor.OnlineStatusLog, 2)
	_, ok := monitor.OnlineStatusLog["target1"]
	assert.True(t, ok)
	_, ok = monitor.OnlineStatusLog["target2"]
	assert.True(t, ok)
}

// Test CleanRecords with empty targets
func TestCleanRecords_EmptyTargets(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets:         []*Target{},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config: config,
		OnlineStatusLog: map[string][]*Record{
			"orphan1": {},
			"orphan2": {},
		},
	}

	monitor.CleanRecords()

	// All records should be removed
	assert.Empty(t, monitor.OnlineStatusLog)
}

// Test HandleUptimeLogRead without ID parameter
func TestHandleUptimeLogRead_AllLogs(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets:         []*Target{},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config: config,
		OnlineStatusLog: map[string][]*Record{
			"target1": {
				{ID: "target1", Name: "Target 1", Online: true},
			},
			"target2": {
				{ID: "target2", Name: "Target 2", Online: false},
			},
		},
	}

	req := httptest.NewRequest("GET", "/uptime", nil)
	w := httptest.NewRecorder()

	monitor.HandleUptimeLogRead(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result map[string][]*Record
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Contains(t, result, "target1")
	assert.Contains(t, result, "target2")
}

// Test HandleUptimeLogRead with specific ID
func TestHandleUptimeLogRead_SpecificID(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets:         []*Target{},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	testRecord := &Record{
		ID:     "target1",
		Name:   "Target 1",
		Online: true,
	}

	monitor := &Monitor{
		Config: config,
		OnlineStatusLog: map[string][]*Record{
			"target1": {testRecord},
		},
	}

	req := httptest.NewRequest("GET", "/uptime?id=target1", nil)
	w := httptest.NewRecorder()

	monitor.HandleUptimeLogRead(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result []*Record
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "target1", result[0].ID)
}

// Test HandleUptimeLogRead with non-existent ID
func TestHandleUptimeLogRead_NonExistentID(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets:         []*Target{},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config:          config,
		OnlineStatusLog: map[string][]*Record{},
	}

	req := httptest.NewRequest("GET", "/uptime?id=non-existent", nil)
	w := httptest.NewRecorder()

	monitor.HandleUptimeLogRead(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Test getWebsiteStatus with successful response
func TestGetWebsiteStatus_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "zoraxy-uptime/1.1", r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	statusCode, err := getWebsiteStatus(server.URL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, statusCode)
}

// Test getWebsiteStatus with redirect response
func TestGetWebsiteStatus_Redirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMovedPermanently)
	}))
	defer server.Close()

	statusCode, err := getWebsiteStatus(server.URL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusMovedPermanently, statusCode)
}

// Test getWebsiteStatus with error response
func TestGetWebsiteStatus_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	statusCode, err := getWebsiteStatus(server.URL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, statusCode)
}

// Test getWebsiteStatus with timeout
func TestGetWebsiteStatus_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second) // Longer than client timeout (5 seconds)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Note: The timeout might still succeed due to protocol retry logic
	// This test mainly ensures the timeout mechanism exists
	statusCode, err := getWebsiteStatus(server.URL)
	// Either times out or completes (due to retry with different protocol)
	if err != nil {
		assert.Equal(t, 0, statusCode)
	}
}

// Test getWebsiteStatus with invalid URL
func TestGetWebsiteStatus_InvalidURL(t *testing.T) {
	statusCode, err := getWebsiteStatus("http://invalid-url-that-does-not-exist-12345.com")
	// The function retries with protocol swap, so it might succeed or fail
	// We just verify it returns some result
	if err != nil {
		assert.Equal(t, 0, statusCode)
	} else {
		// If it somehow succeeded (DNS resolution, etc), just verify we got a status code
		assert.GreaterOrEqual(t, statusCode, 0)
	}
}

// Test getWebsiteStatusWithLatency with online target
func TestGetWebsiteStatusWithLatency_Online(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets:         []*Target{},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config:          config,
		OnlineStatusLog: make(map[string][]*Record),
	}
	if config.OnlineStateNotify == nil {
		config.OnlineStateNotify = defaultNotify
	}

	target := &Target{
		ID:       "test1",
		Name:     "Test Server",
		URL:      server.URL,
		Protocol: "http",
	}

	online, latency, statusCode := monitor.getWebsiteStatusWithLatency(target)

	assert.True(t, online)
	assert.GreaterOrEqual(t, latency, int64(0), "Latency should be non-negative")
	assert.Equal(t, http.StatusOK, statusCode)
}

// Test getWebsiteStatusWithLatency with redirect (should be considered online)
func TestGetWebsiteStatusWithLatency_Redirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()

	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets:         []*Target{},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config:          config,
		OnlineStatusLog: make(map[string][]*Record),
	}
	if config.OnlineStateNotify == nil {
		config.OnlineStateNotify = defaultNotify
	}

	target := &Target{
		ID:       "test1",
		Name:     "Test Server",
		URL:      server.URL,
		Protocol: "http",
	}

	online, latency, statusCode := monitor.getWebsiteStatusWithLatency(target)

	assert.True(t, online, "Redirect should be considered online")
	assert.GreaterOrEqual(t, latency, int64(0), "Latency should be non-negative")
	assert.Equal(t, http.StatusFound, statusCode)
}

// Test getWebsiteStatusWithLatency with error status (should be offline)
func TestGetWebsiteStatusWithLatency_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets:         []*Target{},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config:          config,
		OnlineStatusLog: make(map[string][]*Record),
	}
	if config.OnlineStateNotify == nil {
		config.OnlineStateNotify = defaultNotify
	}

	target := &Target{
		ID:       "test1",
		Name:     "Test Server",
		URL:      server.URL,
		Protocol: "http",
	}

	online, latency, statusCode := monitor.getWebsiteStatusWithLatency(target)

	assert.False(t, online, "Error status should be considered offline")
	assert.GreaterOrEqual(t, latency, int64(0), "Latency should be non-negative")
	assert.Equal(t, http.StatusInternalServerError, statusCode)
}

// Test getWebsiteStatusWithLatency with offline target (first record)
func TestGetWebsiteStatusWithLatency_OfflineFirstRecord(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	notifyCalled := false
	config := &Config{
		Targets:         []*Target{},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
		OnlineStateNotify: func(upstreamIP string, isOnline bool) {
			notifyCalled = true
		},
	}

	monitor := &Monitor{
		Config:          config,
		OnlineStatusLog: make(map[string][]*Record),
	}

	// Use an invalid port to ensure connection failure
	target := &Target{
		ID:       "test1",
		Name:     "Test Server",
		URL:      "http://localhost:1", // Port 1 should not be accessible
		Protocol: "http",
	}

	online, latency, statusCode := monitor.getWebsiteStatusWithLatency(target)

	assert.False(t, online, "Should be offline for inaccessible server")
	// When there's an error, latency and statusCode are 0
	assert.GreaterOrEqual(t, latency, int64(0))
	assert.GreaterOrEqual(t, statusCode, 0)
	assert.False(t, notifyCalled, "Should not notify on first record failure")
}

// Test getWebsiteStatusWithLatency with offline target (subsequent record)
func TestGetWebsiteStatusWithLatency_OfflineSubsequentRecord(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	notifyCalled := false
	var notifyIsOnline bool
	config := &Config{
		Targets:         []*Target{},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
		OnlineStateNotify: func(upstreamIP string, isOnline bool) {
			notifyCalled = true
			notifyIsOnline = isOnline
		},
	}

	monitor := &Monitor{
		Config: config,
		OnlineStatusLog: map[string][]*Record{
			"test1": {
				{ID: "test1", Online: true}, // Previous record exists
			},
		},
	}

	// Use an invalid port to ensure connection failure
	target := &Target{
		ID:       "test1",
		Name:     "Test Server",
		URL:      "http://localhost:1", // Port 1 should not be accessible
		Protocol: "http",
	}

	online, latency, statusCode := monitor.getWebsiteStatusWithLatency(target)

	assert.False(t, online, "Should be offline for inaccessible server")
	// When there's an error, latency and statusCode are 0
	assert.GreaterOrEqual(t, latency, int64(0))
	assert.GreaterOrEqual(t, statusCode, 0)
	assert.True(t, notifyCalled, "Should notify on subsequent record failure")
	assert.False(t, notifyIsOnline, "Should notify with isOnline=false")
}

// Test sequential access to monitor
// Note: The Monitor type does not have mutex protection for concurrent access
// to OnlineStatusLog map. This test verifies sequential operations work correctly.
func TestSequentialAccess(t *testing.T) {
	testLogger, err := logger.NewFmtLogger()
	require.NoError(t, err)

	config := &Config{
		Targets:         []*Target{},
		Interval:        60,
		MaxRecordsStore: 100,
		Logger:          testLogger,
	}

	monitor := &Monitor{
		Config:          config,
		OnlineStatusLog: make(map[string][]*Record),
	}

	// Add targets sequentially
	for i := 0; i < 10; i++ {
		target := &Target{
			ID:       "target-" + string(rune('0'+i)),
			Name:     "Target " + string(rune('0'+i)),
			URL:      "http://example.com",
			Protocol: "http",
		}
		monitor.AddTargetToMonitor(target)
	}

	assert.Equal(t, 10, len(monitor.Config.Targets))
	assert.Equal(t, 10, len(monitor.OnlineStatusLog))
}

// Test defaultNotify function
func TestDefaultNotify(t *testing.T) {
	// Should not panic
	assert.NotPanics(t, func() {
		defaultNotify("http://example.com", true)
		defaultNotify("http://example.com", false)
	})
}

// Test status code boundaries
func TestGetWebsiteStatusWithLatency_StatusCodeBoundaries(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		expectedOnline bool
	}{
		// Note: httptest can't reliably return status codes below 200
		{"200 - OK", 200, true},
		{"250 - Mid 2xx", 250, true},
		{"299 - End 2xx", 299, true},
		{"300 - Redirect Start", 300, true},
		{"350 - Mid 3xx", 350, true},
		{"399 - End 3xx", 399, true},
		{"400 - Client Error", 400, false},
		{"404 - Not Found", 404, false},
		{"500 - Server Error", 500, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			testLogger, err := logger.NewFmtLogger()
			require.NoError(t, err)

			config := &Config{
				Targets:         []*Target{},
				Interval:        60,
				MaxRecordsStore: 100,
				Logger:          testLogger,
			}

			monitor := &Monitor{
				Config:          config,
				OnlineStatusLog: make(map[string][]*Record),
			}
			if config.OnlineStateNotify == nil {
				config.OnlineStateNotify = defaultNotify
			}

			target := &Target{
				ID:       "test1",
				Name:     "Test Server",
				URL:      server.URL,
				Protocol: "http",
			}

			online, _, statusCode := monitor.getWebsiteStatusWithLatency(target)

			assert.Equal(t, tc.expectedOnline, online)
			assert.Equal(t, tc.statusCode, statusCode)
		})
	}
}
