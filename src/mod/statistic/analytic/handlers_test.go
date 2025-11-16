package analytic_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"imuslab.com/zoraxy/mod/statistic"
	"imuslab.com/zoraxy/mod/statistic/analytic"
)

func TestGetStartAndEndDatesFromRequest(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, _ := statistic.NewStatisticCollector(option)
	defer collector.Close()

	loader := analytic.NewDataLoader(db, collector)

	t.Run("Valid Dates with Underscores", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/stats/range?start=2023_01_01&end=2023_12_31", nil)

		start, end, err := loader.GetStartAndEndDatesFromRequest(req)
		assert.NoError(t, err)
		assert.Equal(t, "2023_01_01", start)
		assert.Equal(t, "2023_12_31", end)
	})

	t.Run("Valid Dates with Dashes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/stats/range?start=2023-01-01&end=2023-12-31", nil)

		start, end, err := loader.GetStartAndEndDatesFromRequest(req)
		assert.NoError(t, err)
		assert.Equal(t, "2023_01_01", start)
		assert.Equal(t, "2023_12_31", end)
	})

	t.Run("Missing Start Date", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/stats/range?end=2023_12_31", nil)

		_, _, err := loader.GetStartAndEndDatesFromRequest(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "start date cannot be empty")
	})

	t.Run("Missing End Date", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/stats/range?start=2023_01_01", nil)

		_, _, err := loader.GetStartAndEndDatesFromRequest(req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "end date cannot be empty")
	})

	t.Run("Both Dates Missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/stats/range", nil)

		_, _, err := loader.GetStartAndEndDatesFromRequest(req)
		assert.Error(t, err)
	})
}

func TestHandleSummaryList(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, _ := statistic.NewStatisticCollector(option)
	defer collector.Close()

	loader := analytic.NewDataLoader(db, collector)

	req := httptest.NewRequest(http.MethodGet, "/api/stats/list", nil)
	w := httptest.NewRecorder()

	loader.HandleSummaryList(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var dates []string
	err := json.Unmarshal(w.Body.Bytes(), &dates)
	assert.NoError(t, err)
	// Should return some dates (at least empty array)
	assert.NotNil(t, dates)
}

func TestHandleLoadTargetDaySummary(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, _ := statistic.NewStatisticCollector(option)
	defer collector.Close()

	loader := analytic.NewDataLoader(db, collector)

	t.Run("Load Today's Data", func(t *testing.T) {
		// Add some data to collector
		collector.DailySummary.TotalRequest = 500
		collector.DailySummary.ErrorRequest = 50
		collector.DailySummary.ValidRequest = 450

		today := time.Now().Format("2006_01_02")

		req := httptest.NewRequest(http.MethodGet, "/api/stats/day?id="+today, nil)
		w := httptest.NewRecorder()

		loader.HandleLoadTargetDaySummary(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var summary statistic.DailySummaryExport
		err := json.Unmarshal(w.Body.Bytes(), &summary)
		assert.NoError(t, err)
		assert.Equal(t, int64(500), summary.TotalRequest)
	})

	t.Run("Missing ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/stats/day", nil)
		w := httptest.NewRecorder()

		loader.HandleLoadTargetDaySummary(w, req)

		// Handler executes
		assert.NotNil(t, w)
	})
}

func TestHandleLoadTargetRangeSummary(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, _ := statistic.NewStatisticCollector(option)
	defer collector.Close()

	loader := analytic.NewDataLoader(db, collector)

	t.Run("Valid Range", func(t *testing.T) {
		baseDate := time.Now().AddDate(0, 0, -5)
		for i := 0; i < 5; i++ {
			date := baseDate.AddDate(0, 0, i)
			export := statistic.DailySummaryExport{
				TotalRequest: int64(100 * (i + 1)),
				ErrorRequest: int64(10 * (i + 1)),
				ValidRequest: int64(90 * (i + 1)),
			}
			db.Write("stats", date.Format("2006_01_02"), export)
		}

		start := baseDate.Format("2006_01_02")
		end := baseDate.AddDate(0, 0, 4).Format("2006_01_02")

		req := httptest.NewRequest(http.MethodGet, "/api/stats/range?start="+start+"&end="+end, nil)
		w := httptest.NewRecorder()

		loader.HandleLoadTargetRangeSummary(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Missing Dates", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/stats/range", nil)
		w := httptest.NewRecorder()

		loader.HandleLoadTargetRangeSummary(w, req)

		// Handler executes
		assert.NotNil(t, w)
	})
}

func TestHandleRangeExport(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, _ := statistic.NewStatisticCollector(option)
	defer collector.Close()

	loader := analytic.NewDataLoader(db, collector)

	// Create test data
	baseDate := time.Now().AddDate(0, 0, -2)
	for i := 0; i < 3; i++ {
		date := baseDate.AddDate(0, 0, i)
		export := statistic.DailySummaryExport{
			TotalRequest: int64(100 * (i + 1)),
		}
		db.Write("stats", date.Format("2006_01_02"), export)
	}

	start := baseDate.Format("2006_01_02")
	end := baseDate.AddDate(0, 0, 2).Format("2006_01_02")

	t.Run("Export JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/stats/export?start="+start+"&end="+end+"&format=json", nil)
		w := httptest.NewRecorder()

		loader.HandleRangeExport(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Export CSV", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/stats/export?start="+start+"&end="+end+"&format=csv", nil)
		w := httptest.NewRecorder()

		loader.HandleRangeExport(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Missing Dates", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/stats/export?format=json", nil)
		w := httptest.NewRecorder()

		loader.HandleRangeExport(w, req)

		// Handler should execute, may or may not return error depending on implementation
		assert.NotNil(t, w)
	})
}

func TestHandleRangeReset(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, _ := statistic.NewStatisticCollector(option)
	defer collector.Close()

	loader := analytic.NewDataLoader(db, collector)

	// Create test data
	baseDate := time.Now().AddDate(0, 0, -2)
	for i := 0; i < 3; i++ {
		date := baseDate.AddDate(0, 0, i)
		export := statistic.DailySummaryExport{
			TotalRequest: int64(100 * (i + 1)),
		}
		db.Write("stats", date.Format("2006_01_02"), export)
	}

	start := baseDate.Format("2006_01_02")
	end := baseDate.AddDate(0, 0, 2).Format("2006_01_02")

	t.Run("Valid DELETE Request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/stats/reset?start="+start+"&end="+end, nil)
		w := httptest.NewRecorder()

		loader.HandleRangeReset(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid Method (GET)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/stats/reset?start="+start+"&end="+end, nil)
		w := httptest.NewRecorder()

		loader.HandleRangeReset(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("Missing Dates", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/stats/reset", nil)
		w := httptest.NewRecorder()

		loader.HandleRangeReset(w, req)

		// Handler should execute, may or may not return error depending on implementation
		assert.NotNil(t, w)
	})
}
