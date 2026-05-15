package background

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/0x48core/go-sidecar-logging/sidecar-api/config"
	"github.com/0x48core/go-sidecar-logging/sidecar-api/elastic"
)

// LogWatcher polls the shared log file, deduplicates entries, and batches
// them to Elasticsearch when the batch size threshold is reached.
type LogWatcher struct {
	cfg    *config.Config
	es     elastic.Client
	seen   map[uint32]struct{} // FNV-32a hash of each raw line — prevents re-indexing
	batch  []elastic.LogEntry
	ticker *time.Ticker
	log    *zap.Logger
}

func NewLogWatcher(cfg *config.Config, es elastic.Client, log *zap.Logger) *LogWatcher {
	return &LogWatcher{
		cfg:    cfg,
		es:     es,
		seen:   make(map[uint32]struct{}),
		batch:  make([]elastic.LogEntry, 0, cfg.MaxBatchSize),
		ticker: time.NewTicker(cfg.WatchInterval),
		log:    log,
	}
}

// Run blocks until ctx is cancelled.
// On shutdown it performs a final flush to avoid losing buffered entries.
func (w *LogWatcher) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.ticker.Stop()
			w.flush(context.Background()) // flush with fresh ctx — original is cancelled
			return
		case <-w.ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *LogWatcher) tick(ctx context.Context) {
	path := filepath.Join(w.cfg.LogDir, w.cfg.LogFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			w.log.Error("read log file failed", zap.String("path", path), zap.Error(err))
		}
		return // file not yet created — silently wait
	}

	lines := strings.Split(string(data), "\n")
	for _, raw := range lines {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		h := hash(raw)
		if _, exists := w.seen[h]; exists {
			continue // already indexed
		}
		w.seen[h] = struct{}{}

		entry, err := w.parseLine(raw)
		if err != nil {
			w.log.Warn("skip malformed log line", zap.String("raw", raw), zap.Error(err))
			continue
		}

		w.batch = append(w.batch, entry)

		if len(w.batch) >= w.cfg.MaxBatchSize {
			w.flush(ctx)
		}
	}
}

func (w *LogWatcher) flush(ctx context.Context) {
	if len(w.batch) == 0 {
		return
	}

	index := fmt.Sprintf("%s-%s", w.cfg.ESIndex, time.Now().UTC().Format("2006.01.02"))

	if err := w.es.EnsureIndex(ctx, index); err != nil {
		w.log.Error("ensure index failed", zap.String("index", index), zap.Error(err))
		return
	}

	if err := w.es.BulkIndex(ctx, index, w.batch); err != nil {
		w.log.Error("bulk index failed", zap.Int("count", len(w.batch)), zap.Error(err))
		return
	}

	w.log.Info("flushed batch to elasticsearch",
		zap.String("index", index),
		zap.Int("count", len(w.batch)),
	)

	w.batch = w.batch[:0] // reset slice, keep allocated memory
}

// parseLine parses a pipe-delimited log line produced by transactions-api.
// Format: timestamp|level|transaction_id|amount|from|to
func (w *LogWatcher) parseLine(raw string) (elastic.LogEntry, error) {
	parts := strings.Split(raw, "|")
	if len(parts) != 6 {
		return elastic.LogEntry{}, fmt.Errorf("expected 6 fields, got %d", len(parts))
	}

	ts, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return elastic.LogEntry{}, fmt.Errorf("parse timestamp: %w", err)
	}

	amount, err := strconv.ParseFloat(parts[3], 64)
	if err != nil {
		return elastic.LogEntry{}, fmt.Errorf("parse amount: %w", err)
	}

	return elastic.LogEntry{
		Timestamp:     ts,
		Level:         parts[1],
		TransactionID: parts[2],
		Amount:        amount,
		From:          parts[4],
		To:            parts[5],
		Raw:           raw,
	}, nil
}

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}
