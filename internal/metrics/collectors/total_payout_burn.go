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
		)
		SELECT 
		  (SELECT amount FROM payout_amount) AS payout_amount,
		  (SELECT amount FROM burn_amount) AS burn_amount
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
