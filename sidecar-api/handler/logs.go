package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/0x48core/go-sidecar-logging/sidecar-api/config"
	"github.com/0x48core/go-sidecar-logging/sidecar-api/elastic"
)

func NewLogsHandler(es elastic.Client, cfg *config.Config, log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		index := buildIndexName(cfg.ESIndex)

		entries, err := es.Search(c.Request.Context(), index)
		if err != nil {
			log.Error("search failed",
				zap.String("index", index),
				zap.Error(err),
			)
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "failed to retrieve logs from elasticsearch",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"index":   index,
			"count":   len(entries),
			"entries": entries,
		})
	}
}

// buildIndexName returns today's index name in the format: application-logs-2026.05.16
func buildIndexName(base string) string {
	return fmt.Sprintf("%s-%s", base, time.Now().UTC().Format("2006.01.02"))
}
