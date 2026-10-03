package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kuldeep-poonia/homeboard/backend/internal/config"
	"github.com/kuldeep-poonia/homeboard/backend/internal/db"
	"github.com/kuldeep-poonia/homeboard/backend/internal/router"
	"github.com/kuldeep-poonia/homeboard/backend/internal/ws"
)

func main() {
	log.Println("Starting HomeBoard server...")

	cfg := config.Load()

	// Initialize SQLite database
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("Fatal error opening database: %v", err)
	}
	defer database.Close()
	log.Printf("Connected to database at %s", cfg.DBPath)

	// Initialize WebSocket hub for live sync
	hub := ws.NewHub(cfg.MaxDevicesPerBoard)
	go hub.Run()
	log.Println("WebSocket live-sync hub started")

	// Setup HTTP router
	appHandler := router.NewRouter(cfg, database, hub)

	addr := cfg.Host + ":" + cfg.Port
	server := &http.Server{
		Addr:              addr,
		Handler:           appHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Channel to listen for interrupt signals for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("HomeBoard listening on %s (Environment: %s)", addr, cfg.AppEnv)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down HomeBoard gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced shutdown: %v", err)
	}

	log.Println("HomeBoard stopped cleanly.")
}
