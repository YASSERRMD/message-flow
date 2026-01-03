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
	"message-flow/backend/internal/middleware"
	"message-flow/backend/internal/realtime"
	"message-flow/backend/internal/router"
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
	api := handlers.NewAPI(store, authService, hub)
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
