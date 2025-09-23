//go:build manifest

package collectors

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
)

// TotalPayoutBurnQuery expected format of the cosmwasm event attribute for burn amounts, e.g., 123456umfx
const TotalPayoutBurnQuery = `
	WITH mfx_burned AS (
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
	  JOIN api.transactions_main tx
		ON tx.id = b.id
	   AND tx.error IS NULL
	  JOIN LATERAL regexp_matches(
		b.amount_raw,
		'^([0-9]+)([[:alnum:]_\/\.]+)$'
	  ) AS rm(captures) ON TRUE
	  WHERE (rm.captures)[2] = 'umfx'
	),
	payout_amount AS (
	  SELECT
		COALESCE(SUM((pp.element->'coin'->>'amount')::numeric), 0) AS amount
	  FROM api.messages_main m
	  JOIN api.transactions_main t
		ON t.id = m.id
	   AND t.error IS NULL
	  CROSS JOIN LATERAL jsonb_array_elements(
		COALESCE(m.metadata->'payoutPairs', '[]'::jsonb)
	  ) AS pp(element)
	  WHERE m.type = '/liftedinit.manifest.v1.MsgPayout'
		AND (pp.element->'coin'->>'denom') = 'umfx'           -- keep only umfx
		AND (
		  m.message_index < 10000
		  OR (
			m.message_index >= 10000
			AND EXISTS (
			  SELECT 1
			  FROM api.transactions_main tx2
			  JOIN api.messages_main m2 ON tx2.id = m2.id
			  WHERE tx2.error IS NULL
				AND m2.type = '/cosmos.group.v1.MsgExec'
				AND tx2.proposal_ids IS NOT NULL
            	AND tx2.proposal_ids && t.proposal_ids
			)
		  )
		)
	)
	SELECT
	  (SELECT amount FROM payout_amount)  AS total_payout_umfx,
	  (SELECT amount FROM mfx_burned)     AS total_burned_umfx;
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
