package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/config"
	"github.com/Waschndolos/open-clubmanager/server/internal/handlers"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// corsMiddleware adds CORS headers for the Tauri desktop frontend and the
// Vite dev server. Only recognised origins are reflected; all others are
// silently ignored so the default browser same-origin policy still applies.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isCORSAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isCORSAllowed returns true for origins that may call the API:
//   - tauri://localhost       – Tauri 2 production app (macOS / Linux)
//   - https://tauri.localhost – Tauri 2 production app (Windows WebView2)
//   - http://localhost:<port> – Vite dev server (any port)
func isCORSAllowed(origin string) bool {
	switch origin {
	case "tauri://localhost", "https://tauri.localhost":
		return true
	}
	return strings.HasPrefix(origin, "http://localhost:")
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	router := chi.NewRouter()
	router.Use(middleware.RealIP)
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(corsMiddleware)

	api := handlers.New(cfg)
	openapi.HandlerWithOptions(api, openapi.ChiServerOptions{
		BaseURL:     "/api/v2",
		BaseRouter:  router,
		Middlewares: []openapi.MiddlewareFunc{auth.Middleware(cfg.JWTSecret)},
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGINT, syscall.SIGTERM)
	<-shutdownSignal

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}
}
