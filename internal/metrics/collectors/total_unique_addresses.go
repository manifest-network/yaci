package collectors

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

const TotalUniqueAddressesQuery = `
		WITH all_addresses AS (
			SELECT sender AS address
			FROM api.messages_main
			WHERE sender LIKE $1
			
			UNION
			
			SELECT unnested_address AS address
			FROM api.messages_main
			CROSS JOIN LATERAL unnest(mentions) AS m(unnested_address)
			WHERE unnested_address LIKE $1
		),
		counts AS (
			SELECT 
				COUNT(DISTINCT CASE WHEN LENGTH(address) - $2 <= 38 THEN address END) AS user_count,
				COUNT(DISTINCT CASE WHEN LENGTH(address) - $2 > 38 THEN address END) AS group_count
			FROM all_addresses
		)
		SELECT user_count, group_count FROM counts
	`

// TotalUniqueAddressesCollector collects metrics for both user and group addresses in one query
type TotalUniqueAddressesCollector struct {
	db                        *sql.DB
	totalUniqueUserAddresses  *prometheus.Desc
	totalUniqueGroupAddresses *prometheus.Desc
	bech32Prefix              string
}

func NewTotalUniqueAddressesCollector(db *sql.DB, bech32Prefix string) *TotalUniqueAddressesCollector {
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
		bech32Prefix: bech32Prefix + "1",
	}
}

func (c *TotalUniqueAddressesCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalUniqueUserAddresses
	ch <- c.totalUniqueGroupAddresses
}

func (c *TotalUniqueAddressesCollector) Collect(ch chan<- prometheus.Metric) {
	var userCount, groupCount int64
	prefixLen := len(c.bech32Prefix)

	// Single query to get both counts
	err := c.db.QueryRow(TotalUniqueAddressesQuery, c.bech32Prefix+"%", prefixLen).Scan(&userCount, &groupCount)

	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.totalUniqueUserAddresses, err)
		ch <- prometheus.NewInvalidMetric(c.totalUniqueGroupAddresses, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.totalUniqueUserAddresses, prometheus.CounterValue, float64(userCount))
	ch <- prometheus.MustNewConstMetric(c.totalUniqueGroupAddresses, prometheus.CounterValue, float64(groupCount))
}
