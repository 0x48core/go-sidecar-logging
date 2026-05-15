package background

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/0x48core/go-sidecar-logging/transactions-api/logger"
	"github.com/0x48core/go-sidecar-logging/transactions-api/queue"
)

// LogWriter drains the message queue and persists entries to disk on each tick.
// Run it as a goroutine: go writer.Run(ctx)
type LogWriter struct {
	queue  *queue.MessageQueue
	logger logger.FileLogger
	ticker *time.Ticker
	log    *zap.Logger
}

func NewLogWriter(q *queue.MessageQueue, fl logger.FileLogger, interval time.Duration, log *zap.Logger) *LogWriter {
	return &LogWriter{
		queue:  q,
		logger: fl,
		ticker: time.NewTicker(interval),
		log:    log,
	}
}

// Run blocks until ctx is canceled, flushing the queue on every tick.
// On shutdown it performs one final flush to avoid dropping buffered messages.
func (w *LogWriter) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.ticker.Stop()
			w.flush() // drain remaining messages before exit
			return
		case <-w.ticker.C:
			w.flush()
		}
	}
}

func (w *LogWriter) flush() {
	messages := w.queue.DrainAll()
	for _, message := range messages {
		if err := w.logger.WriteLine(message); err != nil {
			w.log.Error("failed to write log line", zap.String("msg", message), zap.Error(err))
		}
	}
}
