package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/config"
	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/handlers"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// bearerAuthRequired wraps the given handler so that requests must carry a
// valid JWT – mirroring the way the generated openapi middleware applies auth.
func bearerAuthRequired(jwtSecret string, h http.HandlerFunc) http.HandlerFunc {
	jwtMw := auth.Middleware(jwtSecret)
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), openapi.BearerAuthScopes, []string{})
		jwtMw(h).ServeHTTP(w, r.WithContext(ctx))
	}
}

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

	// Open the database if already configured (e.g. on subsequent launches).
	// A missing file is not created here so that the first-run setup decides
	// where the database lives.
	var conn *sql.DB
	if cfg.DatabasePath != "" {
		sqlitePath := dbpkg.CleanPath(cfg.DatabasePath)
		if _, statErr := os.Stat(sqlitePath); os.IsNotExist(statErr) {
			log.Printf("warning: sqlite database missing at %s; skipping auto-open to force setup", sqlitePath)
		} else if statErr != nil {
			log.Printf("warning: failed to stat sqlite database %s: %v", sqlitePath, statErr)
		} else if opened, dbErr := dbpkg.Open(context.Background(), sqlitePath); dbErr != nil {
			log.Printf("warning: failed to open database: %v", dbErr)
		} else {
			conn = opened
			defer conn.Close()
		}
	}

	router := chi.NewRouter()
	router.Use(middleware.RealIP)
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(corsMiddleware)

	api := handlers.New(&cfg, conn)
	openapi.HandlerWithOptions(api, openapi.ChiServerOptions{
		BaseURL:     "/api/v2",
		BaseRouter:  router,
		Middlewares: []openapi.MiddlewareFunc{api.ScopeGuard, auth.Middleware(cfg.JWTSecret)},
	})

	// Backup / restore endpoints are not part of the generated OpenAPI spec, so
	// they are wired up manually here with the same JWT auth enforcement.
	router.Post("/api/v2/system/backup", bearerAuthRequired(cfg.JWTSecret, api.BackupDatabase))
	router.Post("/api/v2/system/restore", bearerAuthRequired(cfg.JWTSecret, api.RestoreDatabase))

	// GDPR data export for a member (Art. 15/20), PDF or JSON.
	router.Get("/api/v2/members/{id}/data-export", bearerAuthRequired(cfg.JWTSecret, api.ExportMemberData))

	// Jubilees and honors (Jubiläen & Ehrungen).
	router.Get("/api/v2/jubilees", bearerAuthRequired(cfg.JWTSecret, api.ListJubilees))
	router.Get("/api/v2/honors", bearerAuthRequired(cfg.JWTSecret, api.ListHonors))
	router.Delete("/api/v2/honors/{id}", bearerAuthRequired(cfg.JWTSecret, api.DeleteHonor))
	router.Post("/api/v2/members/{id}/honors", bearerAuthRequired(cfg.JWTSecret, api.CreateHonor))
	router.Get("/api/v2/members/{id}/membership-adjustment", bearerAuthRequired(cfg.JWTSecret, api.GetMembershipAdjust))
	router.Put("/api/v2/members/{id}/membership-adjustment", bearerAuthRequired(cfg.JWTSecret, api.SetMembershipAdjust))

	// Qualifications and licences with expiry tracking.
	router.Get("/api/v2/qualification-types", bearerAuthRequired(cfg.JWTSecret, api.ListQualificationTypes))
	router.Post("/api/v2/qualification-types", bearerAuthRequired(cfg.JWTSecret, api.CreateQualificationType))
	router.Put("/api/v2/qualification-types/{id}", bearerAuthRequired(cfg.JWTSecret, api.UpdateQualificationType))
	router.Delete("/api/v2/qualification-types/{id}", bearerAuthRequired(cfg.JWTSecret, api.DeleteQualificationType))
	router.Get("/api/v2/qualifications", bearerAuthRequired(cfg.JWTSecret, api.ListQualifications))
	router.Delete("/api/v2/qualifications/{id}", bearerAuthRequired(cfg.JWTSecret, api.DeleteQualification))
	router.Post("/api/v2/members/{id}/qualifications", bearerAuthRequired(cfg.JWTSecret, api.CreateQualification))

	// Tasks for the board and roles.
	router.Get("/api/v2/tasks", bearerAuthRequired(cfg.JWTSecret, api.ListTasks))
	router.Post("/api/v2/tasks", bearerAuthRequired(cfg.JWTSecret, api.CreateTask))
	router.Put("/api/v2/tasks/{id}", bearerAuthRequired(cfg.JWTSecret, api.UpdateTask))
	router.Delete("/api/v2/tasks/{id}", bearerAuthRequired(cfg.JWTSecret, api.DeleteTask))

	// Chart statistics endpoint – extends the existing statistics API with
	// chart-ready time series and distribution data.
	router.Get("/api/v2/statistics/charts", bearerAuthRequired(cfg.JWTSecret, api.GetStatisticsCharts))

	// Refresh the collaboration club in the background (spec §18.2).
	collabCtx, stopCollab := context.WithCancel(context.Background())
	defer stopCollab()
	go api.Collab().Run(collabCtx)

	srv := &http.Server{
		Addr:              net.JoinHostPort(cfg.BindAddress, strconv.Itoa(cfg.Port)),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("listening on %s", srv.Addr)
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
