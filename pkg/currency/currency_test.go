package currency

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupportedCurrencies(t *testing.T) {
	currencies := SupportedCurrencies()
	assert.Len(t, currencies, 10)
	assert.Contains(t, currencies, USD)
	assert.Contains(t, currencies, EUR)
	assert.Contains(t, currencies, VND)
}

func TestSupportedCurrencyCodes(t *testing.T) {
	codes := SupportedCurrencyCodes()
	assert.Len(t, codes, 10)
	assert.Contains(t, codes, "USD")
	assert.Contains(t, codes, "EUR")
	assert.Contains(t, codes, "VND")
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		code  string
		valid bool
	}{
		{"USD", true},
		{"EUR", true},
		{"VND", true},
		{"INVALID", false},
		{"", false},
		{"usd", false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			assert.Equal(t, tt.valid, IsValid(tt.code))
		})
	}
}

func TestGetInfo(t *testing.T) {
	t.Run("valid currency", func(t *testing.T) {
		info, ok := GetInfo(USD)
		assert.True(t, ok)
		assert.Equal(t, USD, info.Code)
		assert.Equal(t, "US Dollar", info.Name)
		assert.Equal(t, "$", info.Symbol)
		assert.Equal(t, 2, info.DecimalPlaces)
		assert.True(t, info.SymbolBefore)
	})

	t.Run("VND currency", func(t *testing.T) {
		info, ok := GetInfo(VND)
		assert.True(t, ok)
		assert.Equal(t, VND, info.Code)
		assert.Equal(t, "Vietnamese Dong", info.Name)
		assert.Equal(t, "₫", info.Symbol)
		assert.Equal(t, 0, info.DecimalPlaces)
		assert.False(t, info.SymbolBefore)
	})

	t.Run("invalid currency", func(t *testing.T) {
		_, ok := GetInfo(Currency("INVALID"))
		assert.False(t, ok)
	})
}

func TestNewMoney(t *testing.T) {
	t.Run("with currency", func(t *testing.T) {
		m := NewMoney(decimal.NewFromFloat(100.50), USD)
		assert.Equal(t, "100.5", m.Amount.String())
		assert.Equal(t, USD, m.Currency)
	})

	t.Run("with empty currency defaults to USD", func(t *testing.T) {
		m := NewMoney(decimal.NewFromFloat(100), "")
		assert.Equal(t, DefaultCurrency, m.Currency)
	})
}

func TestNewMoneyFromFloat(t *testing.T) {
	m := NewMoneyFromFloat(99.99, EUR)
	assert.Equal(t, "99.99", m.Amount.String())
	assert.Equal(t, EUR, m.Currency)
}

func TestNewMoneyFromString(t *testing.T) {
	t.Run("valid amount", func(t *testing.T) {
		m, err := NewMoneyFromString("123.45", GBP)
		require.NoError(t, err)
		assert.Equal(t, "123.45", m.Amount.String())
		assert.Equal(t, GBP, m.Currency)
	})

	t.Run("invalid amount", func(t *testing.T) {
		_, err := NewMoneyFromString("not-a-number", USD)
		assert.Error(t, err)
	})
}

func TestZero(t *testing.T) {
	m := Zero(VND)
	assert.True(t, m.Amount.IsZero())
	assert.Equal(t, VND, m.Currency)
}

func TestMoneyAdd(t *testing.T) {
	t.Run("same currency", func(t *testing.T) {
		m1 := NewMoneyFromFloat(100.50, USD)
		m2 := NewMoneyFromFloat(50.25, USD)
		result, err := m1.Add(m2)
		require.NoError(t, err)
		assert.Equal(t, "150.75", result.Amount.String())
		assert.Equal(t, USD, result.Currency)
	})

	t.Run("different currencies", func(t *testing.T) {
		m1 := NewMoneyFromFloat(100, USD)
		m2 := NewMoneyFromFloat(100, EUR)
		_, err := m1.Add(m2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "currency mismatch")
	})
}

func TestMoneySub(t *testing.T) {
	t.Run("same currency", func(t *testing.T) {
		m1 := NewMoneyFromFloat(100.50, USD)
		m2 := NewMoneyFromFloat(50.25, USD)
		result, err := m1.Sub(m2)
		require.NoError(t, err)
		assert.Equal(t, "50.25", result.Amount.String())
	})

	t.Run("different currencies", func(t *testing.T) {
		m1 := NewMoneyFromFloat(100, USD)
		m2 := NewMoneyFromFloat(100, EUR)
		_, err := m1.Sub(m2)
		assert.Error(t, err)
	})
}

func TestMoneyMultiply(t *testing.T) {
	m := NewMoneyFromFloat(100, USD)
	result := m.Multiply(decimal.NewFromFloat(1.5))
	assert.Equal(t, "150", result.Amount.String())
	assert.Equal(t, USD, result.Currency)
}

func TestMoneyIsZero(t *testing.T) {
	assert.True(t, Zero(USD).IsZero())
	assert.False(t, NewMoneyFromFloat(0.01, USD).IsZero())
}

func TestMoneyIsPositive(t *testing.T) {
	assert.True(t, NewMoneyFromFloat(100, USD).IsPositive())
	assert.False(t, NewMoneyFromFloat(-100, USD).IsPositive())
	assert.False(t, Zero(USD).IsPositive())
}

func TestMoneyIsNegative(t *testing.T) {
	assert.True(t, NewMoneyFromFloat(-100, USD).IsNegative())
	assert.False(t, NewMoneyFromFloat(100, USD).IsNegative())
	assert.False(t, Zero(USD).IsNegative())
}

func TestMoneyAbs(t *testing.T) {
	m := NewMoneyFromFloat(-100.50, USD)
	result := m.Abs()
	assert.Equal(t, "100.5", result.Amount.String())
	assert.True(t, result.IsPositive())
}

func TestMoneyRound(t *testing.T) {
	t.Run("USD 2 decimals", func(t *testing.T) {
		m := NewMoneyFromFloat(100.556, USD)
		result := m.Round()
		assert.Equal(t, "100.56", result.Amount.String())
	})

	t.Run("JPY 0 decimals", func(t *testing.T) {
		m := NewMoneyFromFloat(100.6, JPY)
		result := m.Round()
		assert.Equal(t, "101", result.Amount.String())
	})

	t.Run("VND 0 decimals", func(t *testing.T) {
		m := NewMoneyFromFloat(25000.5, VND)
		result := m.Round()
		assert.Equal(t, "25001", result.Amount.String())
	})

	t.Run("invalid currency defaults to USD", func(t *testing.T) {
		m := Money{Amount: decimal.NewFromFloat(100.556), Currency: Currency("INVALID")}
		result := m.Round()
		assert.Equal(t, "100.56", result.Amount.String())
	})
}

func TestMoneyFormat(t *testing.T) {
	tests := []struct {
		name     string
		money    Money
		expected string
	}{
		{"USD", NewMoneyFromFloat(1234.56, USD), "$1234.56"},
		{"EUR", NewMoneyFromFloat(1234.56, EUR), "1234.56€"},
		{"JPY", NewMoneyFromFloat(1234, JPY), "¥1234"},
		{"VND", NewMoneyFromFloat(25000, VND), "25000₫"},
		{"GBP", NewMoneyFromFloat(99.99, GBP), "£99.99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.money.Format())
		})
	}

	t.Run("invalid currency", func(t *testing.T) {
		m := Money{Amount: decimal.NewFromFloat(100.50), Currency: Currency("INVALID")}
		result := m.Format()
		assert.Contains(t, result, "100.50")
		assert.Contains(t, result, "INVALID")
	})
}

func TestMoneyString(t *testing.T) {
	t.Run("USD", func(t *testing.T) {
		m := NewMoneyFromFloat(100.556, USD)
		assert.Equal(t, "100.56", m.String())
	})

	t.Run("JPY", func(t *testing.T) {
		m := NewMoneyFromFloat(100.6, JPY)
		assert.Equal(t, "101", m.String())
	})

	t.Run("invalid currency", func(t *testing.T) {
		m := Money{Amount: decimal.NewFromFloat(100.50), Currency: Currency("INVALID")}
		assert.Equal(t, "100.5", m.String())
	})
}
