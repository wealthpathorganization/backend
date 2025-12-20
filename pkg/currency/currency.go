// Package currency provides standardized currency handling across the application.
// All monetary amounts are stored as decimal.Decimal to avoid floating-point errors.
package currency

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Currency represents an ISO 4217 currency code.
type Currency string

// Supported currencies.
const (
	USD Currency = "USD" // US Dollar
	EUR Currency = "EUR" // Euro
	GBP Currency = "GBP" // British Pound
	JPY Currency = "JPY" // Japanese Yen
	CNY Currency = "CNY" // Chinese Yuan
	VND Currency = "VND" // Vietnamese Dong
	CAD Currency = "CAD" // Canadian Dollar
	AUD Currency = "AUD" // Australian Dollar
	CHF Currency = "CHF" // Swiss Franc
	SGD Currency = "SGD" // Singapore Dollar
)

// DefaultCurrency is the default currency when none is specified.
const DefaultCurrency = USD

// CurrencyInfo contains metadata about a currency.
type CurrencyInfo struct {
	Code          Currency
	Name          string
	Symbol        string
	DecimalPlaces int    // Number of decimal places (e.g., 2 for USD, 0 for JPY)
	SymbolBefore  bool   // Whether symbol appears before amount
	ThousandsSep  string // Thousands separator
	DecimalSep    string // Decimal separator
}

// currencies maps currency codes to their info.
var currencies = map[Currency]CurrencyInfo{
	USD: {Code: USD, Name: "US Dollar", Symbol: "$", DecimalPlaces: 2, SymbolBefore: true, ThousandsSep: ",", DecimalSep: "."},
	EUR: {Code: EUR, Name: "Euro", Symbol: "€", DecimalPlaces: 2, SymbolBefore: false, ThousandsSep: ".", DecimalSep: ","},
	GBP: {Code: GBP, Name: "British Pound", Symbol: "£", DecimalPlaces: 2, SymbolBefore: true, ThousandsSep: ",", DecimalSep: "."},
	JPY: {Code: JPY, Name: "Japanese Yen", Symbol: "¥", DecimalPlaces: 0, SymbolBefore: true, ThousandsSep: ",", DecimalSep: "."},
	CNY: {Code: CNY, Name: "Chinese Yuan", Symbol: "¥", DecimalPlaces: 2, SymbolBefore: true, ThousandsSep: ",", DecimalSep: "."},
	VND: {Code: VND, Name: "Vietnamese Dong", Symbol: "₫", DecimalPlaces: 0, SymbolBefore: false, ThousandsSep: ".", DecimalSep: ","},
	CAD: {Code: CAD, Name: "Canadian Dollar", Symbol: "$", DecimalPlaces: 2, SymbolBefore: true, ThousandsSep: ",", DecimalSep: "."},
	AUD: {Code: AUD, Name: "Australian Dollar", Symbol: "$", DecimalPlaces: 2, SymbolBefore: true, ThousandsSep: ",", DecimalSep: "."},
	CHF: {Code: CHF, Name: "Swiss Franc", Symbol: "CHF", DecimalPlaces: 2, SymbolBefore: true, ThousandsSep: "'", DecimalSep: "."},
	SGD: {Code: SGD, Name: "Singapore Dollar", Symbol: "$", DecimalPlaces: 2, SymbolBefore: true, ThousandsSep: ",", DecimalSep: "."},
}

// SupportedCurrencies returns a list of all supported currency codes.
func SupportedCurrencies() []Currency {
	return []Currency{USD, EUR, GBP, JPY, CNY, VND, CAD, AUD, CHF, SGD}
}

// SupportedCurrencyCodes returns a list of all supported currency codes as strings.
func SupportedCurrencyCodes() []string {
	codes := SupportedCurrencies()
	result := make([]string, len(codes))
	for i, c := range codes {
		result[i] = string(c)
	}
	return result
}

// IsValid checks if a currency code is supported.
func IsValid(code string) bool {
	_, ok := currencies[Currency(code)]
	return ok
}

// GetInfo returns metadata for a currency code.
func GetInfo(code Currency) (CurrencyInfo, bool) {
	info, ok := currencies[code]
	return info, ok
}

// Money represents a monetary amount with currency.
type Money struct {
	Amount   decimal.Decimal `json:"amount"`
	Currency Currency        `json:"currency"`
}

// NewMoney creates a new Money value.
func NewMoney(amount decimal.Decimal, curr Currency) Money {
	if curr == "" {
		curr = DefaultCurrency
	}
	return Money{Amount: amount, Currency: curr}
}

// NewMoneyFromFloat creates a Money from a float64 value.
func NewMoneyFromFloat(amount float64, curr Currency) Money {
	return NewMoney(decimal.NewFromFloat(amount), curr)
}

// NewMoneyFromString creates a Money from a string value.
func NewMoneyFromString(amount string, curr Currency) (Money, error) {
	d, err := decimal.NewFromString(amount)
	if err != nil {
		return Money{}, fmt.Errorf("invalid amount: %w", err)
	}
	return NewMoney(d, curr), nil
}

// Zero returns a zero amount in the specified currency.
func Zero(curr Currency) Money {
	return NewMoney(decimal.Zero, curr)
}

// Add returns the sum of two Money values.
// Returns an error if currencies don't match.
func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("currency mismatch: %s vs %s", m.Currency, other.Currency)
	}
	return NewMoney(m.Amount.Add(other.Amount), m.Currency), nil
}

// Sub returns the difference of two Money values.
// Returns an error if currencies don't match.
func (m Money) Sub(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, fmt.Errorf("currency mismatch: %s vs %s", m.Currency, other.Currency)
	}
	return NewMoney(m.Amount.Sub(other.Amount), m.Currency), nil
}

// Multiply returns the Money multiplied by a factor.
func (m Money) Multiply(factor decimal.Decimal) Money {
	return NewMoney(m.Amount.Mul(factor), m.Currency)
}

// IsZero returns true if the amount is zero.
func (m Money) IsZero() bool {
	return m.Amount.IsZero()
}

// IsPositive returns true if the amount is positive.
func (m Money) IsPositive() bool {
	return m.Amount.IsPositive()
}

// IsNegative returns true if the amount is negative.
func (m Money) IsNegative() bool {
	return m.Amount.IsNegative()
}

// Abs returns the absolute value.
func (m Money) Abs() Money {
	return NewMoney(m.Amount.Abs(), m.Currency)
}

// Round rounds the amount to the currency's decimal places.
func (m Money) Round() Money {
	info, ok := GetInfo(m.Currency)
	if !ok {
		info = currencies[DefaultCurrency]
	}
	return NewMoney(m.Amount.Round(int32(info.DecimalPlaces)), m.Currency)
}

// Format returns a formatted string representation.
// Uses the currency's standard formatting rules.
func (m Money) Format() string {
	info, ok := GetInfo(m.Currency)
	if !ok {
		return fmt.Sprintf("%s %s", m.Amount.StringFixed(2), m.Currency)
	}

	rounded := m.Amount.Round(int32(info.DecimalPlaces))

	if info.SymbolBefore {
		return fmt.Sprintf("%s%s", info.Symbol, rounded.StringFixed(int32(info.DecimalPlaces)))
	}
	return fmt.Sprintf("%s%s", rounded.StringFixed(int32(info.DecimalPlaces)), info.Symbol)
}

// String returns the amount as a plain string.
func (m Money) String() string {
	info, ok := GetInfo(m.Currency)
	if !ok {
		return m.Amount.String()
	}
	return m.Amount.Round(int32(info.DecimalPlaces)).String()
}
