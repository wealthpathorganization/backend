package datetime

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDate(t *testing.T) {
	d := NewDate(2024, time.December, 25)
	assert.Equal(t, 2024, d.Year())
	assert.Equal(t, time.December, d.Month())
	assert.Equal(t, 25, d.Day())
	assert.Equal(t, time.UTC, d.Location())
}

func TestToday(t *testing.T) {
	today := Today()
	now := time.Now().UTC()
	assert.Equal(t, now.Year(), today.Year())
	assert.Equal(t, now.Month(), today.Month())
	assert.Equal(t, now.Day(), today.Day())
}

func TestParseDate(t *testing.T) {
	t.Run("valid date", func(t *testing.T) {
		d, err := ParseDate("2024-12-25")
		require.NoError(t, err)
		assert.Equal(t, 2024, d.Year())
		assert.Equal(t, time.December, d.Month())
		assert.Equal(t, 25, d.Day())
	})

	t.Run("invalid date", func(t *testing.T) {
		_, err := ParseDate("not-a-date")
		assert.Error(t, err)
	})

	t.Run("wrong format", func(t *testing.T) {
		_, err := ParseDate("25/12/2024")
		assert.Error(t, err)
	})
}

func TestDateMarshalJSON(t *testing.T) {
	t.Run("valid date", func(t *testing.T) {
		d := NewDate(2024, time.December, 25)
		data, err := json.Marshal(d)
		require.NoError(t, err)
		assert.Equal(t, `"2024-12-25"`, string(data))
	})

	t.Run("zero date", func(t *testing.T) {
		d := Date{}
		data, err := json.Marshal(d)
		require.NoError(t, err)
		assert.Equal(t, "null", string(data))
	})
}

func TestDateUnmarshalJSON(t *testing.T) {
	t.Run("date-only format", func(t *testing.T) {
		var d Date
		err := json.Unmarshal([]byte(`"2024-12-25"`), &d)
		require.NoError(t, err)
		assert.Equal(t, 2024, d.Year())
		assert.Equal(t, time.December, d.Month())
		assert.Equal(t, 25, d.Day())
	})

	t.Run("RFC3339 format", func(t *testing.T) {
		var d Date
		err := json.Unmarshal([]byte(`"2024-12-25T10:30:00Z"`), &d)
		require.NoError(t, err)
		assert.Equal(t, 2024, d.Year())
		assert.Equal(t, time.December, d.Month())
		assert.Equal(t, 25, d.Day())
	})

	t.Run("null value", func(t *testing.T) {
		var d Date
		err := json.Unmarshal([]byte(`null`), &d)
		require.NoError(t, err)
		assert.True(t, d.IsZero())
	})

	t.Run("empty string", func(t *testing.T) {
		var d Date
		err := json.Unmarshal([]byte(`""`), &d)
		require.NoError(t, err)
		assert.True(t, d.IsZero())
	})

	t.Run("invalid format", func(t *testing.T) {
		var d Date
		err := json.Unmarshal([]byte(`"invalid-date"`), &d)
		assert.Error(t, err)
	})
}

func TestDateString(t *testing.T) {
	t.Run("valid date", func(t *testing.T) {
		d := NewDate(2024, time.December, 25)
		assert.Equal(t, "2024-12-25", d.String())
	})

	t.Run("zero date", func(t *testing.T) {
		d := Date{}
		assert.Equal(t, "", d.String())
	})
}

func TestNow(t *testing.T) {
	dt := Now()
	now := time.Now().UTC()
	// Allow 1 second difference
	assert.WithinDuration(t, now, dt.Time, time.Second)
}

func TestParseDateTime(t *testing.T) {
	t.Run("valid datetime", func(t *testing.T) {
		dt, err := ParseDateTime("2024-12-25T10:30:00Z")
		require.NoError(t, err)
		assert.Equal(t, 2024, dt.Year())
		assert.Equal(t, time.December, dt.Month())
		assert.Equal(t, 25, dt.Day())
		assert.Equal(t, 10, dt.Hour())
		assert.Equal(t, 30, dt.Minute())
	})

	t.Run("invalid datetime", func(t *testing.T) {
		_, err := ParseDateTime("not-a-datetime")
		assert.Error(t, err)
	})
}

func TestDateTimeMarshalJSON(t *testing.T) {
	t.Run("valid datetime", func(t *testing.T) {
		dt := DateTime{time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC)}
		data, err := json.Marshal(dt)
		require.NoError(t, err)
		assert.Equal(t, `"2024-12-25T10:30:00Z"`, string(data))
	})

	t.Run("zero datetime", func(t *testing.T) {
		dt := DateTime{}
		data, err := json.Marshal(dt)
		require.NoError(t, err)
		assert.Equal(t, "null", string(data))
	})
}

func TestDateTimeUnmarshalJSON(t *testing.T) {
	t.Run("RFC3339 format", func(t *testing.T) {
		var dt DateTime
		err := json.Unmarshal([]byte(`"2024-12-25T10:30:00Z"`), &dt)
		require.NoError(t, err)
		assert.Equal(t, 2024, dt.Year())
		assert.Equal(t, 10, dt.Hour())
	})

	t.Run("date-only format fallback", func(t *testing.T) {
		var dt DateTime
		err := json.Unmarshal([]byte(`"2024-12-25"`), &dt)
		require.NoError(t, err)
		assert.Equal(t, 2024, dt.Year())
		assert.Equal(t, time.December, dt.Month())
		assert.Equal(t, 25, dt.Day())
	})

	t.Run("null value", func(t *testing.T) {
		var dt DateTime
		err := json.Unmarshal([]byte(`null`), &dt)
		require.NoError(t, err)
		assert.True(t, dt.IsZero())
	})

	t.Run("empty string", func(t *testing.T) {
		var dt DateTime
		err := json.Unmarshal([]byte(`""`), &dt)
		require.NoError(t, err)
		assert.True(t, dt.IsZero())
	})

	t.Run("invalid format", func(t *testing.T) {
		var dt DateTime
		err := json.Unmarshal([]byte(`"invalid"`), &dt)
		assert.Error(t, err)
	})
}

func TestDateTimeString(t *testing.T) {
	t.Run("valid datetime", func(t *testing.T) {
		dt := DateTime{time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC)}
		assert.Equal(t, "2024-12-25T10:30:00Z", dt.String())
	})

	t.Run("zero datetime", func(t *testing.T) {
		dt := DateTime{}
		assert.Equal(t, "", dt.String())
	})
}

func TestDateTimeToDate(t *testing.T) {
	dt := DateTime{time.Date(2024, 12, 25, 10, 30, 45, 0, time.UTC)}
	d := dt.ToDate()
	assert.Equal(t, 2024, d.Year())
	assert.Equal(t, time.December, d.Month())
	assert.Equal(t, 25, d.Day())
	assert.Equal(t, 0, d.Hour())
	assert.Equal(t, 0, d.Minute())
	assert.Equal(t, 0, d.Second())
}

func TestStartOfDay(t *testing.T) {
	input := time.Date(2024, 12, 25, 15, 30, 45, 123456789, time.UTC)
	result := StartOfDay(input)
	assert.Equal(t, 2024, result.Year())
	assert.Equal(t, time.December, result.Month())
	assert.Equal(t, 25, result.Day())
	assert.Equal(t, 0, result.Hour())
	assert.Equal(t, 0, result.Minute())
	assert.Equal(t, 0, result.Second())
	assert.Equal(t, 0, result.Nanosecond())
}

func TestEndOfDay(t *testing.T) {
	input := time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC)
	result := EndOfDay(input)
	assert.Equal(t, 2024, result.Year())
	assert.Equal(t, time.December, result.Month())
	assert.Equal(t, 25, result.Day())
	assert.Equal(t, 23, result.Hour())
	assert.Equal(t, 59, result.Minute())
	assert.Equal(t, 59, result.Second())
	assert.Equal(t, 999999999, result.Nanosecond())
}

func TestStartOfMonth(t *testing.T) {
	input := time.Date(2024, 12, 25, 15, 30, 45, 0, time.UTC)
	result := StartOfMonth(input)
	assert.Equal(t, 2024, result.Year())
	assert.Equal(t, time.December, result.Month())
	assert.Equal(t, 1, result.Day())
	assert.Equal(t, 0, result.Hour())
}

func TestEndOfMonth(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected int // expected day of month
	}{
		{"December", time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC), 31},
		{"February leap year", time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC), 29},
		{"February non-leap year", time.Date(2023, 2, 15, 0, 0, 0, 0, time.UTC), 28},
		{"April", time.Date(2024, 4, 15, 0, 0, 0, 0, time.UTC), 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EndOfMonth(tt.input)
			assert.Equal(t, tt.expected, result.Day())
			assert.Equal(t, 23, result.Hour())
			assert.Equal(t, 59, result.Minute())
		})
	}
}

func TestStartOfYear(t *testing.T) {
	input := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	result := StartOfYear(input)
	assert.Equal(t, 2024, result.Year())
	assert.Equal(t, time.January, result.Month())
	assert.Equal(t, 1, result.Day())
	assert.Equal(t, 0, result.Hour())
}

func TestEndOfYear(t *testing.T) {
	input := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	result := EndOfYear(input)
	assert.Equal(t, 2024, result.Year())
	assert.Equal(t, time.December, result.Month())
	assert.Equal(t, 31, result.Day())
	assert.Equal(t, 23, result.Hour())
	assert.Equal(t, 59, result.Minute())
	assert.Equal(t, 59, result.Second())
}
