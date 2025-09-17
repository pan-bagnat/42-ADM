package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"adm-backend/internal/api"
	"adm-backend/internal/db"
	"adm-backend/internal/panbagnat"
	"adm-backend/internal/server"
	"adm-backend/internal/store"
)

func main() {
	ctx := context.Background()

	port := 3000
	if v := os.Getenv("PORT"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			port = parsed
		}
	}

	dbURL := os.Getenv("DATABASE_URL")
	dbConn, err := connectWithRetry(ctx, dbURL, 10, 3*time.Second)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer dbConn.Close()

	sessionStore := store.NewSessionStore(dbConn)
	panClient := panbagnat.NewClient(os.Getenv("PAN_BAGNAT_API_BASE_URL"))
	serviceToken := os.Getenv("PAN_BAGNAT_SERVICE_TOKEN")
	adminHandler := &api.AdminHandler{
		Sessions:     sessionStore,
		Client:       panClient,
		ServiceToken: serviceToken,
	}

	allowedOrigins := parseAllowedOrigins(os.Getenv("CORS_ORIGIN"))

	addr := ":" + strconv.Itoa(port)
	router := server.NewRouter(adminHandler, allowedOrigins)
	httpServer := server.NewHTTPServer(addr, router)

	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("[adm-backend] listening on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-shutdownCtx.Done()
	log.Println("[adm-backend] shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func parseAllowedOrigins(raw string) []string {
	if raw == "" {
		return []string{
			"http://localhost:8080",
			"http://localhost:8081",
		}
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

func connectWithRetry(ctx context.Context, url string, attempts int, delay time.Duration) (*sql.DB, error) {
	if attempts <= 0 {
		attempts = 1
	}

	var lastErr error
	for i := 0; i < attempts; i++ {
		conn, err := db.Connect(ctx, url)
		if err == nil {
			return conn, nil
		}

		lastErr = err
		log.Printf("database connection failed (attempt %d/%d): %v", i+1, attempts, err)

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("database connection failed after %d attempts: %w", attempts, lastErr)
}
