package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/0x48core/go-sidecar-logging/transactions-api/background"
	"github.com/0x48core/go-sidecar-logging/transactions-api/config"
	"github.com/0x48core/go-sidecar-logging/transactions-api/handler"
	"github.com/0x48core/go-sidecar-logging/transactions-api/logger"
	"github.com/0x48core/go-sidecar-logging/transactions-api/queue"
)

func main() {
	cfg := config.Load()

	log, _ := zap.NewProduction()
	defer log.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// wire components
	q := queue.NewMessageQueue(cfg.QueueSize)
	fl := logger.NewFileLogger(filepath.Join(cfg.LogDir, cfg.LogFile))
	w := background.NewLogWriter(q, fl, cfg.FlushInterval, log)

	// start background flush goroutine
	go w.Run(ctx)

	// HTTP Router
	r := gin.New()
	r.Use(gin.Recovery())
	r.POST("/transactions", handler.NewTransactionHandler(q, log))

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Info("transactions-api starting", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	// block until SIGINT / SIGTERM
	<-ctx.Done()
	log.Info("shutting down...")

	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
	}

	log.Info("shutdown complete")
}
