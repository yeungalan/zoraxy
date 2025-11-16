package analytic_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"imuslab.com/zoraxy/mod/database"
	"imuslab.com/zoraxy/mod/database/dbinc"
	"imuslab.com/zoraxy/mod/statistic"
	"imuslab.com/zoraxy/mod/statistic/analytic"
)

const test_db_path = "test_analytic_db"

func getNewDatabase() *database.Database {
	db, err := database.NewDatabase(test_db_path, dbinc.BackendLevelDB)
	if err != nil {
		panic(err)
	}
	db.NewTable("stats")
	return db
}

func clearDatabase(db *database.Database) {
	db.Close()
	os.RemoveAll(test_db_path)
}

func TestNewDataLoader(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, err := statistic.NewStatisticCollector(option)
	assert.NoError(t, err)
	defer collector.Close()

	loader := analytic.NewDataLoader(db, collector)
	assert.NotNil(t, loader)
	assert.Equal(t, db, loader.Database)
	assert.Equal(t, collector, loader.StatisticCollector)
}

func TestGetAllStatisticSummaryInRange(t *testing.T) {
	db := getNewDatabase()
	defer clearDatabase(db)

	option := statistic.CollectorOption{Database: db}
	collector, err := statistic.NewStatisticCollector(option)
	assert.NoError(t, err)
	defer collector.Close()

	loader := analytic.NewDataLoader(db, collector)

	t.Run("Single Day Range", func(t *testing.T) {
		today := time.Now().Format("2006_01_02")

		summaries, dates, err := loader.GetAllStatisticSummaryInRange(today, today)
		assert.NoError(t, err)
		assert.NotNil(t, summaries)
		assert.NotNil(t, dates)
	})

	t.Run("Invalid Start Date", func(t *testing.T) {
		summaries, dates, err := loader.GetAllStatisticSummaryInRange("invalid", "2023_12_31")
		assert.Error(t, err)
		assert.NotNil(t, summaries)
		assert.NotNil(t, dates)
	})

	t.Run("Invalid End Date", func(t *testing.T) {
		summaries, dates, err := loader.GetAllStatisticSummaryInRange("2023_01_01", "invalid")
		assert.Error(t, err)
		assert.NotNil(t, summaries)
		assert.NotNil(t, dates)
	})

	t.Run("End Before Start", func(t *testing.T) {
		start := "2023_12_31"
		end := "2023_01_01"

		summaries, dates, err := loader.GetAllStatisticSummaryInRange(start, end)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(summaries))
		assert.Equal(t, 0, len(dates))
	})

	t.Run("Week Range", func(t *testing.T) {
		weekAgo := time.Now().AddDate(0, 0, -7)
		today := time.Now()

		start := weekAgo.Format("2006_01_02")
		end := today.Format("2006_01_02")

		summaries, dates, err := loader.GetAllStatisticSummaryInRange(start, end)
		assert.NoError(t, err)
		assert.NotNil(t, summaries)
		assert.NotNil(t, dates)
	})

	t.Run("Month Range", func(t *testing.T) {
		monthAgo := time.Now().AddDate(0, -1, 0)
		today := time.Now()

		start := monthAgo.Format("2006_01_02")
		end := today.Format("2006_01_02")

		summaries, dates, err := loader.GetAllStatisticSummaryInRange(start, end)
		assert.NoError(t, err)
		assert.NotNil(t, summaries)
		assert.NotNil(t, dates)
	})
}
