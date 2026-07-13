package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (works with CGO disabled)
)

// Light history for the internal dashboard: a handful of series sampled every
// few minutes, auto-pruned to a short window, rendered as sparklines. All
// methods are nil-safe, so if the DB can't be opened the dashboard just loses
// the trend lines and keeps working.
const (
	metricsSampleEvery = 5 * time.Minute
	metricsPruneEvery  = 6 * time.Hour
	metricsRetention   = 7 * 24 * time.Hour
	metricsWindow      = 24 * time.Hour // how far back sparklines look
)

type metricsStore struct {
	db *sql.DB
}

func newMetricsStore(path string) (*metricsStore, error) {
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening metrics db: %w", err)
	}
	db.SetMaxOpenConns(1)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS metrics (
		ts     INTEGER NOT NULL,
		series TEXT NOT NULL,
		value  REAL NOT NULL
	)`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("creating metrics table: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_metrics_series_ts ON metrics(series, ts)`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("creating metrics index: %w", err)
	}
	return &metricsStore{db: db}, nil
}

func (s *metricsStore) record(ctx context.Context, ts int64, series string, value float64) {
	if s == nil {
		return
	}
	_, _ = s.db.ExecContext(ctx, `INSERT INTO metrics (ts, series, value) VALUES (?, ?, ?)`, ts, series, value)
}

// recent returns a series' values since the given unix time, oldest first.
func (s *metricsStore) recent(ctx context.Context, series string, since int64) []float64 {
	if s == nil {
		return nil
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT value FROM metrics WHERE series = ? AND ts >= ? ORDER BY ts`, series, since)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []float64
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err != nil {
			return out
		}
		out = append(out, v)
	}
	if rows.Err() != nil {
		return out
	}
	return out
}

func (s *metricsStore) purgeOlderThan(ctx context.Context, cutoff int64) {
	if s == nil {
		return
	}
	_, _ = s.db.ExecContext(ctx, `DELETE FROM metrics WHERE ts < ?`, cutoff)
}

// run samples the live stores on a ticker and prunes old rows, until ctx is
// cancelled. Safe to call on a nil store (returns immediately).
func (s *metricsStore) run(ctx context.Context, pihole *piholeClient, hosts *hostStore) {
	if s == nil {
		return
	}
	sample := func() {
		now := time.Now().Unix()
		if snap := pihole.snapshot(); snap.Up {
			s.record(ctx, now, "pihole.blocked_pct", snap.PercentBlocked)
		}
		for _, e := range hosts.all() {
			b := e.rep.Box
			s.record(ctx, now, b+".load", e.rep.Load1)
			if e.rep.TempC > 0 {
				s.record(ctx, now, b+".temp", e.rep.TempC)
			}
			maxDisk := 0
			for _, d := range e.rep.Disks {
				if d.UsedPct > maxDisk {
					maxDisk = d.UsedPct
				}
			}
			s.record(ctx, now, b+".disk", float64(maxDisk))
		}
	}
	prune := func() {
		s.purgeOlderThan(ctx, time.Now().Add(-metricsRetention).Unix())
	}

	sample()
	prune()
	sampleT := time.NewTicker(metricsSampleEvery)
	pruneT := time.NewTicker(metricsPruneEvery)
	defer sampleT.Stop()
	defer pruneT.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-sampleT.C:
			sample()
		case <-pruneT.C:
			prune()
		}
	}
}
