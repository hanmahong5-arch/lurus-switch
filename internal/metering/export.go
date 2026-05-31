package metering

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"lurus-switch/internal/pricing"
)

// ExportRange returns every record whose timestamp falls within [from, to],
// each carrying its stable Record.ID. It's the raw feed a reseller exports to
// cross-check Switch's local metering against their Hub console. Pure read over
// the in-memory + on-disk daily ledger.
func (s *Store) ExportRange(from, to time.Time) []Record {
	return s.recordsInRange(from, to)
}

// exportColumns is the stable CSV header, keyed by Record.ID first so a
// reseller can join rows against Hub logs. Cost is computed at export time
// from the pricing table (same source the dashboards use).
var exportColumns = []string{
	"id", "timestamp", "appId", "model",
	"tokensIn", "tokensOut", "cacheCreate", "cacheRead",
	"costUSD", "statusCode", "cached", "servedBy", "matchedBy",
}

// MarshalRecordsCSV renders records as CSV with exportColumns. Deterministic:
// rows preserve the input order, costs use pricing.Cost. The header is always
// emitted, so an empty slice yields a header-only document.
func MarshalRecordsCSV(records []Record) (string, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(exportColumns); err != nil {
		return "", fmt.Errorf("write csv header: %w", err)
	}
	for _, r := range records {
		cost := pricing.Cost(r.Model, r.TokensIn, r.TokensOut, r.CacheCreateTokens, r.CacheReadTokens)
		row := []string{
			r.ID,
			r.Timestamp.UTC().Format(time.RFC3339),
			r.AppID,
			r.Model,
			strconv.FormatInt(r.TokensIn, 10),
			strconv.FormatInt(r.TokensOut, 10),
			strconv.FormatInt(r.CacheCreateTokens, 10),
			strconv.FormatInt(r.CacheReadTokens, 10),
			strconv.FormatFloat(cost, 'f', 6, 64),
			strconv.Itoa(r.StatusCode),
			strconv.FormatBool(r.CachedHit),
			r.ServedBy,
			r.MatchedBy,
		}
		if err := w.Write(row); err != nil {
			return "", fmt.Errorf("write csv row %s: %w", r.ID, err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", fmt.Errorf("flush csv: %w", err)
	}
	return buf.String(), nil
}

// MarshalRecordsJSON renders records as an indented JSON array. nil/empty
// slices serialize to "[]" so the consumer never has to special-case null.
func MarshalRecordsJSON(records []Record) ([]byte, error) {
	if records == nil {
		records = []Record{}
	}
	return json.MarshalIndent(records, "", "  ")
}
