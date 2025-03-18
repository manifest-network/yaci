package collectors

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

// TotalUniqueAddressesCollector collects metrics for both user and group addresses in one query
type TotalUniqueAddressesCollector struct {
	db                        *sql.DB
	totalUniqueUserAddresses  *prometheus.Desc
	totalUniqueGroupAddresses *prometheus.Desc
}

func NewTotalUniqueAddressesCollector(db *sql.DB) *TotalUniqueAddressesCollector {
	return &TotalUniqueAddressesCollector{
		db: db,
		totalUniqueUserAddresses: prometheus.NewDesc(
			prometheus.BuildFQName("yaci", "addresses", "total_unique_user"),
			"Total unique user addresses",
			nil,
			prometheus.Labels{"source": "postgres"},
		),
		totalUniqueGroupAddresses: prometheus.NewDesc(
			prometheus.BuildFQName("yaci", "addresses", "total_unique_group"),
			"Total unique group addresses",
			nil,
			prometheus.Labels{"source": "postgres"},
		),
	}
}

func (c *TotalUniqueAddressesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalUniqueUserAddresses
	ch <- c.totalUniqueGroupAddresses
}

func (c *TotalUniqueAddressesCollector) Collect(ch chan<- prometheus.Metric) {
	var userCount, groupCount int64

	// Single query to get both counts
	err := c.db.QueryRow(`
		WITH all_addresses AS (
			SELECT sender AS address
			FROM api.messages_main
			WHERE sender LIKE 'manifest1%'
			
			UNION
			
			SELECT unnested_address AS address
			FROM api.messages_main
			CROSS JOIN LATERAL unnest(mentions) AS m(unnested_address)
			WHERE unnested_address LIKE 'manifest1%'
		),
		counts AS (
			SELECT 
				COUNT(DISTINCT CASE WHEN LENGTH(address) <= 47 THEN address END) AS user_count,
				COUNT(DISTINCT CASE WHEN LENGTH(address) > 47 THEN address END) AS group_count
			FROM all_addresses
		)
		SELECT user_count, group_count FROM counts
	`).Scan(&userCount, &groupCount)

	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.totalUniqueUserAddresses, err)
		ch <- prometheus.NewInvalidMetric(c.totalUniqueGroupAddresses, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.totalUniqueUserAddresses, prometheus.CounterValue, float64(userCount))
	ch <- prometheus.MustNewConstMetric(c.totalUniqueGroupAddresses, prometheus.CounterValue, float64(groupCount))
}
