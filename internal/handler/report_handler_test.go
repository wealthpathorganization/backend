package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wealthpath/backend/internal/service"
)

// MockReportServiceImpl implements ReportServiceInterface for testing
type MockReportServiceImpl struct {
	mock.Mock
}

func (m *MockReportServiceImpl) GetMonthlyReport(ctx context.Context, userID uuid.UUID, year, month int) (*service.MonthlyReport, error) {
	args := m.Called(ctx, userID, year, month)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.MonthlyReport), args.Error(1)
}

func (m *MockReportServiceImpl) GetCategoryTrends(ctx context.Context, userID uuid.UUID, months, limit int) (*service.CategoryTrendsResponse, error) {
	args := m.Called(ctx, userID, months, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.CategoryTrendsResponse), args.Error(1)
}

func TestReportHandler_GetMonthlyReport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     uuid.UUID
		year       string
		month      string
		setupMock  func(*MockReportServiceImpl, uuid.UUID)
		wantStatus int
	}{
		{
			name:   "success - valid parameters",
			userID: uuid.New(),
			year:   "2026",
			month:  "1",
			setupMock: func(m *MockReportServiceImpl, userID uuid.UUID) {
				m.On("GetMonthlyReport", mock.Anything, userID, 2026, 1).Return(&service.MonthlyReport{
					Year:          2026,
					Month:         1,
					Currency:      "USD",
					TotalIncome:   "8500.00",
					TotalExpenses: "5200.00",
					NetSavings:    "3300.00",
					SavingsRate:   38.82,
					TopCategories: []service.TopCategory{
						{Category: "Housing", Amount: "1800.00", Percentage: 34.62, TransactionCount: 2},
					},
					Anomalies:      []service.Anomaly{},
					ComparedToLast: service.MonthComparison{Trend: "improving"},
					GeneratedAt:    time.Now().Format(time.RFC3339),
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthorized - nil userID",
			userID:     uuid.Nil,
			year:       "2026",
			month:      "1",
			setupMock:  func(m *MockReportServiceImpl, userID uuid.UUID) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "bad request - missing year",
			userID:     uuid.New(),
			year:       "",
			month:      "1",
			setupMock:  func(m *MockReportServiceImpl, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "bad request - missing month",
			userID:     uuid.New(),
			year:       "2026",
			month:      "",
			setupMock:  func(m *MockReportServiceImpl, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "bad request - invalid year format",
			userID:     uuid.New(),
			year:       "invalid",
			month:      "1",
			setupMock:  func(m *MockReportServiceImpl, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "bad request - invalid month format",
			userID:     uuid.New(),
			year:       "2026",
			month:      "invalid",
			setupMock:  func(m *MockReportServiceImpl, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "bad request - year too old (< 1900)",
			userID:     uuid.New(),
			year:       "1800",
			month:      "1",
			setupMock:  func(m *MockReportServiceImpl, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "bad request - year too far future (> 2100)",
			userID:     uuid.New(),
			year:       "2200",
			month:      "1",
			setupMock:  func(m *MockReportServiceImpl, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "bad request - month < 1",
			userID:     uuid.New(),
			year:       "2026",
			month:      "0",
			setupMock:  func(m *MockReportServiceImpl, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "bad request - month > 12",
			userID:     uuid.New(),
			year:       "2026",
			month:      "13",
			setupMock:  func(m *MockReportServiceImpl, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "internal server error - service failure",
			userID: uuid.New(),
			year:   "2026",
			month:  "1",
			setupMock: func(m *MockReportServiceImpl, userID uuid.UUID) {
				m.On("GetMonthlyReport", mock.Anything, userID, 2026, 1).Return(nil, errors.New("database error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "success - boundary month January",
			userID: uuid.New(),
			year:   "2026",
			month:  "1",
			setupMock: func(m *MockReportServiceImpl, userID uuid.UUID) {
				m.On("GetMonthlyReport", mock.Anything, userID, 2026, 1).Return(&service.MonthlyReport{
					Year:  2026,
					Month: 1,
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "success - boundary month December",
			userID: uuid.New(),
			year:   "2026",
			month:  "12",
			setupMock: func(m *MockReportServiceImpl, userID uuid.UUID) {
				m.On("GetMonthlyReport", mock.Anything, userID, 2026, 12).Return(&service.MonthlyReport{
					Year:  2026,
					Month: 12,
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockReportServiceImpl)
			handler := NewReportHandler(mockService)

			tt.setupMock(mockService, tt.userID)

			url := "/api/reports/monthly"
			if tt.year != "" || tt.month != "" {
				url += "?"
				if tt.year != "" {
					url += "year=" + tt.year
				}
				if tt.month != "" {
					if tt.year != "" {
						url += "&"
					}
					url += "month=" + tt.month
				}
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			req = req.WithContext(context.WithValue(context.Background(), UserIDKey, tt.userID))
			w := httptest.NewRecorder()

			handler.GetMonthlyReport(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestReportHandler_GetCategoryTrends(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     uuid.UUID
		months     string
		limit      string
		setupMock  func(*MockReportServiceImpl, uuid.UUID, int, int)
		wantStatus int
	}{
		{
			name:   "success - default parameters",
			userID: uuid.New(),
			months: "",
			limit:  "",
			setupMock: func(m *MockReportServiceImpl, userID uuid.UUID, months, limit int) {
				m.On("GetCategoryTrends", mock.Anything, userID, 6, 10).Return(&service.CategoryTrendsResponse{
					Currency:    "USD",
					PeriodStart: "2025-08-01",
					PeriodEnd:   "2026-01-31",
					Trends: []service.CategoryTrend{
						{
							Category:        "Housing",
							TotalAmount:     "10800.00",
							AverageAmount:   "1800.00",
							TrendDirection:  "stable",
							TrendPercentage: 0.0,
							MonthlyData: []service.MonthlyAmount{
								{Month: "2025-08", Amount: "1800.00"},
							},
						},
					},
					GeneratedAt: time.Now().Format(time.RFC3339),
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "success - custom months parameter",
			userID: uuid.New(),
			months: "12",
			limit:  "",
			setupMock: func(m *MockReportServiceImpl, userID uuid.UUID, months, limit int) {
				m.On("GetCategoryTrends", mock.Anything, userID, 12, 10).Return(&service.CategoryTrendsResponse{
					Currency: "USD",
					Trends:   []service.CategoryTrend{},
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "success - custom limit parameter",
			userID: uuid.New(),
			months: "",
			limit:  "5",
			setupMock: func(m *MockReportServiceImpl, userID uuid.UUID, months, limit int) {
				m.On("GetCategoryTrends", mock.Anything, userID, 6, 5).Return(&service.CategoryTrendsResponse{
					Currency: "USD",
					Trends:   []service.CategoryTrend{},
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthorized - nil userID",
			userID:     uuid.Nil,
			months:     "",
			limit:      "",
			setupMock:  func(m *MockReportServiceImpl, userID uuid.UUID, months, limit int) {},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "bad request - months > 24",
			userID:     uuid.New(),
			months:     "25",
			limit:      "",
			setupMock:  func(m *MockReportServiceImpl, userID uuid.UUID, months, limit int) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "bad request - limit > 20",
			userID:     uuid.New(),
			months:     "",
			limit:      "25",
			setupMock:  func(m *MockReportServiceImpl, userID uuid.UUID, months, limit int) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "internal server error - service failure",
			userID: uuid.New(),
			months: "",
			limit:  "",
			setupMock: func(m *MockReportServiceImpl, userID uuid.UUID, months, limit int) {
				m.On("GetCategoryTrends", mock.Anything, userID, 6, 10).Return(nil, errors.New("database error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "success - boundary months = 1",
			userID: uuid.New(),
			months: "1",
			limit:  "",
			setupMock: func(m *MockReportServiceImpl, userID uuid.UUID, months, limit int) {
				m.On("GetCategoryTrends", mock.Anything, userID, 1, 10).Return(&service.CategoryTrendsResponse{
					Currency: "USD",
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "success - boundary months = 24",
			userID: uuid.New(),
			months: "24",
			limit:  "",
			setupMock: func(m *MockReportServiceImpl, userID uuid.UUID, months, limit int) {
				m.On("GetCategoryTrends", mock.Anything, userID, 24, 10).Return(&service.CategoryTrendsResponse{
					Currency: "USD",
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "success - boundary limit = 1",
			userID: uuid.New(),
			months: "",
			limit:  "1",
			setupMock: func(m *MockReportServiceImpl, userID uuid.UUID, months, limit int) {
				m.On("GetCategoryTrends", mock.Anything, userID, 6, 1).Return(&service.CategoryTrendsResponse{
					Currency: "USD",
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "success - boundary limit = 20",
			userID: uuid.New(),
			months: "",
			limit:  "20",
			setupMock: func(m *MockReportServiceImpl, userID uuid.UUID, months, limit int) {
				m.On("GetCategoryTrends", mock.Anything, userID, 6, 20).Return(&service.CategoryTrendsResponse{
					Currency: "USD",
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockReportServiceImpl)
			handler := NewReportHandler(mockService)

			// Determine expected months and limit for mock setup
			expectedMonths := 6
			expectedLimit := 10

			tt.setupMock(mockService, tt.userID, expectedMonths, expectedLimit)

			url := "/api/reports/category-trends"
			if tt.months != "" || tt.limit != "" {
				url += "?"
				if tt.months != "" {
					url += "months=" + tt.months
				}
				if tt.limit != "" {
					if tt.months != "" {
						url += "&"
					}
					url += "limit=" + tt.limit
				}
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			req = req.WithContext(context.WithValue(context.Background(), UserIDKey, tt.userID))
			w := httptest.NewRecorder()

			handler.GetCategoryTrends(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// TestReportHandler_ResponseFormat verifies the JSON response format
func TestReportHandler_ResponseFormat(t *testing.T) {
	t.Parallel()

	mockService := new(MockReportServiceImpl)
	handler := NewReportHandler(mockService)
	userID := uuid.New()

	expectedReport := &service.MonthlyReport{
		Year:          2026,
		Month:         1,
		Currency:      "USD",
		TotalIncome:   "8500.00",
		TotalExpenses: "5200.00",
		NetSavings:    "3300.00",
		SavingsRate:   38.82,
		TopCategories: []service.TopCategory{
			{
				Category:         "Housing",
				Amount:           "1800.00",
				Percentage:       34.62,
				TransactionCount: 2,
			},
		},
		Anomalies: []service.Anomaly{
			{
				Type:        "unusual_expense",
				Category:    "Shopping",
				Amount:      "450.00",
				Description: "Spending 85% higher than your 3-month average",
				Severity:    "warning",
			},
		},
		ComparedToLast: service.MonthComparison{
			IncomeChange:  5.50,
			ExpenseChange: -8.25,
			SavingsChange: 24.10,
			Trend:         "improving",
		},
		GeneratedAt: time.Now().Format(time.RFC3339),
	}

	mockService.On("GetMonthlyReport", mock.Anything, userID, 2026, 1).Return(expectedReport, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/reports/monthly?year=2026&month=1", nil)
	req = req.WithContext(context.WithValue(context.Background(), UserIDKey, userID))
	w := httptest.NewRecorder()

	handler.GetMonthlyReport(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "totalIncome")
	assert.Contains(t, w.Body.String(), "topCategories")
	assert.Contains(t, w.Body.String(), "anomalies")
	assert.Contains(t, w.Body.String(), "comparedToLast")
}

// TestReportHandler_EmptyState tests the handler with empty data
func TestReportHandler_EmptyState(t *testing.T) {
	t.Parallel()

	mockService := new(MockReportServiceImpl)
	handler := NewReportHandler(mockService)
	userID := uuid.New()

	emptyReport := &service.MonthlyReport{
		Year:           2026,
		Month:          1,
		Currency:       "USD",
		TotalIncome:    "0.00",
		TotalExpenses:  "0.00",
		NetSavings:     "0.00",
		SavingsRate:    0,
		TopCategories:  []service.TopCategory{},
		Anomalies:      []service.Anomaly{},
		ComparedToLast: service.MonthComparison{Trend: "stable"},
		GeneratedAt:    time.Now().Format(time.RFC3339),
	}

	mockService.On("GetMonthlyReport", mock.Anything, userID, 2026, 1).Return(emptyReport, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/reports/monthly?year=2026&month=1", nil)
	req = req.WithContext(context.WithValue(context.Background(), UserIDKey, userID))
	w := httptest.NewRecorder()

	handler.GetMonthlyReport(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Empty arrays should be present, not null
	assert.Contains(t, w.Body.String(), "topCategories")
}

// Benchmark tests
func BenchmarkReportHandler_GetMonthlyReport(b *testing.B) {
	mockService := new(MockReportServiceImpl)
	handler := NewReportHandler(mockService)
	userID := uuid.New()

	mockService.On("GetMonthlyReport", mock.Anything, userID, 2026, 1).Return(&service.MonthlyReport{
		Year:          2026,
		Month:         1,
		Currency:      "USD",
		TotalIncome:   "8500.00",
		TotalExpenses: "5200.00",
	}, nil)

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/reports/monthly?year=2026&month=1", nil)
		req = req.WithContext(context.WithValue(context.Background(), UserIDKey, userID))
		w := httptest.NewRecorder()
		handler.GetMonthlyReport(w, req)
	}
}

func BenchmarkReportHandler_GetCategoryTrends(b *testing.B) {
	mockService := new(MockReportServiceImpl)
	handler := NewReportHandler(mockService)
	userID := uuid.New()

	mockService.On("GetCategoryTrends", mock.Anything, userID, 6, 10).Return(&service.CategoryTrendsResponse{
		Currency: "USD",
		Trends:   []service.CategoryTrend{},
	}, nil)

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/reports/category-trends", nil)
		req = req.WithContext(context.WithValue(context.Background(), UserIDKey, userID))
		w := httptest.NewRecorder()
		handler.GetCategoryTrends(w, req)
	}
}
