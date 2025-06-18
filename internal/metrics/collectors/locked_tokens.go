package collectors

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const AddressesQuery = `
 SELECT DISTINCT m.metadata->>'toAddress' as to_address
 FROM api.messages_main m
 JOIN api.transactions_main t ON m.id = t.id
 WHERE m.type = '/cosmos.vesting.v1beta1.MsgCreatePeriodicVestingAccount'
 AND t.error IS NULL
`

const AddressMessagesQuery = `
 SELECT 
   m.metadata->>'startTime' as start_time,
   m.metadata->'vestingPeriods' as vesting_periods
 FROM api.get_messages_for_address($1) m
 WHERE m.type = '/cosmos.vesting.v1beta1.MsgCreatePeriodicVestingAccount'
 AND m.error IS NULL
`

type VestingAmount struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

type VestingPeriod struct {
	Length string          `json:"length"`
	Amount []VestingAmount `json:"amount"`
}

type LockedTokensCollector struct {
	db                *sql.DB
	lockedTokensCount *prometheus.Desc
	denom             string
}

func NewLockedTokensCollector(db *sql.DB, denom string) *LockedTokensCollector {
	return &LockedTokensCollector{
		db:    db,
		denom: denom,
		lockedTokensCount: prometheus.NewDesc(
			prometheus.BuildFQName("yaci", "locked_tokens", "count"),
			"Total umfx locked in vesting accounts",
			[]string{"denom", "amount"},
			prometheus.Labels{"source": "postgres"},
		),
	}
}

func (c *LockedTokensCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.lockedTokensCount
}

func (c *LockedTokensCollector) Collect(ch chan<- prometheus.Metric) {
	// First, get all distinct toAddresses from vesting messages
	addressRows, err := c.db.Query(AddressesQuery)
	if err != nil {
		slog.Error("Failed to query addresses", "error", err)
		ch <- prometheus.NewInvalidMetric(c.lockedTokensCount, err)
		return
	}
	defer addressRows.Close()

	var totalLocked = big.NewInt(0)
	currentTime := time.Now().Unix()

	// For each address
	for addressRows.Next() {
		var toAddress string
		if err := addressRows.Scan(&toAddress); err != nil {
			slog.Error("Failed to scan address", "error", err)
			ch <- prometheus.NewInvalidMetric(c.lockedTokensCount, err)
			return
		}

		// Get all vesting messages for this address
		vestingRows, err := c.db.Query(AddressMessagesQuery, toAddress)
		if err != nil {
			slog.Error("Failed to query vesting messages", "address", toAddress, "error", err)
			ch <- prometheus.NewInvalidMetric(c.lockedTokensCount, err)
			return
		}
		defer vestingRows.Close()

		// Process each vesting message
		for vestingRows.Next() {
			var startTimeStr string
			var vestingPeriodsJSON []byte

			if err := vestingRows.Scan(&startTimeStr, &vestingPeriodsJSON); err != nil {
				slog.Error("Failed to scan vesting message", "error", err)
				ch <- prometheus.NewInvalidMetric(c.lockedTokensCount, err)
				return
			}

			startTime, err := strconv.ParseInt(startTimeStr, 10, 64)
			if err != nil {
				slog.Error("Failed to parse start time", "error", err)
				ch <- prometheus.NewInvalidMetric(c.lockedTokensCount, err)
				return
			}

			var vestingPeriods []VestingPeriod
			if err := json.Unmarshal(vestingPeriodsJSON, &vestingPeriods); err != nil {
				slog.Error("Failed to unmarshal vesting periods", "error", err)
				ch <- prometheus.NewInvalidMetric(c.lockedTokensCount, err)
				return
			}

			accountLocked, err := calculateLockedTokens(startTime, currentTime, vestingPeriods, c.denom)
			if err != nil {
				slog.Error("Failed to calculate locked tokens", "error", err)
				ch <- prometheus.NewInvalidMetric(c.lockedTokensCount, err)
				return
			}

			totalLocked = totalLocked.Add(totalLocked, accountLocked)
		}

		if err := vestingRows.Err(); err != nil {
			slog.Error("Failed to process vesting messages", "error", err)
			ch <- prometheus.NewInvalidMetric(c.lockedTokensCount, err)
			return
		}
	}

	if err := addressRows.Err(); err != nil {
		ch <- prometheus.NewInvalidMetric(c.lockedTokensCount, err)
		return
	}

	metric, err := prometheus.NewConstMetric(c.lockedTokensCount, prometheus.GaugeValue, 1, c.denom, totalLocked.String())
	if err != nil {
		slog.Error("Failed to create locked tokens metric", "denom", c.denom, "error", err)
		ch <- prometheus.NewInvalidMetric(c.lockedTokensCount, err)
		return
	}

	ch <- metric
}

// calculateLockedTokens calculates the amount of tokens still locked
func calculateLockedTokens(startTime, currentTime int64, periods []VestingPeriod, denom string) (*big.Int, error) {
	if len(periods) == 0 {
		return big.NewInt(0), nil
	}

	var lockedAmount = new(big.Int)
	unlockTime := startTime

	for _, period := range periods {
		lengthSeconds, err := strconv.ParseInt(period.Length, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse length: %s", period.Length)
		}

		unlockTime += lengthSeconds

		// Process all amounts in this period
		for _, amount := range period.Amount {
			if amount.Denom != denom {
				continue
			}

			var amountValue big.Int
			_, ok := amountValue.SetString(amount.Amount, 10)
			if !ok {
				return nil, fmt.Errorf("failed to parse amount: %s", amount.Amount)
			}

			// If unlock time is in the future, these tokens are still locked
			if unlockTime > currentTime {
				lockedAmount = lockedAmount.Add(lockedAmount, &amountValue)
			}
		}
	}

	return lockedAmount, nil
}

// See `locked_umfx.go` for an example of how to register this collector
