package statistic_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"imuslab.com/zoraxy/mod/statistic"
)

func TestHandleTodayStatLoad_Fast(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, err := statistic.NewStatisticCollector(option)
	assert.NoError(t, err)
	defer collector.Close()

	// Add some test data
	requestInfo := statistic.RequestInfo{
		IpAddr:                        "127.0.0.1",
		RequestOriginalCountryISOCode: "US",
		Succ:                          true,
		StatusCode:                    200,
		ForwardType:                   "host-http",
		Referer:                       "http://example.com",
		UserAgent:                     "Mozilla/5.0",
		RequestURL:                    "/test",
		Target:                        "target1",
	}
	collector.RecordRequest(requestInfo)

	// Create request with fast=true
	req := httptest.NewRequest(http.MethodGet, "/api/stats/today?fast=true", nil)
	w := httptest.NewRecorder()

	collector.HandleTodayStatLoad(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response statistic.DailySummaryExport
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	// In fast mode, only counters are returned, maps should be nil
	assert.Nil(t, response.ForwardTypes)
}

func TestHandleTodayStatLoad_Full(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, err := statistic.NewStatisticCollector(option)
	assert.NoError(t, err)
	defer collector.Close()

	// Add some test data
	requestInfo := statistic.RequestInfo{
		IpAddr:                        "192.168.1.1",
		RequestOriginalCountryISOCode: "US",
		Succ:                          true,
		StatusCode:                    200,
		ForwardType:                   "host-http",
		Referer:                       "http://example.com",
		UserAgent:                     "Mozilla/5.0",
		RequestURL:                    "/test",
		Target:                        "target1",
		Upstream:                      "upstream1",
	}
	collector.RecordRequest(requestInfo)

	// Wait for goroutine
	// time.Sleep(100 * time.Millisecond)

	// Create request with fast=false (default)
	req := httptest.NewRequest(http.MethodGet, "/api/stats/today?fast=false", nil)
	w := httptest.NewRecorder()

	collector.HandleTodayStatLoad(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var response statistic.DailySummaryExport
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	// In full mode, all data should be returned
	assert.NotNil(t, response.ForwardTypes)
}

func TestHandleTodayStatLoad_NoFastParam(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, err := statistic.NewStatisticCollector(option)
	assert.NoError(t, err)
	defer collector.Close()

	// Create request without fast parameter
	req := httptest.NewRequest(http.MethodGet, "/api/stats/today", nil)
	w := httptest.NewRecorder()

	collector.HandleTodayStatLoad(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleTodayStatLoad_POST(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, err := statistic.NewStatisticCollector(option)
	assert.NoError(t, err)
	defer collector.Close()

	// Create POST request with form data
	formData := url.Values{}
	formData.Set("fast", "true")
	req := httptest.NewRequest(http.MethodPost, "/api/stats/today", nil)
	req.Form = formData
	w := httptest.NewRecorder()

	collector.HandleTodayStatLoad(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
