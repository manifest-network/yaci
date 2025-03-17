package collectors

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

type TotalTransactionCountCollector struct {
	db           *sql.DB
	totalTxCount *prometheus.Desc
}

func NewTotalTransactionCountCollector(db *sql.DB) *TotalTransactionCountCollector {
	return &TotalTransactionCountCollector{
		db: db,
		totalTxCount: prometheus.NewDesc(
			prometheus.BuildFQName("yaci", "transactions", "total_count"),
			"Total transaction count",
			nil,
			prometheus.Labels{"source": "postgres"},
		),
	}
}

func (c *TotalTransactionCountCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalTxCount
}

func (c *TotalTransactionCountCollector) Collect(ch chan<- prometheus.Metric) {
	var count int64
	err := c.db.QueryRow("SELECT COUNT(*) FROM api.transactions_main").Scan(&count)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.totalTxCount, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.totalTxCount, prometheus.CounterValue, float64(count))
}
