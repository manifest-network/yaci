//go:build manifest

package collectors

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

// TotalPwrMintBurnAmountQuery expected format of the event attribute for mint/burn amounts, e.g., 123456upwr
const TotalPwrMintBurnAmountQuery = `
		WITH pwr_minted AS (
		  WITH tf_mint_events AS (
		    SELECT
		  	e.id,
		  	e.event_index,
			MAX(e.attr_value) FILTER (WHERE e.attr_key = 'amount') AS amount_raw,
		  	MAX(e.msg_index) AS msg_index
		    FROM api.events_main e
		    WHERE e.event_type = 'tf_mint'
		    GROUP BY e.id, e.event_index
		  )
		  SELECT COALESCE(SUM((rm.captures)[1]::numeric), 0) AS amount
		  FROM tf_mint_events b
		  JOIN api.messages_main m
		    ON m.id = b.id
		   AND m.message_index = b.msg_index
		  JOIN LATERAL regexp_matches(
		    b.amount_raw,
		    '^([0-9]+)([[:alnum:]_\/\.]+)$'
		  ) AS rm(captures) ON TRUE
		  WHERE (rm.captures)[2] = 'factory/manifest1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsfmy9qj/upwr'
		),
		pwr_burned AS (
		  WITH burn_events AS (
			SELECT
			  e.id,
			  e.event_index,
			  MAX(e.attr_value) FILTER (WHERE e.attr_key = 'amount') AS amount_raw,
			  MAX(e.msg_index) AS msg_index
			FROM api.events_main e
			WHERE e.event_type = 'burn'
			GROUP BY e.id, e.event_index
		  )
		  SELECT COALESCE(SUM((rm.captures)[1]::numeric), 0) AS amount
		  FROM burn_events b
		  JOIN api.messages_main m
			ON m.id = b.id
		   AND m.message_index = b.msg_index
		  JOIN api.transactions_main tx
			ON tx.id = b.id
		   AND tx.error IS NULL
		  JOIN LATERAL regexp_matches(
			b.amount_raw,
			'^([0-9]+)([[:alnum:]_\/\.]+)$'
		  ) AS rm(captures) ON TRUE
		  WHERE (rm.captures)[2] = 'factory/manifest1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsfmy9qj/upwr'
		)
		SELECT
		  (SELECT amount FROM pwr_minted) as total_pwr_minted,
		  (SELECT amount FROM pwr_burned) as total_pwr_burned;
`

type TotalPwrMintedCollector struct {
	totalPwrMintedAmount *prometheus.Desc
	totalPwrBurnedAmount *prometheus.Desc
	db                   *sql.DB
}

func NewTotalPwrMintedCollector(db *sql.DB) *TotalPwrMintedCollector {
	return &TotalPwrMintedCollector{
		db: db,
		totalPwrMintedAmount: prometheus.NewDesc(
			prometheus.BuildFQName("yaci", "tokenomics", "total_pwr_minted_amount"),
			"Total upwr minted",
			nil,
			prometheus.Labels{"source": "postgres"},
		),
		totalPwrBurnedAmount: prometheus.NewDesc(
			prometheus.BuildFQName("yaci", "tokenomics", "total_pwr_burned_amount"),
			"Total upwr burned",
			nil,
			prometheus.Labels{"source": "postgres"},
		),
	}
}

func (c *TotalPwrMintedCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalPwrMintedAmount
	ch <- c.totalPwrBurnedAmount
}

func (c *TotalPwrMintedCollector) Collect(ch chan<- prometheus.Metric) {
	var mintAmount, burnAmount float64
	err := c.db.QueryRow(TotalPwrMintBurnAmountQuery).Scan(&mintAmount, &burnAmount)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.totalPwrMintedAmount, err)
		ch <- prometheus.NewInvalidMetric(c.totalPwrBurnedAmount, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.totalPwrMintedAmount, prometheus.CounterValue, mintAmount)
	ch <- prometheus.MustNewConstMetric(c.totalPwrBurnedAmount, prometheus.CounterValue, burnAmount)
}

func init() {
	RegisterCollectorFactory(func(db *sql.DB, extraParams ...interface{}) (prometheus.Collector, error) {
		return NewTotalPwrMintedCollector(db), nil
	})
}
