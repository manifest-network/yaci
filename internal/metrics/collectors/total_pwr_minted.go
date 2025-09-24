//go:build manifest

package collectors

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

// TotalPwrMintAmountQuery expected format of the cosmwasm event attribute for mint amounts, e.g., 123456upwr
// Note: only counts PWR minted via the tf_mint event (i.e., via the tf_mint function)
const TotalPwrMintAmountQuery = `
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
		   AND m.type = '/cosmwasm.wasm.v1.MsgExecuteContract'
		  JOIN LATERAL regexp_matches(
		    b.amount_raw,
		    '^([0-9]+)([[:alnum:]_\/\.]+)$'
		  ) AS rm(captures) ON TRUE
		  WHERE (rm.captures)[2] = 'factory/manifest1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsfmy9qj/upwr'
		)
		SELECT amount FROM pwr_minted;
`

type TotalPwrMintedCollector struct {
	totalPwrMintedAmount *prometheus.Desc
	db                   *sql.DB
}

func NewTotalPwrMintedCollector(db *sql.DB) *TotalPwrMintedCollector {
	return &TotalPwrMintedCollector{
		db: db,
		totalPwrMintedAmount: prometheus.NewDesc(
			prometheus.BuildFQName("yaci", "tokenomics", "total_pwr_minted_amount"),
			"Total PWR minted",
			nil,
			prometheus.Labels{"source": "postgres"},
		),
	}
}

func (c *TotalPwrMintedCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalPwrMintedAmount
}

func (c *TotalPwrMintedCollector) Collect(ch chan<- prometheus.Metric) {
	var mintAmount float64
	err := c.db.QueryRow(TotalPwrMintAmountQuery).Scan(&mintAmount)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.totalPwrMintedAmount, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.totalPwrMintedAmount, prometheus.CounterValue, mintAmount)
}

func init() {
	RegisterCollectorFactory(func(db *sql.DB, extraParams ...interface{}) (prometheus.Collector, error) {
		return NewTotalPwrMintedCollector(db), nil
	})
}
