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
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Waschndolos/open-clubmanager/server/internal/auth"
	"github.com/Waschndolos/open-clubmanager/server/internal/config"
	dbpkg "github.com/Waschndolos/open-clubmanager/server/internal/db"
	"github.com/Waschndolos/open-clubmanager/server/internal/handlers"
	"github.com/Waschndolos/open-clubmanager/server/internal/httpguard"
	"github.com/Waschndolos/open-clubmanager/server/internal/openapi"
)

// bearerAuthRequired wraps the given handler so that requests must carry a
// valid JWT.
func bearerAuthRequired(jwtMw openapi.MiddlewareFunc, h http.HandlerFunc) http.HandlerFunc {
	return jwtMw(h).ServeHTTP
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
	// Extra origins / host names come from the config file or, without being
	// persisted, from the environment (the dev scripts set the Vite origin).
	guard := httpguard.New(
		append(append([]string{}, cfg.AllowedOrigins...), httpguard.SplitList(os.Getenv("CORS_ALLOWED_ORIGINS"))...),
		append(append([]string{}, cfg.AllowedHosts...), httpguard.SplitList(os.Getenv("ALLOWED_HOSTS"))...),
	)
	router.Use(guard.Middleware)

	api := handlers.New(&cfg, conn)
	jwtAuth := auth.Middleware(cfg.JWTSecret, api.ValidateSession)
	openapi.HandlerWithOptions(api, openapi.ChiServerOptions{
		BaseURL:     "/api/v2",
		BaseRouter:  router,
		Middlewares: []openapi.MiddlewareFunc{api.ScopeGuard, auth.MiddlewareAllowing(cfg.JWTSecret, api.AllowAnonymous, api.ValidateSession)},
	})

	// Backup / restore endpoints are not part of the generated OpenAPI spec, so
	// they are wired up manually here with the same JWT auth enforcement.
	router.Post("/api/v2/system/backup", bearerAuthRequired(jwtAuth, api.BackupDatabase))
	router.Post("/api/v2/system/restore", bearerAuthRequired(jwtAuth, api.RestoreDatabase))

	// GDPR data export for a member (Art. 15/20), PDF or JSON.
	router.Get("/api/v2/members/{id}/data-export", bearerAuthRequired(jwtAuth, api.ExportMemberData))

	// Jubilees and honors (Jubiläen & Ehrungen).
	router.Get("/api/v2/jubilees", bearerAuthRequired(jwtAuth, api.ListJubilees))
	router.Get("/api/v2/honors", bearerAuthRequired(jwtAuth, api.ListHonors))
	router.Delete("/api/v2/honors/{id}", bearerAuthRequired(jwtAuth, api.DeleteHonor))
	router.Post("/api/v2/members/{id}/honors", bearerAuthRequired(jwtAuth, api.CreateHonor))
	router.Get("/api/v2/members/{id}/membership-adjustment", bearerAuthRequired(jwtAuth, api.GetMembershipAdjust))
	router.Put("/api/v2/members/{id}/membership-adjustment", bearerAuthRequired(jwtAuth, api.SetMembershipAdjust))

	// Qualifications and licences with expiry tracking.
	router.Get("/api/v2/qualification-types", bearerAuthRequired(jwtAuth, api.ListQualificationTypes))
	router.Post("/api/v2/qualification-types", bearerAuthRequired(jwtAuth, api.CreateQualificationType))
	router.Put("/api/v2/qualification-types/{id}", bearerAuthRequired(jwtAuth, api.UpdateQualificationType))
	router.Delete("/api/v2/qualification-types/{id}", bearerAuthRequired(jwtAuth, api.DeleteQualificationType))
	router.Get("/api/v2/qualifications", bearerAuthRequired(jwtAuth, api.ListQualifications))
	router.Delete("/api/v2/qualifications/{id}", bearerAuthRequired(jwtAuth, api.DeleteQualification))
	router.Post("/api/v2/members/{id}/qualifications", bearerAuthRequired(jwtAuth, api.CreateQualification))

	// Tasks for the board and roles.
	router.Get("/api/v2/tasks", bearerAuthRequired(jwtAuth, api.ListTasks))
	router.Post("/api/v2/tasks", bearerAuthRequired(jwtAuth, api.CreateTask))
	router.Put("/api/v2/tasks/{id}", bearerAuthRequired(jwtAuth, api.UpdateTask))
	router.Delete("/api/v2/tasks/{id}", bearerAuthRequired(jwtAuth, api.DeleteTask))

	// Inventory condition, inspection/maintenance schedule and history.
	router.Get("/api/v2/inventory/maintenance", bearerAuthRequired(jwtAuth, api.ListMaintenanceStates))
	router.Delete("/api/v2/inventory/maintenance/{id}", bearerAuthRequired(jwtAuth, api.DeleteMaintenance))
	router.Put("/api/v2/inventory/items/{id}/status", bearerAuthRequired(jwtAuth, api.SetItemStatus))
	router.Get("/api/v2/inventory/items/{id}/maintenance", bearerAuthRequired(jwtAuth, api.ListItemMaintenance))
	router.Post("/api/v2/inventory/items/{id}/maintenance", bearerAuthRequired(jwtAuth, api.RecordMaintenance))

	// Attendance at event occurrences.
	router.Get("/api/v2/attendance/occurrences", bearerAuthRequired(jwtAuth, api.ListAttendanceOccurrences))
	router.Get("/api/v2/attendance/sheet", bearerAuthRequired(jwtAuth, api.GetAttendanceSheet))
	router.Put("/api/v2/attendance/sheet", bearerAuthRequired(jwtAuth, api.SetAttendance))
	router.Get("/api/v2/attendance/stats", bearerAuthRequired(jwtAuth, api.AttendanceStats))
	router.Get("/api/v2/attendance/export", bearerAuthRequired(jwtAuth, api.ExportAttendance))

	// Households (families) and guardians.
	router.Get("/api/v2/households", bearerAuthRequired(jwtAuth, api.ListHouseholds))
	router.Post("/api/v2/households", bearerAuthRequired(jwtAuth, api.CreateHousehold))
	router.Put("/api/v2/households/{id}", bearerAuthRequired(jwtAuth, api.UpdateHousehold))
	router.Delete("/api/v2/households/{id}", bearerAuthRequired(jwtAuth, api.DeleteHousehold))
	router.Get("/api/v2/members/{id}/guardians", bearerAuthRequired(jwtAuth, api.ListGuardians))
	router.Post("/api/v2/members/{id}/guardians", bearerAuthRequired(jwtAuth, api.CreateGuardian))
	router.Delete("/api/v2/guardians/{id}", bearerAuthRequired(jwtAuth, api.DeleteGuardian))

	// SEPA direct debit runs (pain.008) for member fees.
	router.Get("/api/v2/sepa/creditor", bearerAuthRequired(jwtAuth, api.GetSEPACreditor))
	router.Put("/api/v2/sepa/creditor", bearerAuthRequired(jwtAuth, api.SetSEPACreditor))
	router.Get("/api/v2/sepa/candidates", bearerAuthRequired(jwtAuth, api.ListSEPACandidates))
	router.Get("/api/v2/sepa/runs", bearerAuthRequired(jwtAuth, api.ListSEPARuns))
	router.Post("/api/v2/sepa/runs", bearerAuthRequired(jwtAuth, api.CreateSEPARun))
	router.Get("/api/v2/sepa/runs/{id}/xml", bearerAuthRequired(jwtAuth, api.DownloadSEPARun))
	router.Post("/api/v2/sepa/runs/{id}/confirm", bearerAuthRequired(jwtAuth, api.ConfirmSEPARun))
	router.Delete("/api/v2/sepa/runs/{id}", bearerAuthRequired(jwtAuth, api.CancelSEPARun))
	router.Get("/api/v2/members/{id}/sepa-mandate", bearerAuthRequired(jwtAuth, api.GetSEPAMandate))
	router.Put("/api/v2/members/{id}/sepa-mandate", bearerAuthRequired(jwtAuth, api.SetSEPAMandate))

	// Fee models and billing runs (Beitragsmodelle & Sollstellung).
	router.Get("/api/v2/fee-models", bearerAuthRequired(jwtAuth, api.ListFeeModels))
	router.Post("/api/v2/fee-models", bearerAuthRequired(jwtAuth, api.CreateFeeModel))
	router.Get("/api/v2/fee-models/members", bearerAuthRequired(jwtAuth, api.ListMemberFeeModels))
	router.Put("/api/v2/fee-models/{id}", bearerAuthRequired(jwtAuth, api.UpdateFeeModel))
	router.Delete("/api/v2/fee-models/{id}", bearerAuthRequired(jwtAuth, api.DeleteFeeModel))
	router.Get("/api/v2/members/{id}/fee-model", bearerAuthRequired(jwtAuth, api.GetMemberFeeModel))
	router.Put("/api/v2/members/{id}/fee-model", bearerAuthRequired(jwtAuth, api.SetMemberFeeModel))
	router.Get("/api/v2/fee-runs", bearerAuthRequired(jwtAuth, api.ListFeeRuns))
	router.Post("/api/v2/fee-runs", bearerAuthRequired(jwtAuth, api.CreateFeeRun))
	router.Post("/api/v2/fee-runs/preview", bearerAuthRequired(jwtAuth, api.PreviewFeeRun))
	router.Delete("/api/v2/fee-runs/{id}", bearerAuthRequired(jwtAuth, api.DeleteFeeRun))

	// Payment reconciliation: imported bank transactions assigned to member fees.
	router.Get("/api/v2/reconciliation", bearerAuthRequired(jwtAuth, api.GetReconciliation))
	router.Post("/api/v2/reconciliation/confirm-certain", bearerAuthRequired(jwtAuth, api.ConfirmCertainReconciliations))
	router.Post("/api/v2/reconciliation/{id}/confirm", bearerAuthRequired(jwtAuth, api.ConfirmReconciliation))
	router.Post("/api/v2/reconciliation/{id}/dismiss", bearerAuthRequired(jwtAuth, api.DismissReconciliation))
	router.Post("/api/v2/reconciliation/{id}/reopen", bearerAuthRequired(jwtAuth, api.ReopenReconciliation))

	// Donation receipts (Zuwendungsbestätigungen) after the official template.
	router.Get("/api/v2/donations", bearerAuthRequired(jwtAuth, api.ListDonations))
	router.Get("/api/v2/donations/settings", bearerAuthRequired(jwtAuth, api.GetDonationSettings))
	router.Put("/api/v2/donations/settings", bearerAuthRequired(jwtAuth, api.SetDonationSettings))
	router.Get("/api/v2/donations/donors", bearerAuthRequired(jwtAuth, api.ListDonors))
	router.Post("/api/v2/donations/donors", bearerAuthRequired(jwtAuth, api.CreateDonor))
	router.Put("/api/v2/donations/donors/{id}", bearerAuthRequired(jwtAuth, api.UpdateDonor))
	router.Delete("/api/v2/donations/donors/{id}", bearerAuthRequired(jwtAuth, api.DeleteDonor))
	router.Get("/api/v2/donations/candidates", bearerAuthRequired(jwtAuth, api.ListDonationCandidates))
	router.Get("/api/v2/donations/receipts", bearerAuthRequired(jwtAuth, api.ListDonationReceipts))
	router.Post("/api/v2/donations/receipts", bearerAuthRequired(jwtAuth, api.CreateDonationReceipt))
	router.Post("/api/v2/donations/receipts/annual", bearerAuthRequired(jwtAuth, api.CreateAnnualDonationReceipts))
	router.Get("/api/v2/donations/receipts/pdf", bearerAuthRequired(jwtAuth, api.DownloadDonationReceipts))
	router.Get("/api/v2/donations/receipts/{id}/pdf", bearerAuthRequired(jwtAuth, api.DownloadDonationReceipts))
	router.Post("/api/v2/donations/receipts/{id}/duplicate", bearerAuthRequired(jwtAuth, api.DuplicateDonationReceipt))
	router.Post("/api/v2/donations/receipts/{id}/void", bearerAuthRequired(jwtAuth, api.VoidDonationReceipt))
	router.Put("/api/v2/donations/{id}", bearerAuthRequired(jwtAuth, api.SetDonation))
	router.Delete("/api/v2/donations/{id}", bearerAuthRequired(jwtAuth, api.RemoveDonation))

	// Chart statistics endpoint – extends the existing statistics API with
	// chart-ready time series and distribution data.
	router.Get("/api/v2/statistics/charts", bearerAuthRequired(jwtAuth, api.GetStatisticsCharts))

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
