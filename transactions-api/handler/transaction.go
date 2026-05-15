package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/0x48core/go-sidecar-logging/transactions-api/queue"
)

type TransactionRequest struct {
	ID     string  `json:"id"     binding:"required"`
	From   string  `json:"from"   binding:"required"`
	To     string  `json:"to"     binding:"required"`
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

func NewTransactionHandler(q *queue.MessageQueue, log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req TransactionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		line := buildLogLine(req)

		if !q.Enqueue(line) {
			log.Warn("queue full — log message dropped",
				zap.String("transaction_id", req.ID),
			)
		}

		c.JSON(http.StatusAccepted, gin.H{"status": "accepted", "id": req.ID})
	}
}

// buildLogLine formats a pipe-delimited log entry.
// Format: timestamp|level|transaction_id|amount|from|to
func buildLogLine(req TransactionRequest) string {
	return fmt.Sprintf("%s|INFO|%s|%.2f|%s|%s",
		time.Now().UTC().Format(time.RFC3339),
		req.ID,
		req.Amount,
		req.From,
		req.To,
	)
}
