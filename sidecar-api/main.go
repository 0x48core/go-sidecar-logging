package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/0x48core/go-sidecar-logging/sidecar-api/background"
	"github.com/0x48core/go-sidecar-logging/sidecar-api/config"
	"github.com/0x48core/go-sidecar-logging/sidecar-api/elastic"
	"github.com/0x48core/go-sidecar-logging/sidecar-api/handler"
)

func main() {
	cfg := config.Load()

	log, _ := zap.NewProduction()
	defer log.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// connect to Elasticsearch
	es, err := elastic.NewClient(cfg.ESUrl)
	if err != nil {
		log.Fatal("failed to create elasticsearch client", zap.Error(err))
	}

	// wire components
	watcher := background.NewLogWatcher(cfg, es, log)

	// start background watch goroutine
	go watcher.Run(ctx)

	// HTTP router
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/logs", handler.NewLogsHandler(es, cfg, log))

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Info("sidecar-api starting", zap.String("port", cfg.Port))
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
