package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/wealthpath/backend/docs"
	"github.com/wealthpath/backend/internal/handler"
	"github.com/wealthpath/backend/internal/repository"
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
	dbURL := os.Getenv("DATABASE_URL")
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

	// Initialize handlers
	authHandler := handler.NewAuthHandler(userService)
	oauthHandler := handler.NewOAuthHandler(userService)
	transactionHandler := handler.NewTransactionHandler(transactionService)
	budgetHandler := handler.NewBudgetHandler(budgetService)
	savingsHandler := handler.NewSavingsGoalHandler(savingsService)
	debtHandler := handler.NewDebtHandler(debtService)
	recurringHandler := handler.NewRecurringHandler(recurringService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)
	aiHandler := handler.NewAIHandler(aiService)
	interestRateHandler := handler.NewInterestRateHandler(interestRateService)

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	// CORS - allow frontend origin from env or default
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3000"
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{allowedOrigins},
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

	// OAuth routes - supports facebook, google, etc.
	r.Get("/api/auth/{provider}", oauthHandler.OAuthLogin)
	r.Get("/api/auth/{provider}/callback", oauthHandler.OAuthCallback)
	r.Post("/api/auth/{provider}/token", oauthHandler.OAuthToken)

	// Interest rates (public - no auth required)
	r.Get("/api/interest-rates", interestRateHandler.ListRates)
	r.Get("/api/interest-rates/best", interestRateHandler.GetBestRates)
	r.Get("/api/interest-rates/compare", interestRateHandler.CompareRates)
	r.Get("/api/interest-rates/banks", interestRateHandler.GetBanks)
	r.Get("/api/interest-rates/history", interestRateHandler.GetHistory)
	r.Post("/api/interest-rates/seed", interestRateHandler.SeedRates)     // Admin: seed sample data
	r.Post("/api/interest-rates/scrape", interestRateHandler.ScrapeRates) // Admin: scrape live rates

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(handler.AuthMiddleware)

		// Current user
		r.Get("/api/auth/me", authHandler.Me)
		r.Put("/api/auth/settings", authHandler.UpdateSettings)

		// Dashboard
		r.Get("/api/dashboard", dashboardHandler.GetDashboard)
		r.Get("/api/dashboard/monthly/{year}/{month}", dashboardHandler.GetMonthlyDashboard)

		// Transactions
		r.Get("/api/transactions", transactionHandler.List)
		r.Post("/api/transactions", transactionHandler.Create)
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
		r.Get("/api/debts/{id}", debtHandler.Get)
		r.Put("/api/debts/{id}", debtHandler.Update)
		r.Delete("/api/debts/{id}", debtHandler.Delete)
		r.Post("/api/debts/{id}/payment", debtHandler.MakePayment)
		r.Get("/api/debts/{id}/payoff-plan", debtHandler.GetPayoffPlan)
		r.Get("/api/debts/calculator", debtHandler.InterestCalculator)

		// Recurring Transactions
		r.Get("/api/recurring", recurringHandler.List)
		r.Post("/api/recurring", recurringHandler.Create)
		r.Get("/api/recurring/upcoming", recurringHandler.Upcoming)
		r.Get("/api/recurring/{id}", recurringHandler.Get)
		r.Put("/api/recurring/{id}", recurringHandler.Update)
		r.Delete("/api/recurring/{id}", recurringHandler.Delete)
		r.Post("/api/recurring/{id}/pause", recurringHandler.Pause)
		r.Post("/api/recurring/{id}/resume", recurringHandler.Resume)

		// AI Chat
		r.Post("/api/chat", aiHandler.Chat)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Printf("Server failed: %v", err)
	}
}
