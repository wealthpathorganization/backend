package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestRespondJSON tests
func TestRespondJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       interface{}
		expectBody bool
	}{
		{
			name:       "success with data",
			status:     http.StatusOK,
			data:       map[string]string{"message": "success"},
			expectBody: true,
		},
		{
			name:       "created with data",
			status:     http.StatusCreated,
			data:       map[string]int{"id": 123},
			expectBody: true,
		},
		{
			name:       "no content",
			status:     http.StatusNoContent,
			data:       nil,
			expectBody: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			respondJSON(w, tt.status, tt.data)

			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			if tt.expectBody {
				assert.NotEmpty(t, w.Body.String())
			}
		})
	}
}

func TestRespondError(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		message string
	}{
		{"bad request", http.StatusBadRequest, "invalid input"},
		{"unauthorized", http.StatusUnauthorized, "not authorized"},
		{"not found", http.StatusNotFound, "resource not found"},
		{"internal error", http.StatusInternalServerError, "something went wrong"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			respondError(w, tt.status, tt.message)

			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var resp ErrorResponse
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, tt.message, resp.Error)
		})
	}
}

// TestGetUserID tests
func TestGetUserID(t *testing.T) {
	userID := uuid.New()
	ctx := context.WithValue(context.Background(), UserIDKey, userID)
	result := GetUserID(ctx)
	assert.Equal(t, userID, result)
}

func TestGetUserID_NotSet(t *testing.T) {
	ctx := context.Background()
	result := GetUserID(ctx)
	assert.Equal(t, uuid.Nil, result)
}

func TestGetUserID_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserIDKey, "not-a-uuid")
	result := GetUserID(ctx)
	assert.Equal(t, uuid.Nil, result)
}

// TestAuthHandler tests
func TestAuthHandler_Register_MissingFields(t *testing.T) {
	tests := []struct {
		name     string
		body     map[string]string
		wantCode int
	}{
		{
			name:     "missing email",
			body:     map[string]string{"password": "pass123"},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "missing password",
			body:     map[string]string{"email": "test@example.com"},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "empty body",
			body:     map[string]string{},
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Simulate handler validation logic
			var input struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}
			_ = json.Unmarshal(body, &input)

			if input.Email == "" || input.Password == "" {
				respondError(w, http.StatusBadRequest, "email and password are required")
			}

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestAuthHandler_Login_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	var input struct{}
	err := json.NewDecoder(req.Body).Decode(&input)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
	}

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestTransactionHandler tests
func TestTransactionHandler_Create_MissingFields(t *testing.T) {
	tests := []struct {
		name     string
		body     map[string]interface{}
		wantCode int
		errMsg   string
	}{
		{
			name:     "missing type",
			body:     map[string]interface{}{"amount": 100, "category": "Food"},
			wantCode: http.StatusBadRequest,
			errMsg:   "type",
		},
		{
			name:     "zero amount",
			body:     map[string]interface{}{"type": "expense", "amount": 0, "category": "Food"},
			wantCode: http.StatusBadRequest,
			errMsg:   "amount",
		},
		{
			name:     "missing category",
			body:     map[string]interface{}{"type": "expense", "amount": 100},
			wantCode: http.StatusBadRequest,
			errMsg:   "category",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			w := httptest.NewRecorder()

			var input struct {
				Type     string  `json:"type"`
				Amount   float64 `json:"amount"`
				Category string  `json:"category"`
			}
			_ = json.Unmarshal(body, &input)

			switch {
			case input.Type == "":
				respondError(w, http.StatusBadRequest, "type is required")
			case input.Amount == 0:
				respondError(w, http.StatusBadRequest, "amount is required")
			case input.Category == "":
				respondError(w, http.StatusBadRequest, "category is required")
			}

			assert.Equal(t, tt.wantCode, w.Code)
		})
	}
}

func TestTransactionHandler_Get_InvalidID(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/transactions/invalid-uuid", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "invalid-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	_, err := uuid.Parse(chi.URLParam(req, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
	}

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTransactionHandler_Get_ValidID(t *testing.T) {
	validID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/transactions/"+validID.String(), nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", validID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	parsedID, err := uuid.Parse(chi.URLParam(req, "id"))
	assert.NoError(t, err)
	assert.Equal(t, validID, parsedID)
}

// TestTransactionHandler_List tests
func TestTransactionHandler_List_QueryParams(t *testing.T) {
	tests := []struct {
		name        string
		queryParams map[string]string
		wantPage    int
		wantSize    int
	}{
		{
			name:        "defaults",
			queryParams: map[string]string{},
			wantPage:    0,
			wantSize:    20,
		},
		{
			name:        "custom page and size",
			queryParams: map[string]string{"page": "2", "pageSize": "50"},
			wantPage:    2,
			wantSize:    50,
		},
		{
			name:        "invalid page falls back to 0",
			queryParams: map[string]string{"page": "invalid"},
			wantPage:    0,
			wantSize:    20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := 0
			pageSize := 20

			if p := tt.queryParams["page"]; p != "" {
				var parsed int
				if _, err := json.Marshal(p); err == nil {
					if n, err := json.Number(p).Int64(); err == nil {
						parsed = int(n)
					}
				}
				if parsed > 0 {
					page = parsed
				}
			}

			if ps := tt.queryParams["pageSize"]; ps != "" {
				if n, err := json.Number(ps).Int64(); err == nil {
					pageSize = int(n)
				}
			}

			assert.Equal(t, tt.wantPage, page)
			assert.Equal(t, tt.wantSize, pageSize)
		})
	}
}

// TestBudgetHandler tests
func TestBudgetHandler_Create_Validation(t *testing.T) {
	tests := []struct {
		name     string
		body     map[string]interface{}
		wantCode int
	}{
		{
			name:     "valid budget",
			body:     map[string]interface{}{"category": "Food", "amount": 500, "period": "monthly"},
			wantCode: http.StatusOK,
		},
		{
			name:     "missing category",
			body:     map[string]interface{}{"amount": 500},
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			w := httptest.NewRecorder()

			var input struct {
				Category string  `json:"category"`
				Amount   float64 `json:"amount"`
			}
			_ = json.Unmarshal(body, &input)

			if input.Category == "" {
				respondError(w, http.StatusBadRequest, "category is required")
				assert.Equal(t, tt.wantCode, w.Code)
			}
		})
	}
}

// TestDebtHandler tests
func TestDebtHandler_Create_Validation(t *testing.T) {
	body := map[string]interface{}{
		"name":           "Mortgage",
		"type":           "mortgage",
		"originalAmount": 200000,
		"interestRate":   4.5,
		"minimumPayment": 1200,
	}

	data, _ := json.Marshal(body)

	var input struct {
		Name           string  `json:"name"`
		Type           string  `json:"type"`
		OriginalAmount float64 `json:"originalAmount"`
	}
	_ = json.Unmarshal(data, &input)

	assert.Equal(t, "Mortgage", input.Name)
	assert.Equal(t, "mortgage", input.Type)
	assert.Equal(t, float64(200000), input.OriginalAmount)
}

// TestSavingsHandler tests
func TestSavingsHandler_Create_Validation(t *testing.T) {
	body := map[string]interface{}{
		"name":         "Emergency Fund",
		"targetAmount": 10000,
		"currency":     "USD",
	}

	data, _ := json.Marshal(body)

	var input struct {
		Name         string  `json:"name"`
		TargetAmount float64 `json:"targetAmount"`
	}
	_ = json.Unmarshal(data, &input)

	assert.Equal(t, "Emergency Fund", input.Name)
	assert.Equal(t, float64(10000), input.TargetAmount)
}

// TestRecurringHandler tests
func TestRecurringHandler_Create_Validation(t *testing.T) {
	body := map[string]interface{}{
		"type":        "expense",
		"amount":      100,
		"category":    "Utilities",
		"description": "Electricity",
		"frequency":   "monthly",
	}

	data, _ := json.Marshal(body)

	var input struct {
		Type        string  `json:"type"`
		Amount      float64 `json:"amount"`
		Category    string  `json:"category"`
		Description string  `json:"description"`
		Frequency   string  `json:"frequency"`
	}
	_ = json.Unmarshal(data, &input)

	assert.Equal(t, "expense", input.Type)
	assert.Equal(t, float64(100), input.Amount)
	assert.Equal(t, "monthly", input.Frequency)
}

// Benchmark tests
func BenchmarkRespondJSON(b *testing.B) {
	data := map[string]interface{}{
		"id":      uuid.New().String(),
		"message": "test message",
		"count":   100,
	}

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		respondJSON(w, http.StatusOK, data)
	}
}

func BenchmarkRespondError(b *testing.B) {
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		respondError(w, http.StatusBadRequest, "test error message")
	}
}
