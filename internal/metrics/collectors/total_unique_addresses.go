package collectors

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

type TotalUniqueAddressesCollector struct {
	db                   *sql.DB
	totalUniqueAddresses *prometheus.Desc
}

func NewTotalUniqueAddressesCollector(db *sql.DB) *TotalUniqueAddressesCollector {
	return &TotalUniqueAddressesCollector{
		db: db,
		totalUniqueAddresses: prometheus.NewDesc(
			prometheus.BuildFQName("yaci", "addresses", "total_unique"),
			"Total unique addresses",
			nil,
			prometheus.Labels{"source": "postgres"},
		),
	}
}

func (c *TotalUniqueAddressesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalUniqueAddresses
}

func (c *TotalUniqueAddressesCollector) Collect(ch chan<- prometheus.Metric) {
	var count int64
	err := c.db.QueryRow(`SELECT COUNT(DISTINCT address) FROM (
			SELECT sender AS address
			FROM api.messages_main
			WHERE sender LIKE 'manifest1%'
			
			UNION
			
			SELECT unnested_address AS address
			FROM api.messages_main
			CROSS JOIN LATERAL unnest(mentions) AS m(unnested_address)
			WHERE unnested_address LIKE 'manifest1%'
		);`).Scan(&count)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.totalUniqueAddresses, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.totalUniqueAddresses, prometheus.CounterValue, float64(count))
}
