//go:build manifest

package collectors

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

const TotalPayoutBurnQuery = `
		WITH payout_amount AS (
		  SELECT
			COALESCE(SUM((pp.element->'coin'->>'amount')::numeric), 0) AS amount
		  FROM api.messages_main m
		  CROSS JOIN LATERAL jsonb_array_elements(m.metadata->'payoutPairs') AS pp(element)
		  WHERE m.type = '/liftedinit.manifest.v1.MsgPayout'
		),
		burn_amount AS (
		  SELECT
			COALESCE(SUM((pp.element->>'amount')::numeric), 0) AS amount
		  FROM api.messages_main m
		  CROSS JOIN LATERAL jsonb_array_elements(m.metadata->'burnCoins') AS pp(element)
		  WHERE type = '/liftedinit.manifest.v1.MsgBurnHeldBalance'
		),
		burn_amount_contract AS (
		  WITH burn_events AS (
		    SELECT
		  	e.id,
		  	e.event_index,
		  	MAX(CASE WHEN e.attr_key = 'amount' THEN e.attr_value END) AS amount_raw,
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
		   AND m.type = '/cosmwasm.wasm.v1.MsgExecuteContract'
		  JOIN LATERAL regexp_matches(
		    b.amount_raw,
		    '^([0-9]+)([[:alnum:]_\/\.]+)$'
		  ) AS rm(captures) ON TRUE
		  WHERE (rm.captures)[2] = 'umfx'
	    )
		SELECT 
		  (SELECT amount FROM payout_amount) AS payout_amount,
		  (SELECT amount FROM burn_amount) + (SELECT amount FROM burn_amount_contract) AS burn_amount
	`

type TotalPayoutBurnCollector struct {
	db                *sql.DB
	totalPayoutAmount *prometheus.Desc
	totalBurnAmount   *prometheus.Desc
}

func NewTotalPayoutBurnCollector(db *sql.DB) *TotalPayoutBurnCollector {
	return &TotalPayoutBurnCollector{
		db: db,
		totalPayoutAmount: prometheus.NewDesc(
			prometheus.BuildFQName("yaci", "tokenomics", "total_payout_amount"),
			"Total umfx payout (mint)",
			nil,
			prometheus.Labels{"source": "postgres"},
		),
		totalBurnAmount: prometheus.NewDesc(
			prometheus.BuildFQName("yaci", "tokenomics", "total_burn_amount"),
			"Total umfx burn",
			nil,
			prometheus.Labels{"source": "postgres"},
		),
	}
}

func (c *TotalPayoutBurnCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalPayoutAmount
	ch <- c.totalBurnAmount
}

func (c *TotalPayoutBurnCollector) Collect(ch chan<- prometheus.Metric) {
	var payoutAmount, burnAmount float64
	err := c.db.QueryRow(TotalPayoutBurnQuery).Scan(&payoutAmount, &burnAmount)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(c.totalPayoutAmount, err)
		ch <- prometheus.NewInvalidMetric(c.totalBurnAmount, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(c.totalPayoutAmount, prometheus.CounterValue, payoutAmount)
	ch <- prometheus.MustNewConstMetric(c.totalBurnAmount, prometheus.CounterValue, burnAmount)
}

func init() {
	RegisterCollectorFactory(func(db *sql.DB, extraParams ...interface{}) (prometheus.Collector, error) {
		return NewTotalPayoutBurnCollector(db), nil
	})
}
