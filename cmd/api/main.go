package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/wealthpath/backend/docs"
	"github.com/wealthpath/backend/internal/config"
	"github.com/wealthpath/backend/internal/handler"
	"github.com/wealthpath/backend/internal/repository"
	"github.com/wealthpath/backend/internal/scheduler"
	"github.com/wealthpath/backend/internal/service"
)

// @title WealthPath API
// @version 1.0
// @description Personal finance management API for tracking transactions, budgets, savings goals, and debts.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@wealthpath.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load configuration
	cfg := config.Load()

	// Setup structured logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	dbURL := cfg.DatabaseURL
	if dbURL == "" {
		dbURL = "postgres://localhost:5432/wealthpath?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	transactionRepo := repository.NewTransactionRepository(db)
	budgetRepo := repository.NewBudgetRepository(db)
	savingsRepo := repository.NewSavingsGoalRepository(db)
	debtRepo := repository.NewDebtRepository(db)
	recurringRepo := repository.NewRecurringRepository(db)
	interestRateRepo := repository.NewInterestRateRepository(db)

	// Initialize services
	userService := service.NewUserService(userRepo)
	transactionService := service.NewTransactionService(transactionRepo)
	budgetService := service.NewBudgetService(budgetRepo)
	savingsService := service.NewSavingsGoalService(savingsRepo)
	debtService := service.NewDebtService(debtRepo)
	recurringService := service.NewRecurringService(recurringRepo, transactionRepo)
	dashboardService := service.NewDashboardService(transactionRepo, budgetRepo, savingsRepo, debtRepo)
	aiService := service.NewAIService(transactionService, budgetService, savingsService)
	interestRateService := service.NewInterestRateService(interestRateRepo)
	reportRepo := repository.NewReportRepository(db)
	reportService := service.NewReportService(reportRepo)
	exportService := service.NewExportService(transactionRepo)

	// Initialize TOTP service with repository adapter
	totpRepoAdapter := &TOTPUserRepoAdapter{userRepo: userRepo}
	totpService := service.NewTOTPService(totpRepoAdapter, "WealthPath")

	// Initialize push notification service
	pushRepo := repository.NewPushRepository(db)
	pushService := service.NewPushNotificationService(pushRepo, cfg)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(userService)
	oauthHandler := handler.NewOAuthHandler(userService)
	totpHandler := handler.NewTOTPHandler(totpService)
	transactionHandler := handler.NewTransactionHandler(transactionService)
	budgetHandler := handler.NewBudgetHandler(budgetService)
	savingsHandler := handler.NewSavingsGoalHandler(savingsService)
	debtHandler := handler.NewDebtHandler(debtService)
	recurringHandler := handler.NewRecurringHandler(recurringService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)
	aiHandler := handler.NewAIHandler(aiService)
	interestRateHandler := handler.NewInterestRateHandler(interestRateService)
	reportHandler := handler.NewReportHandler(reportService)
	exportHandler := handler.NewExportHandler(exportService, reportService)
	calendarHandler := handler.NewCalendarHandler(recurringService, func(ctx context.Context, userID uuid.UUID) string {
		currency, _ := reportRepo.GetUserCurrency(ctx, userID)
		return currency
	})
	pushHandler := handler.NewPushHandler(pushService)

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	// CORS - allow frontend origin from env or default
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	// Health check
	// @Summary Health check
	// @Description Check if the API is running
	// @Tags health
	// @Produce json
	// @Success 200 {object} map[string]string
	// @Router /health [get]
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Public routes
	r.Post("/api/auth/register", authHandler.Register)
	r.Post("/api/auth/login", authHandler.Login)
	r.Post("/api/auth/login/2fa", authHandler.LoginWithTOTP)
	r.Post("/api/auth/login/2fa/backup", authHandler.LoginWithBackupCode)

	// OAuth routes - supports facebook, google, etc.
	r.Get("/api/auth/{provider}", oauthHandler.OAuthLogin)
	r.Get("/api/auth/{provider}/callback", oauthHandler.OAuthCallback)
	r.Post("/api/auth/{provider}/token", oauthHandler.OAuthToken)

	// Push notifications (public - VAPID key needed before auth)
	r.Get("/api/notifications/vapid-public-key", pushHandler.GetVAPIDPublicKey)

	// Interest rates (public - no auth required)
	r.Get("/api/interest-rates", interestRateHandler.ListRates)
	r.Get("/api/interest-rates/best", interestRateHandler.GetBestRates)
	r.Get("/api/interest-rates/compare", interestRateHandler.CompareRates)
	r.Get("/api/interest-rates/banks", interestRateHandler.GetBanks)
	r.Get("/api/interest-rates/history", interestRateHandler.GetHistory)
	r.Get("/api/interest-rates/scraper-health", interestRateHandler.GetScraperHealth) // Scraper health status
	r.Post("/api/interest-rates/seed", interestRateHandler.SeedRates)                 // Admin: seed sample data
	r.Post("/api/interest-rates/scrape", interestRateHandler.ScrapeRates)             // Admin: scrape live rates

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(handler.AuthMiddleware)

		// Current user
		r.Get("/api/auth/me", authHandler.Me)
		r.Put("/api/auth/settings", authHandler.UpdateSettings)
		r.Post("/api/auth/refresh", authHandler.RefreshToken)

		// 2FA management
		r.Post("/api/auth/2fa/setup", totpHandler.Setup)
		r.Post("/api/auth/2fa/verify", totpHandler.Verify)
		r.Post("/api/auth/2fa/disable", totpHandler.Disable)
		r.Post("/api/auth/2fa/backup-codes", totpHandler.RegenerateBackupCodes)

		// Dashboard
		r.Get("/api/dashboard", dashboardHandler.GetDashboard)
		r.Get("/api/dashboard/monthly/{year}/{month}", dashboardHandler.GetMonthlyDashboard)

		// Transactions
		r.Get("/api/transactions", transactionHandler.List)
		r.Post("/api/transactions", transactionHandler.Create)
		r.Get("/api/transactions/export/csv", exportHandler.ExportTransactionsCSV)
		r.Get("/api/transactions/{id}", transactionHandler.Get)
		r.Put("/api/transactions/{id}", transactionHandler.Update)
		r.Delete("/api/transactions/{id}", transactionHandler.Delete)

		// Budgets
		r.Get("/api/budgets", budgetHandler.List)
		r.Post("/api/budgets", budgetHandler.Create)
		r.Get("/api/budgets/{id}", budgetHandler.Get)
		r.Put("/api/budgets/{id}", budgetHandler.Update)
		r.Delete("/api/budgets/{id}", budgetHandler.Delete)

		// Savings Goals
		r.Get("/api/savings-goals", savingsHandler.List)
		r.Post("/api/savings-goals", savingsHandler.Create)
		r.Get("/api/savings-goals/{id}", savingsHandler.Get)
		r.Put("/api/savings-goals/{id}", savingsHandler.Update)
		r.Delete("/api/savings-goals/{id}", savingsHandler.Delete)
		r.Post("/api/savings-goals/{id}/contribute", savingsHandler.Contribute)

		// Debt Management
		r.Get("/api/debts", debtHandler.List)
		r.Post("/api/debts", debtHandler.Create)
		r.Get("/api/debts/summary", debtHandler.GetSummary)
		r.Get("/api/debts/calculator", debtHandler.InterestCalculator)
		r.Get("/api/debts/{id}", debtHandler.Get)
		r.Put("/api/debts/{id}", debtHandler.Update)
		r.Delete("/api/debts/{id}", debtHandler.Delete)
		r.Post("/api/debts/{id}/payment", debtHandler.MakePayment)
		r.Get("/api/debts/{id}/payoff-plan", debtHandler.GetPayoffPlan)

		// Recurring Transactions
		r.Get("/api/recurring", recurringHandler.List)
		r.Post("/api/recurring", recurringHandler.Create)
		r.Get("/api/recurring/upcoming", recurringHandler.Upcoming)
		r.Get("/api/recurring/calendar", calendarHandler.GetCalendar)
		r.Get("/api/recurring/{id}", recurringHandler.Get)
		r.Put("/api/recurring/{id}", recurringHandler.Update)
		r.Delete("/api/recurring/{id}", recurringHandler.Delete)
		r.Post("/api/recurring/{id}/pause", recurringHandler.Pause)
		r.Post("/api/recurring/{id}/resume", recurringHandler.Resume)

		// Reports
		r.Get("/api/reports/monthly", reportHandler.GetMonthlyReport)
		r.Get("/api/reports/category-trends", reportHandler.GetCategoryTrends)
		r.Get("/api/reports/monthly/{year}/{month}/export/pdf", exportHandler.ExportMonthlyReportPDF)

		// AI Chat
		r.Post("/api/chat", aiHandler.Chat)

		// Push Notifications
		r.Post("/api/notifications/subscribe", pushHandler.Subscribe)
		r.Delete("/api/notifications/unsubscribe", pushHandler.Unsubscribe)
		r.Get("/api/notifications/preferences", pushHandler.GetPreferences)
		r.Put("/api/notifications/preferences", pushHandler.UpdatePreferences)
	})

	// Initialize and start scheduler for interest rate scraping
	var scraperScheduler *scheduler.Scheduler
	if cfg.ScraperEnabled {
		schedCfg := scheduler.Config{
			Schedule: cfg.ScraperSchedule,
			Timeout:  cfg.ScraperTimeout,
			Enabled:  cfg.ScraperEnabled,
		}
		scraperScheduler = scheduler.New(schedCfg, interestRateService, logger)
		if err := scraperScheduler.Start(); err != nil {
			logger.Error("Failed to start scraper scheduler", slog.String("error", err.Error()))
		} else {
			logger.Info("Scraper scheduler started",
				slog.String("schedule", cfg.ScraperSchedule),
				slog.Duration("timeout", cfg.ScraperTimeout),
			)
		}
	}

	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	// Create server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		logger.Info("Shutting down server...")

		// Stop scheduler first
		if scraperScheduler != nil {
			ctx := scraperScheduler.Stop()
			<-ctx.Done()
			logger.Info("Scheduler stopped")
		}

		// Shutdown HTTP server
		if err := server.Shutdown(context.Background()); err != nil {
			logger.Error("Server shutdown error", slog.String("error", err.Error()))
		}
	}()

	log.Printf("Server starting on port %s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("Server failed: %v", err)
	}
}

// TOTPUserRepoAdapter adapts UserRepository to TOTPUserRepository interface.
type TOTPUserRepoAdapter struct {
	userRepo *repository.UserRepository
}

func (a *TOTPUserRepoAdapter) GetByID(ctx context.Context, id uuid.UUID) (*service.UserEntity, error) {
	user, err := a.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &service.UserEntity{
		ID:              user.ID,
		Email:           user.Email,
		Name:            user.Name,
		TOTPSecret:      user.TOTPSecret,
		TOTPEnabled:     user.TOTPEnabled,
		TOTPBackupCodes: user.TOTPBackupCodes,
		TOTPVerifiedAt:  user.TOTPVerifiedAt,
	}, nil
}

func (a *TOTPUserRepoAdapter) UpdateTOTPSecret(ctx context.Context, userID uuid.UUID, secret *string) error {
	return a.userRepo.UpdateTOTPSecret(ctx, userID, secret)
}

func (a *TOTPUserRepoAdapter) EnableTOTP(ctx context.Context, userID uuid.UUID, backupCodes []string) error {
	return a.userRepo.EnableTOTP(ctx, userID, backupCodes)
}

func (a *TOTPUserRepoAdapter) DisableTOTP(ctx context.Context, userID uuid.UUID) error {
	return a.userRepo.DisableTOTP(ctx, userID)
}

func (a *TOTPUserRepoAdapter) UpdateBackupCodes(ctx context.Context, userID uuid.UUID, codes []string) error {
	return a.userRepo.UpdateBackupCodes(ctx, userID, codes)
}
