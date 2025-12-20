package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wealthpath/backend/internal/model"
)

// MockAITransactionService for AI service testing
type MockAITransactionService struct {
	mock.Mock
}

func (m *MockAITransactionService) Create(ctx context.Context, userID uuid.UUID, input CreateTransactionInput) (*model.Transaction, error) {
	args := m.Called(ctx, userID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Transaction), args.Error(1)
}

// MockAIBudgetService for AI service testing
type MockAIBudgetService struct {
	mock.Mock
}

func (m *MockAIBudgetService) Create(ctx context.Context, userID uuid.UUID, input CreateBudgetInput) (*model.Budget, error) {
	args := m.Called(ctx, userID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Budget), args.Error(1)
}

// MockAISavingsService for AI service testing
type MockAISavingsService struct {
	mock.Mock
}

func (m *MockAISavingsService) Create(ctx context.Context, userID uuid.UUID, input CreateSavingsGoalInput) (*model.SavingsGoal, error) {
	args := m.Called(ctx, userID, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.SavingsGoal), args.Error(1)
}

func TestNewAIService(t *testing.T) {
	t.Parallel()

	// Test that NewAIService creates a valid service
	service := NewAIService(nil, nil, nil)
	assert.NotNil(t, service)
}

func TestAIService_Chat_NoAPIKey(t *testing.T) {
	t.Parallel()

	// Ensure no API key is set
	_ = os.Unsetenv("OPENAI_API_KEY")

	service := NewAIService(nil, nil, nil)
	userID := uuid.New()

	resp, err := service.Chat(context.Background(), userID, ChatRequest{Message: "test"})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Contains(t, resp.Message, "AI features are not configured")
}

func TestChatRequest_Struct(t *testing.T) {
	t.Parallel()

	req := ChatRequest{
		Message: "Spent $50 on lunch",
	}

	assert.Equal(t, "Spent $50 on lunch", req.Message)
}

func TestChatResponse_Struct(t *testing.T) {
	t.Parallel()

	resp := ChatResponse{
		Message: "Transaction recorded",
		Action: &ActionResult{
			Type: "transaction",
			Data: map[string]interface{}{"id": "123"},
		},
	}

	assert.Equal(t, "Transaction recorded", resp.Message)
	assert.NotNil(t, resp.Action)
	assert.Equal(t, "transaction", resp.Action.Type)
}

func TestActionResult_Struct(t *testing.T) {
	t.Parallel()

	result := ActionResult{
		Type: "budget",
		Data: map[string]interface{}{"category": "Food"},
	}

	assert.Equal(t, "budget", result.Type)
	assert.NotNil(t, result.Data)
}

func TestParsedIntent_Struct(t *testing.T) {
	t.Parallel()

	intent := ParsedIntent{
		Action:      "add_transaction",
		Type:        "expense",
		Amount:      50.0,
		Category:    "Food & Dining",
		Description: "Lunch",
		Response:    "Recorded $50 expense for lunch",
	}

	assert.Equal(t, "add_transaction", intent.Action)
	assert.Equal(t, "expense", intent.Type)
	assert.Equal(t, 50.0, intent.Amount)
	assert.Equal(t, "Food & Dining", intent.Category)
	assert.Equal(t, "Lunch", intent.Description)
}

func TestOpenAIRequest_Struct(t *testing.T) {
	t.Parallel()

	req := OpenAIRequest{
		Model: "gpt-4o-mini",
		Messages: []OpenAIMessage{
			{Role: "system", Content: "You are a helpful assistant"},
			{Role: "user", Content: "Hello"},
		},
	}

	assert.Equal(t, "gpt-4o-mini", req.Model)
	assert.Len(t, req.Messages, 2)
	assert.Equal(t, "system", req.Messages[0].Role)
}

func TestOpenAIMessage_Struct(t *testing.T) {
	t.Parallel()

	msg := OpenAIMessage{
		Role:    "user",
		Content: "Test message",
	}

	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "Test message", msg.Content)
}

// TestAIService_ExecuteAction tests the executeAction method
func TestAIService_ExecuteAction_Query(t *testing.T) {
	t.Parallel()

	service := NewAIService(nil, nil, nil)
	userID := uuid.New()

	intent := &ParsedIntent{
		Action:   "query",
		Response: "Here's your summary",
	}

	result, err := service.executeAction(context.Background(), userID, intent)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestAIService_ExecuteAction_Unknown(t *testing.T) {
	t.Parallel()

	service := NewAIService(nil, nil, nil)
	userID := uuid.New()

	intent := &ParsedIntent{
		Action:   "unknown",
		Response: "I didn't understand",
	}

	result, err := service.executeAction(context.Background(), userID, intent)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestAIService_ExecuteAction_InvalidAction(t *testing.T) {
	t.Parallel()

	service := NewAIService(nil, nil, nil)
	userID := uuid.New()

	intent := &ParsedIntent{
		Action: "invalid_action",
	}

	result, err := service.executeAction(context.Background(), userID, intent)

	assert.NoError(t, err)
	assert.Nil(t, result)
}

// Test AddTransaction helper
func TestAIService_AddTransaction_ExpenseType(t *testing.T) {
	t.Parallel()

	mockTxService := new(MockTransactionRepo)
	txService := NewTransactionService(mockTxService)

	mockTxService.On("Create", mock.Anything, mock.AnythingOfType("*model.Transaction")).Return(nil)

	service := NewAIService(txService, nil, nil)
	userID := uuid.New()

	intent := &ParsedIntent{
		Action:      "add_transaction",
		Type:        "expense",
		Amount:      50,
		Category:    "Food & Dining",
		Description: "Lunch",
	}

	result, err := service.addTransaction(context.Background(), userID, intent)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "transaction", result.Type)
}

func TestAIService_AddTransaction_IncomeType(t *testing.T) {
	t.Parallel()

	mockTxService := new(MockTransactionRepo)
	txService := NewTransactionService(mockTxService)

	mockTxService.On("Create", mock.Anything, mock.AnythingOfType("*model.Transaction")).Return(nil)

	service := NewAIService(txService, nil, nil)
	userID := uuid.New()

	intent := &ParsedIntent{
		Action:      "add_transaction",
		Type:        "income",
		Amount:      5000,
		Category:    "Salary",
		Description: "Monthly salary",
	}

	result, err := service.addTransaction(context.Background(), userID, intent)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "transaction", result.Type)
}

// Test AddBudget helper
func TestAIService_AddBudget(t *testing.T) {
	t.Parallel()

	mockBudgetRepo := new(MockBudgetRepo)
	budgetService := NewBudgetService(mockBudgetRepo)

	mockBudgetRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Budget")).Return(nil)

	service := NewAIService(nil, budgetService, nil)
	userID := uuid.New()

	intent := &ParsedIntent{
		Action:   "add_budget",
		Amount:   500,
		Category: "Food & Dining",
		Period:   "monthly",
	}

	result, err := service.addBudget(context.Background(), userID, intent)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "budget", result.Type)
}

func TestAIService_AddBudget_DefaultPeriod(t *testing.T) {
	t.Parallel()

	mockBudgetRepo := new(MockBudgetRepo)
	budgetService := NewBudgetService(mockBudgetRepo)

	mockBudgetRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Budget")).Return(nil)

	service := NewAIService(nil, budgetService, nil)
	userID := uuid.New()

	intent := &ParsedIntent{
		Action:   "add_budget",
		Amount:   500,
		Category: "Food & Dining",
		Period:   "", // Empty period should default to "monthly"
	}

	result, err := service.addBudget(context.Background(), userID, intent)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "budget", result.Type)
}

// Test AddSavingsGoal helper
func TestAIService_AddSavingsGoal(t *testing.T) {
	t.Parallel()

	mockSavingsRepo := new(MockSavingsGoalRepo)
	savingsService := NewSavingsGoalService(mockSavingsRepo)

	mockSavingsRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.SavingsGoal")).Return(nil)

	service := NewAIService(nil, nil, savingsService)
	userID := uuid.New()

	intent := &ParsedIntent{
		Action:     "add_savings_goal",
		Amount:     10000,
		Name:       "New Car",
		TargetDate: time.Now().AddDate(1, 0, 0).Format("2006-01-02"),
	}

	result, err := service.addSavingsGoal(context.Background(), userID, intent)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "savings_goal", result.Type)
}

func TestAIService_AddSavingsGoal_NoTargetDate(t *testing.T) {
	t.Parallel()

	mockSavingsRepo := new(MockSavingsGoalRepo)
	savingsService := NewSavingsGoalService(mockSavingsRepo)

	mockSavingsRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.SavingsGoal")).Return(nil)

	service := NewAIService(nil, nil, savingsService)
	userID := uuid.New()

	intent := &ParsedIntent{
		Action:     "add_savings_goal",
		Amount:     10000,
		Name:       "Emergency Fund",
		TargetDate: "", // No target date
	}

	result, err := service.addSavingsGoal(context.Background(), userID, intent)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "savings_goal", result.Type)
}

func TestAIService_AddSavingsGoal_InvalidTargetDate(t *testing.T) {
	t.Parallel()

	mockSavingsRepo := new(MockSavingsGoalRepo)
	savingsService := NewSavingsGoalService(mockSavingsRepo)

	mockSavingsRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.SavingsGoal")).Return(nil)

	service := NewAIService(nil, nil, savingsService)
	userID := uuid.New()

	intent := &ParsedIntent{
		Action:     "add_savings_goal",
		Amount:     10000,
		Name:       "Vacation",
		TargetDate: "invalid-date", // Invalid date format
	}

	result, err := service.addSavingsGoal(context.Background(), userID, intent)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "savings_goal", result.Type)
}

// Test system prompt constant
func TestSystemPrompt(t *testing.T) {
	t.Parallel()

	assert.NotEmpty(t, systemPrompt)
	assert.Contains(t, systemPrompt, "financial assistant")
	assert.Contains(t, systemPrompt, "JSON")
}

// Test decimal conversion
func TestDecimalConversion(t *testing.T) {
	t.Parallel()

	amount := 50.0
	dec := decimal.NewFromFloat(amount)

	assert.True(t, dec.Equal(decimal.NewFromFloat(50)))
}
