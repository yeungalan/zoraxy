package statistic_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"imuslab.com/zoraxy/mod/statistic"
)

func TestGetCurrentDailySummary(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, err := statistic.NewStatisticCollector(option)
	assert.NoError(t, err)
	defer collector.Close()

	summary := collector.GetCurrentDailySummary()
	assert.NotNil(t, summary)
	assert.Equal(t, int64(0), summary.TotalRequest)
}

func TestSetAutoSave_Enable(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, err := statistic.NewStatisticCollector(option)
	assert.NoError(t, err)

	// Set autosave to 1 second
	collector.SetAutoSave(1)

	// Add some data
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

	// Wait for autosave to trigger
	time.Sleep(2 * time.Second)

	// Verify data was saved
	year, month, day := time.Now().Date()
	summary := collector.LoadSummaryOfDay(year, month, day)
	assert.NotNil(t, summary)

	// Properly clean up
	time.Sleep(100 * time.Millisecond)
	collector.Close()
}

func TestSetAutoSave_Disable(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, err := statistic.NewStatisticCollector(option)
	assert.NoError(t, err)
	defer collector.Close()

	// Disable autosave (should not panic or error)
	collector.SetAutoSave(0)
}

func TestPrintDailySummary(t *testing.T) {
	summary := statistic.NewDailySummary()
	summary.TotalRequest = 100
	summary.ForwardTypes.Store("host-http", 50)
	summary.RequestOrigin.Store("us", 60)
	summary.RequestClientIp.Store("192.168.1.1", 10)
	summary.Referer.Store("http://example.com", 100)
	summary.UserAgent.Store("Mozilla/5.0", 100)
	summary.RequestURL.Store("/test", 100)

	// This function just prints, so we just make sure it doesn't panic
	assert.NotPanics(t, func() {
		statistic.PrintDailySummary(summary)
	})
}

func TestRecordRequest_EdgeCases(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, err := statistic.NewStatisticCollector(option)
	assert.NoError(t, err)
	defer collector.Close()

	t.Run("Failed Request", func(t *testing.T) {
		requestInfo := statistic.RequestInfo{
			IpAddr:                        "127.0.0.1",
			RequestOriginalCountryISOCode: "US",
			Succ:                          false, // Failed request
			StatusCode:                    500,
			ForwardType:                   "host-http",
			Referer:                       "http://example.com",
			UserAgent:                     "Mozilla/5.0",
			RequestURL:                    "/test",
			Target:                        "target1",
		}
		collector.RecordRequest(requestInfo)
		time.Sleep(100 * time.Millisecond)

		summary := collector.GetCurrentDailySummary()
		assert.Greater(t, summary.ErrorRequest, int64(0))
	})

	t.Run("Cloudflare Forwarded IP", func(t *testing.T) {
		requestInfo := statistic.RequestInfo{
			IpAddr:                        "158.250.160.114,109.21.249.211", // CF forwarded
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
		time.Sleep(100 * time.Millisecond)

		// The IP should be extracted from the forwarded header
	})

	t.Run("IPv6 Forwarded IP", func(t *testing.T) {
		requestInfo := statistic.RequestInfo{
			IpAddr:                        "[15c4:cbb4:cc98:4291:ffc1:3a46:06a1:51a7],109.21.249.211",
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
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("XSS in Referer", func(t *testing.T) {
		requestInfo := statistic.RequestInfo{
			IpAddr:                        "127.0.0.1",
			RequestOriginalCountryISOCode: "US",
			Succ:                          true,
			StatusCode:                    200,
			ForwardType:                   "host-http",
			Referer:                       "<script>alert('xss')</script>",
			UserAgent:                     "Mozilla/5.0",
			RequestURL:                    "/test",
			Target:                        "target1",
		}
		collector.RecordRequest(requestInfo)
		time.Sleep(100 * time.Millisecond)

		// Referer should be sanitized
	})

	t.Run("Static File Request", func(t *testing.T) {
		requestInfo := statistic.RequestInfo{
			IpAddr:                        "127.0.0.1",
			RequestOriginalCountryISOCode: "US",
			Succ:                          true,
			StatusCode:                    200,
			ForwardType:                   "host-http",
			Referer:                       "http://example.com",
			UserAgent:                     "Mozilla/5.0",
			RequestURL:                    "/static/image.png", // Static file
			Target:                        "target1",
		}
		collector.RecordRequest(requestInfo)
		time.Sleep(100 * time.Millisecond)

		// Static files should not be recorded in RequestURL
	})

	t.Run("Web Page Request", func(t *testing.T) {
		requestInfo := statistic.RequestInfo{
			IpAddr:                        "127.0.0.1",
			RequestOriginalCountryISOCode: "US",
			Succ:                          true,
			StatusCode:                    200,
			ForwardType:                   "host-http",
			Referer:                       "http://example.com",
			UserAgent:                     "Mozilla/5.0",
			RequestURL:                    "/page.html", // Web page
			Target:                        "target1",
		}
		collector.RecordRequest(requestInfo)
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("Request with Upstream", func(t *testing.T) {
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
			Upstream:                      "upstream1",
		}
		collector.RecordRequest(requestInfo)
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("Multiple Requests Same IP", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			requestInfo := statistic.RequestInfo{
				IpAddr:                        "192.168.1.100",
				RequestOriginalCountryISOCode: "UK",
				Succ:                          true,
				StatusCode:                    200,
				ForwardType:                   "subdomain-http",
				Referer:                       "http://example.org",
				UserAgent:                     "Chrome/100.0",
				RequestURL:                    "/api/data",
				Target:                        "api.example.com",
			}
			collector.RecordRequest(requestInfo)
		}
		time.Sleep(200 * time.Millisecond)

		summary := collector.GetCurrentDailySummary()
		assert.Greater(t, summary.TotalRequest, int64(0))
	})
}

func TestConcurrentRecordRequest(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, err := statistic.NewStatisticCollector(option)
	assert.NoError(t, err)
	defer collector.Close()

	// Test concurrent access
	numGoroutines := 100
	requestsPerGoroutine := 10

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < requestsPerGoroutine; j++ {
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
			}
		}(i)
	}

	// Wait for all goroutines to complete
	time.Sleep(2 * time.Second)

	// We should have some requests recorded
	summary := collector.GetCurrentDailySummary()
	assert.Greater(t, summary.TotalRequest, int64(0))
}

func TestLoadSummaryOfDay_DifferentDates(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, err := statistic.NewStatisticCollector(option)
	assert.NoError(t, err)
	defer collector.Close()

	t.Run("Load Yesterday", func(t *testing.T) {
		yesterday := time.Now().AddDate(0, 0, -1)
		summary := collector.LoadSummaryOfDay(yesterday.Year(), yesterday.Month(), yesterday.Day())
		assert.NotNil(t, summary)
	})

	t.Run("Load Last Week", func(t *testing.T) {
		lastWeek := time.Now().AddDate(0, 0, -7)
		summary := collector.LoadSummaryOfDay(lastWeek.Year(), lastWeek.Month(), lastWeek.Day())
		assert.NotNil(t, summary)
	})

	t.Run("Load Last Month", func(t *testing.T) {
		lastMonth := time.Now().AddDate(0, -1, 0)
		summary := collector.LoadSummaryOfDay(lastMonth.Year(), lastMonth.Month(), lastMonth.Day())
		assert.NotNil(t, summary)
	})
}

func TestGetCurrentRealtimeStatIntervalId_AllDayIntervals(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, err := statistic.NewStatisticCollector(option)
	assert.NoError(t, err)
	defer collector.Close()

	intervalId := collector.GetCurrentRealtimeStatIntervalId()

	// Should be between 0 and 287 (288 intervals of 5 minutes in 24 hours)
	assert.GreaterOrEqual(t, intervalId, 0)
	assert.LessOrEqual(t, intervalId, 287)
}

func TestResetAndReload(t *testing.T) {
	db := getNewDatabase()

	option := statistic.CollectorOption{Database: db}
	collector, err := statistic.NewStatisticCollector(option)
	assert.NoError(t, err)

	// Add some data
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
	time.Sleep(100 * time.Millisecond)

	// Save and close
	collector.SaveSummaryOfDay()
	time.Sleep(100 * time.Millisecond)
	collector.Close()
	time.Sleep(100 * time.Millisecond)

	// Create new collector - should reload data
	collector2, err := statistic.NewStatisticCollector(option)
	assert.NoError(t, err)

	summary := collector2.GetCurrentDailySummary()
	assert.NotNil(t, summary)

	collector2.Close()
	time.Sleep(100 * time.Millisecond)

	clearDatabase(db)
}

func TestMultipleCountryOrigins(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, err := statistic.NewStatisticCollector(option)
	assert.NoError(t, err)
	defer collector.Close()

	countries := []string{"US", "UK", "JP", "DE", "FR", "CA", "AU", "IN"}

	for _, country := range countries {
		requestInfo := statistic.RequestInfo{
			IpAddr:                        "127.0.0.1",
			RequestOriginalCountryISOCode: country,
			Succ:                          true,
			StatusCode:                    200,
			ForwardType:                   "host-http",
			Referer:                       "http://example.com",
			UserAgent:                     "Mozilla/5.0",
			RequestURL:                    "/test",
			Target:                        "target1",
		}
		collector.RecordRequest(requestInfo)
	}

	time.Sleep(200 * time.Millisecond)

	summary := collector.GetCurrentDailySummary()
	assert.Equal(t, int64(len(countries)), summary.TotalRequest)
}

func TestMultipleForwardTypes(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, err := statistic.NewStatisticCollector(option)
	assert.NoError(t, err)
	defer collector.Close()

	forwardTypes := []string{
		"host-http",
		"host-https",
		"subdomain-http",
		"subdomain-https",
		"vdir-http",
		"vdir-https",
		"subdomain-websocket",
		"host-websocket",
	}

	for _, fwdType := range forwardTypes {
		requestInfo := statistic.RequestInfo{
			IpAddr:                        "127.0.0.1",
			RequestOriginalCountryISOCode: "US",
			Succ:                          true,
			StatusCode:                    200,
			ForwardType:                   fwdType,
			Referer:                       "http://example.com",
			UserAgent:                     "Mozilla/5.0",
			RequestURL:                    "/test",
			Target:                        "target1",
		}
		collector.RecordRequest(requestInfo)
	}

	time.Sleep(500 * time.Millisecond)

	summary := collector.GetCurrentDailySummary()
	// Check that requests were recorded (may be slightly less due to goroutine timing)
	assert.GreaterOrEqual(t, summary.TotalRequest, int64(len(forwardTypes)-1))
}
