package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wealthpath/backend/internal/model"
)

// MockDashboardService for testing
type MockDashboardService struct {
	mock.Mock
}

func (m *MockDashboardService) GetDashboard(ctx context.Context, userID uuid.UUID) (*model.DashboardData, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.DashboardData), args.Error(1)
}

func (m *MockDashboardService) GetMonthlyDashboard(ctx context.Context, userID uuid.UUID, year, month int) (*model.DashboardData, error) {
	args := m.Called(ctx, userID, year, month)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.DashboardData), args.Error(1)
}

// DashboardServiceInterface for handler
type DashboardServiceInterface interface {
	GetDashboard(ctx context.Context, userID uuid.UUID) (*model.DashboardData, error)
	GetMonthlyDashboard(ctx context.Context, userID uuid.UUID, year, month int) (*model.DashboardData, error)
}

// TestableNewDashboardHandler for creating a handler with mock service
type TestableDashboardHandler struct {
	service DashboardServiceInterface
}

func (h *TestableDashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	data, err := h.service.GetDashboard(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get dashboard")
		return
	}

	respondJSON(w, http.StatusOK, data)
}

func (h *TestableDashboardHandler) GetMonthlyDashboard(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r.Context())

	year := chi.URLParam(r, "year")
	month := chi.URLParam(r, "month")

	var yearInt, monthInt int
	if _, err := fmt.Sscanf(year, "%d", &yearInt); err != nil {
		respondError(w, http.StatusBadRequest, "invalid year")
		return
	}

	if _, err := fmt.Sscanf(month, "%d", &monthInt); err != nil || monthInt < 1 || monthInt > 12 {
		respondError(w, http.StatusBadRequest, "invalid month")
		return
	}

	data, err := h.service.GetMonthlyDashboard(r.Context(), userID, yearInt, monthInt)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get dashboard")
		return
	}

	respondJSON(w, http.StatusOK, data)
}

func TestDashboardHandler_GetDashboard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMock  func(*MockDashboardService, uuid.UUID)
		wantStatus int
	}{
		{
			name: "success",
			setupMock: func(m *MockDashboardService, userID uuid.UUID) {
				m.On("GetDashboard", mock.Anything, userID).Return(&model.DashboardData{
					TotalIncome:   decimal.NewFromFloat(5000),
					TotalExpenses: decimal.NewFromFloat(3000),
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "service error",
			setupMock: func(m *MockDashboardService, userID uuid.UUID) {
				m.On("GetDashboard", mock.Anything, userID).Return(nil, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockDashboardService)
			handler := &TestableDashboardHandler{service: mockService}
			userID := uuid.New()

			tt.setupMock(mockService, userID)

			req := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
			req = req.WithContext(ctxWithUserID(userID))
			w := httptest.NewRecorder()

			handler.GetDashboard(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestDashboardHandler_GetMonthlyDashboard(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		year       string
		month      string
		setupMock  func(*MockDashboardService, uuid.UUID)
		wantStatus int
	}{
		{
			name:  "success",
			year:  "2024",
			month: "6",
			setupMock: func(m *MockDashboardService, userID uuid.UUID) {
				m.On("GetMonthlyDashboard", mock.Anything, userID, 2024, 6).Return(&model.DashboardData{
					TotalIncome:   decimal.NewFromFloat(5000),
					TotalExpenses: decimal.NewFromFloat(3000),
				}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid year",
			year:       "invalid",
			month:      "6",
			setupMock:  func(m *MockDashboardService, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid month - non-numeric",
			year:       "2024",
			month:      "invalid",
			setupMock:  func(m *MockDashboardService, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid month - out of range (0)",
			year:       "2024",
			month:      "0",
			setupMock:  func(m *MockDashboardService, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid month - out of range (13)",
			year:       "2024",
			month:      "13",
			setupMock:  func(m *MockDashboardService, userID uuid.UUID) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "service error",
			year:  "2024",
			month: "6",
			setupMock: func(m *MockDashboardService, userID uuid.UUID) {
				m.On("GetMonthlyDashboard", mock.Anything, userID, 2024, 6).Return(nil, errors.New("db error"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockService := new(MockDashboardService)
			handler := &TestableDashboardHandler{service: mockService}
			userID := uuid.New()

			tt.setupMock(mockService, userID)

			req := httptest.NewRequest(http.MethodGet, "/api/dashboard/"+tt.year+"/"+tt.month, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("year", tt.year)
			rctx.URLParams.Add("month", tt.month)
			req = req.WithContext(context.WithValue(ctxWithUserID(userID), chi.RouteCtxKey, rctx))
			w := httptest.NewRecorder()

			handler.GetMonthlyDashboard(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			mockService.AssertExpectations(t)
		})
	}
}
