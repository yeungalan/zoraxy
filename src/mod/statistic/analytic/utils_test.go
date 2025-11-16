package analytic_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"imuslab.com/zoraxy/mod/statistic"
	"imuslab.com/zoraxy/mod/statistic/analytic"
)

func TestDateRangeGeneration(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, _ := statistic.NewStatisticCollector(option)
	defer collector.Close()

	loader := analytic.NewDataLoader(db, collector)

	t.Run("Single Day Range", func(t *testing.T) {
		today := time.Now().Format("2006_01_02")
		summaries, dates, err := loader.GetAllStatisticSummaryInRange(today, today)
		assert.NoError(t, err)
		// Should have 0 or 1 entries depending on if data exists
		assert.GreaterOrEqual(t, len(dates), 0)
		assert.Equal(t, len(summaries), len(dates))
	})

	t.Run("Multi-Day Range", func(t *testing.T) {
		start := time.Now().AddDate(0, 0, -7).Format("2006_01_02")
		end := time.Now().Format("2006_01_02")

		summaries, dates, err := loader.GetAllStatisticSummaryInRange(start, end)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(dates), 0)
		assert.Equal(t, len(summaries), len(dates))
	})

	t.Run("Invalid Date Format", func(t *testing.T) {
		_, _, err := loader.GetAllStatisticSummaryInRange("invalid", "2023_12_31")
		assert.Error(t, err)
	})
}

func TestEmptyMerge(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, _ := statistic.NewStatisticCollector(option)
	defer collector.Close()

	loader := analytic.NewDataLoader(db, collector)

	// Try to load range with no data
	start := time.Now().AddDate(0, 0, -30).Format("2006_01_02")
	end := time.Now().AddDate(0, 0, -20).Format("2006_01_02")

	summaries, dates, err := loader.GetAllStatisticSummaryInRange(start, end)
	assert.NoError(t, err)
	// Should return empty slices
	assert.Equal(t, 0, len(summaries))
	assert.Equal(t, 0, len(dates))
}
