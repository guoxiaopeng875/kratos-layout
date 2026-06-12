package metrics

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

// DBCollector implements prometheus.Collector to expose sql.DB connection pool stats.
// It reads stats at scrape time via sql.DB.Stats(), avoiding extra goroutines.
type DBCollector struct {
	db *sql.DB

	openConns  *prometheus.Desc
	idleConns  *prometheus.Desc
	inUseConns *prometheus.Desc
	waitCount  *prometheus.Desc
	waitDur    *prometheus.Desc
}

// NewDBCollector creates a new DBCollector for the given *sql.DB.
func NewDBCollector(db *sql.DB) *DBCollector {
	return &DBCollector{
		db: db,
		openConns: prometheus.NewDesc(
			"app_db_open_connections",
			"Number of open database connections.",
			nil, nil,
		),
		idleConns: prometheus.NewDesc(
			"app_db_idle_connections",
			"Number of idle database connections.",
			nil, nil,
		),
		inUseConns: prometheus.NewDesc(
			"app_db_in_use_connections",
			"Number of in-use database connections.",
			nil, nil,
		),
		waitCount: prometheus.NewDesc(
			"app_db_wait_count_total",
			"Total number of connections waited for.",
			nil, nil,
		),
		waitDur: prometheus.NewDesc(
			"app_db_wait_duration_seconds_total",
			"Total time blocked waiting for a new connection.",
			nil, nil,
		),
	}
}

// Describe implements prometheus.Collector.
func (c *DBCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.openConns
	ch <- c.idleConns
	ch <- c.inUseConns
	ch <- c.waitCount
	ch <- c.waitDur
}

// Collect implements prometheus.Collector.
func (c *DBCollector) Collect(ch chan<- prometheus.Metric) {
	stats := c.db.Stats()

	ch <- prometheus.MustNewConstMetric(c.openConns, prometheus.GaugeValue, float64(stats.OpenConnections))
	ch <- prometheus.MustNewConstMetric(c.idleConns, prometheus.GaugeValue, float64(stats.Idle))
	ch <- prometheus.MustNewConstMetric(c.inUseConns, prometheus.GaugeValue, float64(stats.InUse))
	ch <- prometheus.MustNewConstMetric(c.waitCount, prometheus.CounterValue, float64(stats.WaitCount))
	ch <- prometheus.MustNewConstMetric(c.waitDur, prometheus.CounterValue, stats.WaitDuration.Seconds())
}
