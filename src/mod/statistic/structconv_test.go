package statistic_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"imuslab.com/zoraxy/mod/statistic"
)

func TestSyncMapToMapStringInt(t *testing.T) {
	t.Run("Empty SyncMap", func(t *testing.T) {
		sm := &sync.Map{}
		result := statistic.SyncMapToMapStringInt(sm)
		assert.NotNil(t, result)
		assert.Equal(t, 0, len(result))
	})

	t.Run("SyncMap with String Keys and Int Values", func(t *testing.T) {
		sm := &sync.Map{}
		sm.Store("key1", 10)
		sm.Store("key2", 20)
		sm.Store("key3", 30)

		result := statistic.SyncMapToMapStringInt(sm)
		assert.Equal(t, 3, len(result))
		assert.Equal(t, 10, result["key1"])
		assert.Equal(t, 20, result["key2"])
		assert.Equal(t, 30, result["key3"])
	})

	t.Run("SyncMap with Invalid Types", func(t *testing.T) {
		sm := &sync.Map{}
		sm.Store("key1", 10)
		sm.Store(123, 20) // Invalid key type
		sm.Store("key3", "invalid") // Invalid value type

		result := statistic.SyncMapToMapStringInt(sm)
		// Should only include valid entries
		assert.Equal(t, 1, len(result))
		assert.Equal(t, 10, result["key1"])
	})

	t.Run("SyncMap with Mixed Valid and Invalid", func(t *testing.T) {
		sm := &sync.Map{}
		sm.Store("valid1", 100)
		sm.Store("valid2", 200)
		sm.Store(456, "invalid")

		result := statistic.SyncMapToMapStringInt(sm)
		assert.Equal(t, 2, len(result))
		assert.Equal(t, 100, result["valid1"])
		assert.Equal(t, 200, result["valid2"])
	})
}

func TestMapStringIntToSyncMap(t *testing.T) {
	t.Run("Empty Map", func(t *testing.T) {
		m := make(map[string]int)
		result := statistic.MapStringIntToSyncMap(m)
		assert.NotNil(t, result)

		count := 0
		result.Range(func(key, value interface{}) bool {
			count++
			return true
		})
		assert.Equal(t, 0, count)
	})

	t.Run("Map with Values", func(t *testing.T) {
		m := map[string]int{
			"key1": 10,
			"key2": 20,
			"key3": 30,
		}
		result := statistic.MapStringIntToSyncMap(m)
		assert.NotNil(t, result)

		// Verify all values are stored
		val, ok := result.Load("key1")
		assert.True(t, ok)
		assert.Equal(t, 10, val.(int))

		val, ok = result.Load("key2")
		assert.True(t, ok)
		assert.Equal(t, 20, val.(int))

		val, ok = result.Load("key3")
		assert.True(t, ok)
		assert.Equal(t, 30, val.(int))
	})

	t.Run("Nil Map", func(t *testing.T) {
		var m map[string]int
		result := statistic.MapStringIntToSyncMap(m)
		assert.NotNil(t, result)

		count := 0
		result.Range(func(key, value interface{}) bool {
			count++
			return true
		})
		assert.Equal(t, 0, count)
	})
}

func TestDailySummaryToExport(t *testing.T) {
	t.Run("Convert Summary with Data", func(t *testing.T) {
		summary := statistic.NewDailySummary()
		summary.TotalRequest = 100
		summary.ErrorRequest = 10
		summary.ValidRequest = 90
		summary.ForwardTypes.Store("host-http", 50)
		summary.ForwardTypes.Store("subdomain-http", 50)
		summary.RequestOrigin.Store("us", 60)
		summary.RequestOrigin.Store("uk", 40)

		export := statistic.DailySummaryToExport(*summary)

		assert.Equal(t, int64(100), export.TotalRequest)
		assert.Equal(t, int64(10), export.ErrorRequest)
		assert.Equal(t, int64(90), export.ValidRequest)
		assert.Equal(t, 50, export.ForwardTypes["host-http"])
		assert.Equal(t, 50, export.ForwardTypes["subdomain-http"])
		assert.Equal(t, 60, export.RequestOrigin["us"])
		assert.Equal(t, 40, export.RequestOrigin["uk"])
	})

	t.Run("Convert Empty Summary", func(t *testing.T) {
		summary := statistic.NewDailySummary()

		export := statistic.DailySummaryToExport(*summary)

		assert.Equal(t, int64(0), export.TotalRequest)
		assert.Equal(t, int64(0), export.ErrorRequest)
		assert.Equal(t, int64(0), export.ValidRequest)
		assert.NotNil(t, export.ForwardTypes)
		assert.Equal(t, 0, len(export.ForwardTypes))
	})
}

func TestDailySummaryExportToSummary(t *testing.T) {
	t.Run("Convert Export with Data", func(t *testing.T) {
		export := statistic.DailySummaryExport{
			TotalRequest: 200,
			ErrorRequest: 20,
			ValidRequest: 180,
			ForwardTypes: map[string]int{
				"host-http":      100,
				"subdomain-http": 100,
			},
			RequestOrigin: map[string]int{
				"us": 120,
				"uk": 80,
			},
			RequestClientIp: map[string]int{
				"192.168.1.1": 50,
			},
			Referer: map[string]int{
				"http://example.com": 100,
			},
			UserAgent: map[string]int{
				"Mozilla/5.0": 150,
			},
			RequestURL: map[string]int{
				"/test": 200,
			},
			Downstreams: map[string]int{
				"target1": 100,
			},
			Upstreams: map[string]int{
				"upstream1": 100,
			},
		}

		summary := statistic.DailySummaryExportToSummary(export)

		assert.Equal(t, int64(200), summary.TotalRequest)
		assert.Equal(t, int64(20), summary.ErrorRequest)
		assert.Equal(t, int64(180), summary.ValidRequest)

		val, ok := summary.ForwardTypes.Load("host-http")
		assert.True(t, ok)
		assert.Equal(t, 100, val.(int))

		val, ok = summary.RequestOrigin.Load("us")
		assert.True(t, ok)
		assert.Equal(t, 120, val.(int))
	})

	t.Run("Convert Empty Export", func(t *testing.T) {
		export := statistic.DailySummaryExport{}

		summary := statistic.DailySummaryExportToSummary(export)

		assert.Equal(t, int64(0), summary.TotalRequest)
		assert.NotNil(t, summary.ForwardTypes)
		assert.NotNil(t, summary.RequestOrigin)
	})
}

func TestGetExportSummary(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, err := statistic.NewStatisticCollector(option)
	assert.NoError(t, err)
	defer collector.Close()

	// Add some data
	collector.DailySummary.TotalRequest = 500
	collector.DailySummary.ErrorRequest = 50
	collector.DailySummary.ValidRequest = 450
	collector.DailySummary.ForwardTypes.Store("host-http", 250)
	collector.DailySummary.ForwardTypes.Store("subdomain-http", 250)

	export := collector.GetExportSummary()

	assert.NotNil(t, export)
	assert.Equal(t, int64(500), export.TotalRequest)
	assert.Equal(t, int64(50), export.ErrorRequest)
	assert.Equal(t, int64(450), export.ValidRequest)
	assert.Equal(t, 250, export.ForwardTypes["host-http"])
	assert.Equal(t, 250, export.ForwardTypes["subdomain-http"])
}

func TestRoundTripConversion(t *testing.T) {
	// Create original summary
	original := statistic.NewDailySummary()
	original.TotalRequest = 1000
	original.ErrorRequest = 100
	original.ValidRequest = 900
	original.ForwardTypes.Store("type1", 500)
	original.RequestOrigin.Store("us", 600)

	// Convert to export
	export := statistic.DailySummaryToExport(*original)

	// Convert back to summary
	summary := statistic.DailySummaryExportToSummary(export)

	// Verify values match
	assert.Equal(t, original.TotalRequest, summary.TotalRequest)
	assert.Equal(t, original.ErrorRequest, summary.ErrorRequest)
	assert.Equal(t, original.ValidRequest, summary.ValidRequest)

	val, ok := summary.ForwardTypes.Load("type1")
	assert.True(t, ok)
	assert.Equal(t, 500, val.(int))

	val, ok = summary.RequestOrigin.Load("us")
	assert.True(t, ok)
	assert.Equal(t, 600, val.(int))
}
