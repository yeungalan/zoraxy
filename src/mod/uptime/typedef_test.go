package uptime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test ProxyType constants
func TestProxyTypeConstants(t *testing.T) {
	assert.Equal(t, ProxyType("Origin Server"), ProxyType_Host)
	assert.Equal(t, ProxyType("Virtual Directory"), ProxyType_Vdir)
}

// Test ProxyType string values
func TestProxyTypeStringValues(t *testing.T) {
	assert.Equal(t, "Origin Server", string(ProxyType_Host))
	assert.Equal(t, "Virtual Directory", string(ProxyType_Vdir))
}

// Test Record struct initialization
func TestRecordStruct(t *testing.T) {
	record := Record{
		Timestamp:  1234567890,
		ID:         "test-id",
		Name:       "Test Name",
		URL:        "http://example.com",
		Protocol:   "http",
		Online:     true,
		StatusCode: 200,
		Latency:    100,
	}

	assert.Equal(t, int64(1234567890), record.Timestamp)
	assert.Equal(t, "test-id", record.ID)
	assert.Equal(t, "Test Name", record.Name)
	assert.Equal(t, "http://example.com", record.URL)
	assert.Equal(t, "http", record.Protocol)
	assert.True(t, record.Online)
	assert.Equal(t, 200, record.StatusCode)
	assert.Equal(t, int64(100), record.Latency)
}

// Test Record struct zero values
func TestRecordStructZeroValues(t *testing.T) {
	record := Record{}

	assert.Equal(t, int64(0), record.Timestamp)
	assert.Equal(t, "", record.ID)
	assert.Equal(t, "", record.Name)
	assert.Equal(t, "", record.URL)
	assert.Equal(t, "", record.Protocol)
	assert.False(t, record.Online)
	assert.Equal(t, 0, record.StatusCode)
	assert.Equal(t, int64(0), record.Latency)
}

// Test Target struct initialization
func TestTargetStruct(t *testing.T) {
	target := Target{
		ID:        "target-1",
		Name:      "Test Target",
		URL:       "https://example.com",
		Protocol:  "https",
		ProxyType: ProxyType_Host,
	}

	assert.Equal(t, "target-1", target.ID)
	assert.Equal(t, "Test Target", target.Name)
	assert.Equal(t, "https://example.com", target.URL)
	assert.Equal(t, "https", target.Protocol)
	assert.Equal(t, ProxyType_Host, target.ProxyType)
}

// Test Target struct with Virtual Directory ProxyType
func TestTargetStructWithVdirProxyType(t *testing.T) {
	target := Target{
		ID:        "target-2",
		Name:      "Vdir Target",
		URL:       "https://example.com/path",
		Protocol:  "https",
		ProxyType: ProxyType_Vdir,
	}

	assert.Equal(t, ProxyType_Vdir, target.ProxyType)
	assert.Equal(t, "Virtual Directory", string(target.ProxyType))
}

// Test Target struct zero values
func TestTargetStructZeroValues(t *testing.T) {
	target := Target{}

	assert.Equal(t, "", target.ID)
	assert.Equal(t, "", target.Name)
	assert.Equal(t, "", target.URL)
	assert.Equal(t, "", target.Protocol)
	assert.Equal(t, ProxyType(""), target.ProxyType)
}

// Test Config struct initialization
func TestConfigStruct(t *testing.T) {
	target := &Target{
		ID:       "test",
		Name:     "Test",
		URL:      "http://test.com",
		Protocol: "http",
	}

	notifyFunc := func(upstreamIP string, isOnline bool) {}

	config := Config{
		Targets:           []*Target{target},
		Interval:          60,
		MaxRecordsStore:   100,
		OnlineStateNotify: notifyFunc,
		Logger:            nil,
	}

	assert.Len(t, config.Targets, 1)
	assert.Equal(t, 60, config.Interval)
	assert.Equal(t, 100, config.MaxRecordsStore)
	assert.NotNil(t, config.OnlineStateNotify)
	assert.Nil(t, config.Logger)
}

// Test Config struct with multiple targets
func TestConfigStructMultipleTargets(t *testing.T) {
	targets := []*Target{
		{ID: "t1", Name: "Target 1", URL: "http://1.com", Protocol: "http"},
		{ID: "t2", Name: "Target 2", URL: "http://2.com", Protocol: "http"},
		{ID: "t3", Name: "Target 3", URL: "http://3.com", Protocol: "https"},
	}

	config := Config{
		Targets:         targets,
		Interval:        30,
		MaxRecordsStore: 50,
	}

	assert.Len(t, config.Targets, 3)
	assert.Equal(t, 30, config.Interval)
	assert.Equal(t, 50, config.MaxRecordsStore)
}

// Test Config struct with nil targets
func TestConfigStructNilTargets(t *testing.T) {
	config := Config{
		Targets:         nil,
		Interval:        60,
		MaxRecordsStore: 100,
	}

	assert.Nil(t, config.Targets)
}

// Test Config struct with empty targets slice
func TestConfigStructEmptyTargets(t *testing.T) {
	config := Config{
		Targets:         []*Target{},
		Interval:        60,
		MaxRecordsStore: 100,
	}

	assert.NotNil(t, config.Targets)
	assert.Len(t, config.Targets, 0)
}

// Test Monitor struct initialization
func TestMonitorStruct(t *testing.T) {
	config := &Config{
		Targets:         []*Target{},
		Interval:        60,
		MaxRecordsStore: 100,
	}

	monitor := Monitor{
		Config:          config,
		OnlineStatusLog: make(map[string][]*Record),
	}

	assert.NotNil(t, monitor.Config)
	assert.NotNil(t, monitor.OnlineStatusLog)
	assert.Empty(t, monitor.OnlineStatusLog)
}

// Test Monitor struct with pre-populated status log
func TestMonitorStructWithStatusLog(t *testing.T) {
	config := &Config{
		Targets:         []*Target{},
		Interval:        60,
		MaxRecordsStore: 100,
	}

	statusLog := map[string][]*Record{
		"target1": {
			{ID: "target1", Online: true},
			{ID: "target1", Online: false},
		},
		"target2": {
			{ID: "target2", Online: true},
		},
	}

	monitor := Monitor{
		Config:          config,
		OnlineStatusLog: statusLog,
	}

	assert.Len(t, monitor.OnlineStatusLog, 2)
	assert.Len(t, monitor.OnlineStatusLog["target1"], 2)
	assert.Len(t, monitor.OnlineStatusLog["target2"], 1)
}

// Test example target constant
func TestExampleTarget(t *testing.T) {
	assert.Equal(t, "example", exampleTarget.ID)
	assert.Equal(t, "Example", exampleTarget.Name)
	assert.Equal(t, "example.com", exampleTarget.URL)
	assert.Equal(t, "https", exampleTarget.Protocol)
}

// Test logModuleName constant
func TestLogModuleName(t *testing.T) {
	assert.Equal(t, "uptime-monitor", logModuleName)
}

// Test ProxyType custom type
func TestProxyTypeCustomType(t *testing.T) {
	var pt ProxyType
	pt = "Custom Proxy Type"

	assert.Equal(t, "Custom Proxy Type", string(pt))

	// Test type conversion
	pt = ProxyType_Host
	assert.Equal(t, ProxyType("Origin Server"), pt)
}

// Test Record with negative values
func TestRecordWithNegativeValues(t *testing.T) {
	record := Record{
		Timestamp:  -1,
		StatusCode: -1,
		Latency:    -1,
	}

	assert.Equal(t, int64(-1), record.Timestamp)
	assert.Equal(t, -1, record.StatusCode)
	assert.Equal(t, int64(-1), record.Latency)
}

// Test Record with large values
func TestRecordWithLargeValues(t *testing.T) {
	record := Record{
		Timestamp:  9223372036854775807, // Max int64
		StatusCode: 999,
		Latency:    9999999999,
	}

	assert.Equal(t, int64(9223372036854775807), record.Timestamp)
	assert.Equal(t, 999, record.StatusCode)
	assert.Equal(t, int64(9999999999), record.Latency)
}

// Test Config with extreme values
func TestConfigWithExtremeValues(t *testing.T) {
	config := Config{
		Interval:        1,    // Minimum interval
		MaxRecordsStore: 1,    // Minimum records
	}

	assert.Equal(t, 1, config.Interval)
	assert.Equal(t, 1, config.MaxRecordsStore)

	config2 := Config{
		Interval:        2147483647, // Large interval
		MaxRecordsStore: 2147483647, // Large records
	}

	assert.Equal(t, 2147483647, config2.Interval)
	assert.Equal(t, 2147483647, config2.MaxRecordsStore)
}

// Test Config with zero values
func TestConfigZeroValues(t *testing.T) {
	config := Config{
		Interval:        0,
		MaxRecordsStore: 0,
	}

	assert.Equal(t, 0, config.Interval)
	assert.Equal(t, 0, config.MaxRecordsStore)
}

// Test Target with different protocols
func TestTargetWithDifferentProtocols(t *testing.T) {
	protocols := []string{"http", "https", "ftp", "ws", "wss", "custom"}

	for _, proto := range protocols {
		target := Target{
			ID:       "test-" + proto,
			Protocol: proto,
		}
		assert.Equal(t, proto, target.Protocol)
	}
}

// Test Target with empty protocol
func TestTargetWithEmptyProtocol(t *testing.T) {
	target := Target{
		ID:       "test",
		Protocol: "",
	}

	assert.Equal(t, "", target.Protocol)
}

// Test Record with empty strings
func TestRecordWithEmptyStrings(t *testing.T) {
	record := Record{
		ID:       "",
		Name:     "",
		URL:      "",
		Protocol: "",
	}

	assert.Equal(t, "", record.ID)
	assert.Equal(t, "", record.Name)
	assert.Equal(t, "", record.URL)
	assert.Equal(t, "", record.Protocol)
}

// Test Target with special characters in URL
func TestTargetWithSpecialCharactersInURL(t *testing.T) {
	target := Target{
		ID:       "special",
		URL:      "https://example.com/path?query=value&other=123#fragment",
		Protocol: "https",
	}

	assert.Equal(t, "https://example.com/path?query=value&other=123#fragment", target.URL)
}

// Test Target with unicode in name
func TestTargetWithUnicodeName(t *testing.T) {
	target := Target{
		ID:   "unicode",
		Name: "测试目标 🚀",
		URL:  "https://example.com",
	}

	assert.Equal(t, "测试目标 🚀", target.Name)
}

// Test OnlineStatusLog with nil records
func TestOnlineStatusLogWithNilRecords(t *testing.T) {
	statusLog := map[string][]*Record{
		"target1": nil,
	}

	assert.Nil(t, statusLog["target1"])
}

// Test OnlineStatusLog with mixed record states
func TestOnlineStatusLogWithMixedRecordStates(t *testing.T) {
	statusLog := map[string][]*Record{
		"target1": {
			{ID: "target1", Online: true, StatusCode: 200},
			{ID: "target1", Online: false, StatusCode: 500},
			{ID: "target1", Online: true, StatusCode: 200},
			{ID: "target1", Online: false, StatusCode: 404},
		},
	}

	assert.Len(t, statusLog["target1"], 4)
	assert.True(t, statusLog["target1"][0].Online)
	assert.False(t, statusLog["target1"][1].Online)
	assert.True(t, statusLog["target1"][2].Online)
	assert.False(t, statusLog["target1"][3].Online)
}
