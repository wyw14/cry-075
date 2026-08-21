package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyw14/cry-075/internal/application"
	"github.com/wyw14/cry-075/internal/config"
	"github.com/wyw14/cry-075/internal/domain"
	"github.com/wyw14/cry-075/internal/platform"
	"github.com/wyw14/cry-075/internal/repository"
	"github.com/wyw14/cry-075/internal/service"
	"github.com/wyw14/cry-075/internal/transport/httpapi"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func() { _ = logger.Sync() }()
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("configuration rejected", zap.Error(err))
	}
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	store, err := repository.OpenPostgres(rootCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("database unavailable", zap.Error(err))
	}
	defer store.Close()
	clock := application.Clock(application.SystemClock)
	audit := application.AuditWriter{Repository: store, Clock: clock}
	campaigns := application.CampaignService{Campaigns: store, Catalog: store, Transactions: store, Audit: audit, Clock: clock}
	catalog := application.CatalogService{Catalog: store, Transactions: store, Audit: audit}
	review := application.ReviewService{Campaigns: store, Catalog: store, Releases: store, Transactions: store, Audit: audit, Clock: clock}
	releases := application.ReleaseService{Campaigns: store, Catalog: store, Releases: store, Transactions: store, Audit: audit, Clock: clock}
	scheduling := application.ScheduleService{Campaigns: store, Schedules: store, Transactions: store, Audit: audit, Clock: clock}
	preview := application.PreviewService{Campaigns: store, Catalog: store, Releases: store, Clock: clock}
	invalidation := application.InvalidationService{Campaigns: store, Catalog: store, Releases: store, AuditRepository: store, Transactions: store, Audit: audit, Clock: clock}
	operations := application.OperationsService{AuditRepository: store, Campaigns: store, Transactions: store, Audit: audit, Clock: clock}
	attachments := application.AttachmentService{Storage: platform.LocalFileStorage{Root: cfg.AttachmentDir}, MaxBytes: cfg.MaxUploadBytes, AllowedTypes: map[string]bool{"image/png": true, "image/jpeg": true, "video/mp4": true}, Audit: audit}
	readiness := platform.NewReadiness(store)
	api := httpapi.API{Campaigns: campaigns, Catalog: catalog, Review: review, Preview: preview, Invalidation: invalidation, Releases: releases, Scheduling: scheduling, Operations: operations, Attachments: attachments, Ready: func(c *gin.Context) error { return readiness.Check(c.Request.Context()) }}
	router := httpapi.NewRouter(api, cfg.AllowedOrigins, cfg.RequestTimeout, cfg.WebDistDir)
	scheduler := service.LocalScheduler{Service: scheduling, Actor: domain.Actor{ID: "system_scheduler", DisplayName: "本地调度器", Role: domain.RoleScheduler}, Interval: 5 * time.Second, BatchSize: 25, Logger: logger}
	go scheduler.Run(rootCtx)
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: router, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		logger.Info("http server started", zap.String("address", cfg.HTTPAddr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server stopped unexpectedly", zap.Error(err))
			stop()
		}
	}()
	<-rootCtx.Done()
	readiness.StopAccepting()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	}
}
