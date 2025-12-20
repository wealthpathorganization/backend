package service

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/wealthpath/backend/internal/model"
	"github.com/wealthpath/backend/internal/repository"
)

// MockInterestRateRepository is a mock implementation of InterestRateRepository
type MockInterestRateRepository struct {
	mock.Mock
}

func (m *MockInterestRateRepository) List(ctx context.Context, productType string, termMonths *int, bankCode string) ([]model.InterestRate, error) {
	args := m.Called(ctx, productType, termMonths, bankCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.InterestRate), args.Error(1)
}

func (m *MockInterestRateRepository) GetBestRates(ctx context.Context, productType string, termMonths int, limit int) ([]model.InterestRate, error) {
	args := m.Called(ctx, productType, termMonths, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]model.InterestRate), args.Error(1)
}

func (m *MockInterestRateRepository) Upsert(ctx context.Context, rate *model.InterestRate) error {
	args := m.Called(ctx, rate)
	return args.Error(0)
}

func (m *MockInterestRateRepository) DeleteOldRates(ctx context.Context, daysOld int) error {
	args := m.Called(ctx, daysOld)
	return args.Error(0)
}

func (m *MockInterestRateRepository) GetHistory(ctx context.Context, bankCode, productType string, termMonths, days int) ([]repository.RateHistoryEntry, error) {
	args := m.Called(ctx, bankCode, productType, termMonths, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.RateHistoryEntry), args.Error(1)
}

func TestInterestRateService_ListRates(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInterestRateRepository)
	service := &InterestRateService{repo: mockRepo}

	expectedRates := []model.InterestRate{
		{ID: 1, BankCode: "vcb", BankName: "Vietcombank", ProductType: "deposit", TermMonths: 12, Rate: decimal.NewFromFloat(4.7)},
		{ID: 2, BankCode: "tcb", BankName: "Techcombank", ProductType: "deposit", TermMonths: 12, Rate: decimal.NewFromFloat(4.85)},
	}

	termMonths := 12
	mockRepo.On("List", ctx, "deposit", &termMonths, "").Return(expectedRates, nil)

	rates, err := service.ListRates(ctx, "deposit", &termMonths, "")

	assert.NoError(t, err)
	assert.Len(t, rates, 2)
	assert.Equal(t, "vcb", rates[0].BankCode)
	mockRepo.AssertExpectations(t)
}

func TestInterestRateService_GetBestRates(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInterestRateRepository)
	service := &InterestRateService{repo: mockRepo}

	expectedRates := []model.InterestRate{
		{ID: 1, BankCode: "vpbank", BankName: "VPBank", ProductType: "deposit", TermMonths: 12, Rate: decimal.NewFromFloat(5.3)},
		{ID: 2, BankCode: "tpbank", BankName: "TPBank", ProductType: "deposit", TermMonths: 12, Rate: decimal.NewFromFloat(5.3)},
		{ID: 3, BankCode: "mb", BankName: "MB Bank", ProductType: "deposit", TermMonths: 12, Rate: decimal.NewFromFloat(5.0)},
	}

	mockRepo.On("GetBestRates", ctx, "deposit", 12, 5).Return(expectedRates, nil)

	rates, err := service.GetBestRates(ctx, "deposit", 12, 5)

	assert.NoError(t, err)
	assert.Len(t, rates, 3)
	assert.Equal(t, "vpbank", rates[0].BankCode)
	mockRepo.AssertExpectations(t)
}

func TestInterestRateService_CompareRates(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInterestRateRepository)
	service := &InterestRateService{repo: mockRepo}

	expectedRates := []model.InterestRate{
		{ID: 1, BankCode: "vcb", BankName: "Vietcombank", ProductType: "deposit", TermMonths: 6, Rate: decimal.NewFromFloat(3.0)},
		{ID: 2, BankCode: "tcb", BankName: "Techcombank", ProductType: "deposit", TermMonths: 6, Rate: decimal.NewFromFloat(3.55)},
	}

	termMonths := 6
	mockRepo.On("List", ctx, "deposit", &termMonths, "").Return(expectedRates, nil)

	rates, err := service.CompareRates(ctx, "deposit", 6)

	assert.NoError(t, err)
	assert.Len(t, rates, 2)
	mockRepo.AssertExpectations(t)
}

func TestInterestRateService_GetBanks(t *testing.T) {
	service := &InterestRateService{}

	banks := service.GetBanks()

	assert.NotEmpty(t, banks)
	assert.GreaterOrEqual(t, len(banks), 8) // At least 8 banks configured

	// Verify some known banks exist
	bankCodes := make(map[string]bool)
	for _, bank := range banks {
		bankCodes[bank.Code] = true
	}
	assert.True(t, bankCodes["vcb"], "Vietcombank should be in the list")
	assert.True(t, bankCodes["tcb"], "Techcombank should be in the list")
	assert.True(t, bankCodes["mb"], "MB Bank should be in the list")
}

func TestInterestRateService_UpsertRate(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInterestRateRepository)
	service := &InterestRateService{repo: mockRepo}

	rate := &model.InterestRate{
		BankCode:      "vcb",
		BankName:      "Vietcombank",
		ProductType:   "deposit",
		TermMonths:    12,
		Rate:          decimal.NewFromFloat(4.7),
		Currency:      "VND",
		EffectiveDate: time.Now(),
		ScrapedAt:     time.Now(),
	}

	mockRepo.On("Upsert", ctx, rate).Return(nil)

	err := service.UpsertRate(ctx, rate)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestInterestRateService_BulkUpsertRates(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInterestRateRepository)
	service := &InterestRateService{repo: mockRepo}

	rates := []model.InterestRate{
		{BankCode: "vcb", BankName: "Vietcombank", ProductType: "deposit", TermMonths: 12, Rate: decimal.NewFromFloat(4.7)},
		{BankCode: "tcb", BankName: "Techcombank", ProductType: "deposit", TermMonths: 12, Rate: decimal.NewFromFloat(4.85)},
	}

	mockRepo.On("Upsert", ctx, mock.AnythingOfType("*model.InterestRate")).Return(nil).Times(2)

	err := service.BulkUpsertRates(ctx, rates)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestInterestRateService_GetRateHistory(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInterestRateRepository)
	service := &InterestRateService{repo: mockRepo}

	expectedHistory := []repository.RateHistoryEntry{
		{BankCode: "vcb", ProductType: "deposit", TermMonths: 12, Rate: decimal.NewFromFloat(4.5), RecordedDate: time.Now().AddDate(0, 0, -30)},
		{BankCode: "vcb", ProductType: "deposit", TermMonths: 12, Rate: decimal.NewFromFloat(4.6), RecordedDate: time.Now().AddDate(0, 0, -15)},
		{BankCode: "vcb", ProductType: "deposit", TermMonths: 12, Rate: decimal.NewFromFloat(4.7), RecordedDate: time.Now()},
	}

	mockRepo.On("GetHistory", ctx, "vcb", "deposit", 12, 90).Return(expectedHistory, nil)

	history, err := service.GetRateHistory(ctx, "vcb", "deposit", 12, 90)

	assert.NoError(t, err)
	assert.Len(t, history, 3)
	mockRepo.AssertExpectations(t)
}

func TestInterestRateService_SeedDefaultRates(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockInterestRateRepository)
	service := &InterestRateService{repo: mockRepo}

	// SeedDefaultRates will call Upsert for each rate
	mockRepo.On("Upsert", ctx, mock.AnythingOfType("*model.InterestRate")).Return(nil)

	err := service.SeedDefaultRates(ctx)

	assert.NoError(t, err)
	// Should have called Upsert multiple times (once for each seeded rate)
	assert.GreaterOrEqual(t, len(mockRepo.Calls), 20) // At least 20 rates seeded
}
