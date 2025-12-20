// Package datetime provides standardized date and time handling across the application.
// All dates are stored and transmitted in UTC timezone using ISO 8601 format.
package datetime

import (
	"encoding/json"
	"strings"
	"time"
)

// Standard date formats used throughout the application.
const (
	// DateFormat is the standard date-only format (YYYY-MM-DD).
	DateFormat = "2006-01-02"

	// DateTimeFormat is the standard datetime format (ISO 8601 / RFC3339).
	DateTimeFormat = time.RFC3339

	// DisplayDateFormat is for human-readable dates.
	DisplayDateFormat = "Jan 2, 2006"

	// DisplayDateTimeFormat is for human-readable datetimes.
	DisplayDateTimeFormat = "Jan 2, 2006 3:04 PM"
)

// Date represents a date-only value (no time component).
// It serializes to/from JSON as "YYYY-MM-DD" format.
type Date struct {
	time.Time
}

// NewDate creates a Date from year, month, day.
func NewDate(year int, month time.Month, day int) Date {
	return Date{time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

// Today returns today's date in UTC.
func Today() Date {
	now := time.Now().UTC()
	return NewDate(now.Year(), now.Month(), now.Day())
}

// ParseDate parses a date string in YYYY-MM-DD format.
func ParseDate(s string) (Date, error) {
	t, err := time.Parse(DateFormat, s)
	if err != nil {
		return Date{}, err
	}
	return Date{t}, nil
}

// MarshalJSON implements json.Marshaler.
func (d Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.Format(DateFormat))
}

// UnmarshalJSON implements json.Unmarshaler.
func (d *Date) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")
	if s == "" || s == "null" {
		return nil
	}

	// Try date-only format first
	t, err := time.Parse(DateFormat, s)
	if err == nil {
		d.Time = t
		return nil
	}

	// Fall back to RFC3339 (extract date portion)
	t, err = time.Parse(time.RFC3339, s)
	if err == nil {
		d.Time = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		return nil
	}

	return err
}

// String returns the date in YYYY-MM-DD format.
func (d Date) String() string {
	if d.IsZero() {
		return ""
	}
	return d.Format(DateFormat)
}

// DateTime represents a datetime value with timezone.
// It serializes to/from JSON as ISO 8601 / RFC3339 format.
type DateTime struct {
	time.Time
}

// Now returns the current datetime in UTC.
func Now() DateTime {
	return DateTime{time.Now().UTC()}
}

// ParseDateTime parses a datetime string in RFC3339 format.
func ParseDateTime(s string) (DateTime, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return DateTime{}, err
	}
	return DateTime{t}, nil
}

// MarshalJSON implements json.Marshaler.
func (dt DateTime) MarshalJSON() ([]byte, error) {
	if dt.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(dt.UTC().Format(time.RFC3339))
}

// UnmarshalJSON implements json.Unmarshaler.
func (dt *DateTime) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")
	if s == "" || s == "null" {
		return nil
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Try date-only format as fallback
		t, err = time.Parse(DateFormat, s)
		if err != nil {
			return err
		}
	}
	dt.Time = t.UTC()
	return nil
}

// String returns the datetime in RFC3339 format.
func (dt DateTime) String() string {
	if dt.IsZero() {
		return ""
	}
	return dt.UTC().Format(time.RFC3339)
}

// ToDate extracts the date portion from a DateTime.
func (dt DateTime) ToDate() Date {
	return NewDate(dt.Year(), dt.Month(), dt.Day())
}

// StartOfDay returns the datetime at 00:00:00 UTC.
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// EndOfDay returns the datetime at 23:59:59.999999999 UTC.
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.UTC)
}

// StartOfMonth returns the first day of the month at 00:00:00 UTC.
func StartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// EndOfMonth returns the last day of the month at 23:59:59.999999999 UTC.
func EndOfMonth(t time.Time) time.Time {
	return StartOfMonth(t).AddDate(0, 1, 0).Add(-time.Nanosecond)
}

// StartOfYear returns the first day of the year at 00:00:00 UTC.
func StartOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
}

// EndOfYear returns the last day of the year at 23:59:59.999999999 UTC.
func EndOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), time.December, 31, 23, 59, 59, 999999999, time.UTC)
}
