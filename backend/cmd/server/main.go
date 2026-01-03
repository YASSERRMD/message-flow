package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"message-flow/backend/internal/auth"
	"message-flow/backend/internal/config"
	"message-flow/backend/internal/db"
	"message-flow/backend/internal/handlers"
	"message-flow/backend/internal/llm"
	"message-flow/backend/internal/middleware"
	"message-flow/backend/internal/realtime"
	"message-flow/backend/internal/router"
	"message-flow/backend/internal/whatsapp"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	store, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer store.Close()

	authService, err := auth.NewService(cfg.JWTSecret, 24*time.Hour)
	if err != nil {
		log.Fatalf("failed to init auth: %v", err)
	}
	hub := realtime.NewHub()
	var llmQueue *llm.Queue
	if cfg.RedisURL != "" {
		queue, err := llm.NewQueue(cfg.RedisURL)
		if err != nil {
			log.Printf("failed to init redis queue: %v", err)
		} else {
			llmQueue = queue
		}
	}
	llmStore := llm.NewStore(store, cfg.MasterKey)
	llmFactory := llm.NewFactory()
	llmRouter := llm.NewRouter(llmFactory, llmStore)
	llmService := llm.NewService(llmRouter, llmStore)
	healthMonitor := &llm.HealthMonitor{Router: llmRouter, Store: llmStore}
	healthScheduler := llm.NewHealthScheduler(healthMonitor, llmStore)
	var workerScheduler *llm.WorkerScheduler
	if llmQueue != nil {
		workerScheduler = llm.NewWorkerScheduler(llmQueue, llmService, store, hub)
	}

	var waManager *whatsapp.Manager
	if cfg.DatabaseURL != "" {
		manager, err := whatsapp.NewManager(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Printf("failed to init whatsapp manager: %v", err)
		} else {
			waManager = manager
		}
	}
	if waManager != nil {
		waSyncer := whatsapp.NewSyncer(store, llmQueue, hub)
		waManager.SetSyncer(waSyncer)
		// Auto-reconnect existing WhatsApp sessions on startup
		go func() {
			reconnectCtx := context.Background()
			if err := waManager.AutoReconnect(reconnectCtx); err != nil {
				log.Printf("whatsapp auto-reconnect error: %v", err)
			}
		}()
	}

	api := handlers.NewAPI(store, authService, hub, llmService, llmStore, llmQueue, healthScheduler, workerScheduler, waManager)
	limiter := middleware.NewRateLimiter(60, time.Minute)
	rt := router.New(api, authService, limiter, cfg.FrontendOrigin, hub)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      rt,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
