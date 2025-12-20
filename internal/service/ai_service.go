package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/wealthpath/backend/internal/model"
	"github.com/wealthpath/backend/pkg/datetime"
)

// AIService handles natural language processing for financial actions.
// It uses OpenAI to parse user messages and execute corresponding operations.
type AIService struct {
	transactionService *TransactionService
	budgetService      *BudgetService
	savingsService     *SavingsGoalService
}

// NewAIService creates a new AIService with the required service dependencies.
func NewAIService(ts *TransactionService, bs *BudgetService, ss *SavingsGoalService) *AIService {
	return &AIService{
		transactionService: ts,
		budgetService:      bs,
		savingsService:     ss,
	}
}

// ChatRequest represents an incoming chat message from the user.
type ChatRequest struct {
	Message string `json:"message"`
}

// ChatResponse contains the AI response and any executed action.
type ChatResponse struct {
	Message string        `json:"message"`
	Action  *ActionResult `json:"action,omitempty"`
}

// ActionResult represents the result of an executed financial action.
type ActionResult struct {
	Type string      `json:"type"` // transaction, budget, savings_goal
	Data interface{} `json:"data"`
}

// OpenAIRequest represents the request payload for OpenAI API.
type OpenAIRequest struct {
	Model    string          `json:"model"`
	Messages []OpenAIMessage `json:"messages"`
}

// OpenAIMessage represents a single message in the OpenAI conversation.
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents the response from OpenAI API.
type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// ParsedIntent represents the extracted intent from a user message.
type ParsedIntent struct {
	Action      string  `json:"action"` // add_transaction, add_budget, add_savings_goal, query, unknown
	Type        string  `json:"type"`   // income/expense for transactions
	Amount      float64 `json:"amount"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Name        string  `json:"name"`        // for savings goals
	TargetDate  string  `json:"target_date"` // for savings goals
	Period      string  `json:"period"`      // for budgets: monthly, weekly, yearly
	Response    string  `json:"response"`    // AI's friendly response
}

const systemPrompt = `You are a financial assistant for WealthPath app. Parse user messages and extract financial actions.

ALWAYS respond with valid JSON in this exact format:
{
  "action": "add_transaction" | "add_budget" | "add_savings_goal" | "query" | "unknown",
  "type": "income" | "expense" (for transactions only),
  "amount": number,
  "category": "Food & Dining" | "Transportation" | "Shopping" | "Entertainment" | "Housing" | "Utilities" | "Healthcare" | "Salary" | "Freelance" | "Other",
  "description": "brief description",
  "name": "goal name (for savings)",
  "target_date": "YYYY-MM-DD (for savings)",
  "period": "monthly" | "weekly" | "yearly" (for budgets)",
  "response": "friendly confirmation message"
}

Examples:
- "Spent $50 on lunch" → action: add_transaction, type: expense, amount: 50, category: Food & Dining
- "Got paid $5000 salary" → action: add_transaction, type: income, amount: 5000, category: Salary
- "Set $500 budget for food this month" → action: add_budget, amount: 500, category: Food & Dining, period: monthly
- "Save $10000 for a car by December" → action: add_savings_goal, amount: 10000, name: Car, target_date: 2025-12-31

If you can't understand, set action to "unknown" and provide a helpful response.`

// Chat processes a user message, extracts intent using AI, and executes the action.
// Returns a friendly response and the result of any executed action.
func (s *AIService) Chat(ctx context.Context, userID uuid.UUID, req ChatRequest) (*ChatResponse, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return &ChatResponse{
			Message: "AI features are not configured. Please add your OpenAI API key.",
		}, nil
	}

	intent, err := s.parseIntent(apiKey, req.Message)
	if err != nil {
		return &ChatResponse{
			Message: "Sorry, I couldn't understand that. Try something like 'Spent $50 on groceries' or 'Set $500 monthly budget for food'.",
		}, nil
	}

	result, err := s.executeAction(ctx, userID, intent)
	if err != nil {
		return &ChatResponse{
			Message: fmt.Sprintf("I understood your request, but couldn't complete it: %s", err.Error()),
		}, nil
	}

	return &ChatResponse{
		Message: intent.Response,
		Action:  result,
	}, nil
}

// parseIntent calls OpenAI to extract structured intent from natural language.
func (s *AIService) parseIntent(apiKey, message string) (*ParsedIntent, error) {
	reqBody := OpenAIRequest{
		Model: "gpt-4o-mini",
		Messages: []OpenAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: message},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling OpenAI request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating OpenAI request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling OpenAI API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading OpenAI response: %w", err)
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, fmt.Errorf("parsing OpenAI response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI")
	}

	var intent ParsedIntent
	content := openAIResp.Choices[0].Message.Content
	if err := json.Unmarshal([]byte(content), &intent); err != nil {
		return nil, fmt.Errorf("parsing AI intent from response: %w", err)
	}

	return &intent, nil
}

// executeAction dispatches the parsed intent to the appropriate handler.
func (s *AIService) executeAction(ctx context.Context, userID uuid.UUID, intent *ParsedIntent) (*ActionResult, error) {
	switch intent.Action {
	case "add_transaction":
		return s.addTransaction(ctx, userID, intent)
	case "add_budget":
		return s.addBudget(ctx, userID, intent)
	case "add_savings_goal":
		return s.addSavingsGoal(ctx, userID, intent)
	case "query", "unknown":
		return nil, nil
	default:
		return nil, nil
	}
}

// addTransaction creates a new transaction from the parsed intent.
func (s *AIService) addTransaction(ctx context.Context, userID uuid.UUID, intent *ParsedIntent) (*ActionResult, error) {
	txType := model.TransactionTypeExpense
	if intent.Type == "income" {
		txType = model.TransactionTypeIncome
	}

	input := CreateTransactionInput{
		Type:        txType,
		Amount:      decimal.NewFromFloat(intent.Amount),
		Category:    intent.Category,
		Description: intent.Description,
		Date:        datetime.Today(),
	}

	tx, err := s.transactionService.Create(ctx, userID, input)
	if err != nil {
		return nil, fmt.Errorf("creating transaction from AI: %w", err)
	}

	return &ActionResult{
		Type: "transaction",
		Data: tx,
	}, nil
}

// addBudget creates a new budget from the parsed intent.
func (s *AIService) addBudget(ctx context.Context, userID uuid.UUID, intent *ParsedIntent) (*ActionResult, error) {
	period := "monthly"
	if intent.Period != "" {
		period = intent.Period
	}

	input := CreateBudgetInput{
		Category:  intent.Category,
		Amount:    decimal.NewFromFloat(intent.Amount),
		Period:    period,
		StartDate: time.Now(),
	}

	budget, err := s.budgetService.Create(ctx, userID, input)
	if err != nil {
		return nil, fmt.Errorf("creating budget from AI: %w", err)
	}

	return &ActionResult{
		Type: "budget",
		Data: budget,
	}, nil
}

// addSavingsGoal creates a new savings goal from the parsed intent.
func (s *AIService) addSavingsGoal(ctx context.Context, userID uuid.UUID, intent *ParsedIntent) (*ActionResult, error) {
	var targetDate *time.Time
	if intent.TargetDate != "" {
		t, err := time.Parse("2006-01-02", intent.TargetDate)
		if err == nil {
			targetDate = &t
		}
	}

	input := CreateSavingsGoalInput{
		Name:         intent.Name,
		TargetAmount: decimal.NewFromFloat(intent.Amount),
		TargetDate:   targetDate,
	}

	goal, err := s.savingsService.Create(ctx, userID, input)
	if err != nil {
		return nil, fmt.Errorf("creating savings goal from AI: %w", err)
	}

	return &ActionResult{
		Type: "savings_goal",
		Data: goal,
	}, nil
}
