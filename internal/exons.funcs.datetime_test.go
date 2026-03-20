package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test datetime functions are registered
func TestRegisterDateTimeFuncs(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	expectedFuncs := []string{
		FuncNameNow,
		FuncNameFormatDate,
		FuncNameParseDate,
		FuncNameAddDays,
		FuncNameAddHours,
		FuncNameAddMinutes,
		FuncNameDiffDays,
		FuncNameYear,
		FuncNameMonth,
		FuncNameDay,
		FuncNameWeekday,
		FuncNameIsAfter,
		FuncNameIsBefore,
	}

	for _, name := range expectedFuncs {
		assert.True(t, r.Has(name), "expected function %s to be registered", name)
	}
}

// Test now() function
func TestBuiltinFunc_Now(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	before := time.Now()
	result, err := r.Call(FuncNameNow, []any{})
	after := time.Now()

	require.NoError(t, err)
	resultTime, ok := result.(time.Time)
	require.True(t, ok, "expected time.Time result")

	assert.True(t, resultTime.After(before) || resultTime.Equal(before))
	assert.True(t, resultTime.Before(after) || resultTime.Equal(after))
}

// Test formatDate() function
func TestBuiltinFunc_FormatDate(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	testTime := time.Date(2024, 12, 25, 10, 30, 45, 0, time.UTC)

	tests := []struct {
		name     string
		time     any
		layout   string
		expected string
	}{
		{"ISO date", testTime, DateFormatISO, "2024-12-25"},
		{"US date", testTime, DateFormatUS, "12/25/2024"},
		{"EU date", testTime, DateFormatEU, "25/12/2024"},
		{"Full datetime", testTime, "2006-01-02 15:04:05", "2024-12-25 10:30:45"},
		{"Time only 24h", testTime, TimeFormat24H, "10:30:45"},
		{"Custom format", testTime, "Jan 2, 2006", "Dec 25, 2024"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Call(FuncNameFormatDate, []any{tt.time, tt.layout})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuiltinFunc_FormatDate_Errors(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	t.Run("invalid time argument", func(t *testing.T) {
		_, err := r.Call(FuncNameFormatDate, []any{"not a time", DateFormatISO})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgFuncExpectedTime)
	})

	t.Run("invalid layout argument", func(t *testing.T) {
		_, err := r.Call(FuncNameFormatDate, []any{time.Now(), 12345})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgFuncExpectedTimeLayout)
	})
}

// Test parseDate() function
func TestBuiltinFunc_ParseDate(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	t.Run("with explicit layout", func(t *testing.T) {
		result, err := r.Call(FuncNameParseDate, []any{"2024-12-25", DateFormatISO})
		require.NoError(t, err)

		resultTime, ok := result.(time.Time)
		require.True(t, ok)
		assert.Equal(t, 2024, resultTime.Year())
		assert.Equal(t, time.December, resultTime.Month())
		assert.Equal(t, 25, resultTime.Day())
	})

	t.Run("auto-detect RFC3339", func(t *testing.T) {
		result, err := r.Call(FuncNameParseDate, []any{"2024-12-25T10:30:00Z"})
		require.NoError(t, err)

		resultTime, ok := result.(time.Time)
		require.True(t, ok)
		assert.Equal(t, 2024, resultTime.Year())
		assert.Equal(t, 10, resultTime.Hour())
	})

	t.Run("auto-detect ISO date", func(t *testing.T) {
		result, err := r.Call(FuncNameParseDate, []any{"2024-12-25"})
		require.NoError(t, err)

		resultTime, ok := result.(time.Time)
		require.True(t, ok)
		assert.Equal(t, 2024, resultTime.Year())
	})
}

func TestBuiltinFunc_ParseDate_Errors(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	t.Run("invalid string format", func(t *testing.T) {
		_, err := r.Call(FuncNameParseDate, []any{"not a date"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgFuncInvalidTimeFormat)
	})

	t.Run("invalid input type", func(t *testing.T) {
		_, err := r.Call(FuncNameParseDate, []any{12345})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgFuncExpectedString)
	})

	t.Run("invalid layout type", func(t *testing.T) {
		_, err := r.Call(FuncNameParseDate, []any{"2024-12-25", 12345})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgFuncExpectedTimeLayout)
	})

	t.Run("wrong format for layout", func(t *testing.T) {
		_, err := r.Call(FuncNameParseDate, []any{"25/12/2024", DateFormatISO})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgFuncInvalidTimeFormat)
	})
}

// Test addDays() function
func TestBuiltinFunc_AddDays(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	baseTime := time.Date(2024, 12, 25, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		days        int
		expectedDay int
	}{
		{"add 1 day", 1, 26},
		{"add 7 days", 7, 1}, // Wraps to January
		{"subtract 1 day", -1, 24},
		{"add 0 days", 0, 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Call(FuncNameAddDays, []any{baseTime, tt.days})
			require.NoError(t, err)

			resultTime, ok := result.(time.Time)
			require.True(t, ok)
			assert.Equal(t, tt.expectedDay, resultTime.Day())
		})
	}
}

func TestBuiltinFunc_AddDays_Errors(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	t.Run("invalid time argument", func(t *testing.T) {
		_, err := r.Call(FuncNameAddDays, []any{"not a time", 5})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgFuncExpectedTime)
	})

	t.Run("invalid days argument", func(t *testing.T) {
		_, err := r.Call(FuncNameAddDays, []any{time.Now(), "five"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgFuncExpectedInteger)
	})
}

// Test addHours() function
func TestBuiltinFunc_AddHours(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	baseTime := time.Date(2024, 12, 25, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		hours        int
		expectedHour int
	}{
		{"add 2 hours", 2, 12},
		{"add 14 hours (next day)", 14, 0},
		{"subtract 2 hours", -2, 8},
		{"add 0 hours", 0, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Call(FuncNameAddHours, []any{baseTime, tt.hours})
			require.NoError(t, err)

			resultTime, ok := result.(time.Time)
			require.True(t, ok)
			assert.Equal(t, tt.expectedHour, resultTime.Hour())
		})
	}
}

// Test addMinutes() function
func TestBuiltinFunc_AddMinutes(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	baseTime := time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name           string
		minutes        int
		expectedMinute int
	}{
		{"add 15 minutes", 15, 45},
		{"add 30 minutes (next hour)", 30, 0},
		{"subtract 15 minutes", -15, 15},
		{"add 0 minutes", 0, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Call(FuncNameAddMinutes, []any{baseTime, tt.minutes})
			require.NoError(t, err)

			resultTime, ok := result.(time.Time)
			require.True(t, ok)
			assert.Equal(t, tt.expectedMinute, resultTime.Minute())
		})
	}
}

// Test diffDays() function
func TestBuiltinFunc_DiffDays(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	time1 := time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 12, 28, 0, 0, 0, 0, time.UTC)
	time3 := time.Date(2024, 12, 20, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		t1       time.Time
		t2       time.Time
		expected int
	}{
		{"3 days later", time1, time2, 3},
		{"5 days earlier", time1, time3, -5},
		{"same day", time1, time1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := r.Call(FuncNameDiffDays, []any{tt.t1, tt.t2})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuiltinFunc_DiffDays_Errors(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	t.Run("invalid first argument", func(t *testing.T) {
		_, err := r.Call(FuncNameDiffDays, []any{"not a time", time.Now()})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgFuncExpectedTime)
	})

	t.Run("invalid second argument", func(t *testing.T) {
		_, err := r.Call(FuncNameDiffDays, []any{time.Now(), "not a time"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMsgFuncExpectedTime)
	})
}

// Test year() function
func TestBuiltinFunc_Year(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	testTime := time.Date(2024, 12, 25, 10, 30, 45, 0, time.UTC)

	result, err := r.Call(FuncNameYear, []any{testTime})
	require.NoError(t, err)
	assert.Equal(t, 2024, result)
}

// Test month() function
func TestBuiltinFunc_Month(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	testTime := time.Date(2024, 12, 25, 10, 30, 45, 0, time.UTC)

	result, err := r.Call(FuncNameMonth, []any{testTime})
	require.NoError(t, err)
	assert.Equal(t, 12, result)
}

// Test day() function
func TestBuiltinFunc_Day(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	testTime := time.Date(2024, 12, 25, 10, 30, 45, 0, time.UTC)

	result, err := r.Call(FuncNameDay, []any{testTime})
	require.NoError(t, err)
	assert.Equal(t, 25, result)
}

// Test weekday() function
func TestBuiltinFunc_Weekday(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	// December 25, 2024 is a Wednesday
	testTime := time.Date(2024, 12, 25, 10, 30, 45, 0, time.UTC)

	result, err := r.Call(FuncNameWeekday, []any{testTime})
	require.NoError(t, err)
	assert.Equal(t, "Wednesday", result)
}

// Test isAfter() function
func TestBuiltinFunc_IsAfter(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	time1 := time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 12, 20, 0, 0, 0, 0, time.UTC)

	t.Run("t1 after t2", func(t *testing.T) {
		result, err := r.Call(FuncNameIsAfter, []any{time1, time2})
		require.NoError(t, err)
		assert.Equal(t, true, result)
	})

	t.Run("t1 before t2", func(t *testing.T) {
		result, err := r.Call(FuncNameIsAfter, []any{time2, time1})
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})

	t.Run("same time", func(t *testing.T) {
		result, err := r.Call(FuncNameIsAfter, []any{time1, time1})
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})
}

// Test isBefore() function
func TestBuiltinFunc_IsBefore(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	time1 := time.Date(2024, 12, 20, 0, 0, 0, 0, time.UTC)
	time2 := time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC)

	t.Run("t1 before t2", func(t *testing.T) {
		result, err := r.Call(FuncNameIsBefore, []any{time1, time2})
		require.NoError(t, err)
		assert.Equal(t, true, result)
	})

	t.Run("t1 after t2", func(t *testing.T) {
		result, err := r.Call(FuncNameIsBefore, []any{time2, time1})
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})

	t.Run("same time", func(t *testing.T) {
		result, err := r.Call(FuncNameIsBefore, []any{time1, time1})
		require.NoError(t, err)
		assert.Equal(t, false, result)
	})
}

// Test toTime helper function
func TestToTime(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{"time.Time", time.Now(), false},
		{"*time.Time", func() *time.Time { t := time.Now(); return &t }(), false},
		{"string RFC3339", "2024-12-25T10:30:00Z", false},
		{"string ISO date", "2024-12-25", false},
		{"int64 unix timestamp", int64(1735123200), false},
		{"int unix timestamp", 1735123200, false},
		{"float64 unix timestamp", 1735123200.5, false},
		{"nil", nil, true},
		{"nil *time.Time", (*time.Time)(nil), true},
		{"invalid string", "not a date", true},
		{"unsupported type", []string{"not", "valid"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := toTime(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.False(t, result.IsZero())
			}
		})
	}
}

// Test parseTimeString helper function
func TestParseTimeString(t *testing.T) {
	validFormats := []string{
		"2024-12-25T10:30:00Z",           // RFC3339
		"2024-12-25T10:30:00.123456789Z", // RFC3339Nano
		"2024-12-25",                     // ISO date
		"2024-12-25 15:04:05",            // Datetime with space
		"12/25/2024",                     // US format
		"25/12/2024",                     // EU format
		"Dec 25, 2024",                   // Short month name
		"December 25, 2024",              // Long month name
	}

	for _, format := range validFormats {
		t.Run(format, func(t *testing.T) {
			result, err := parseTimeString(format)
			assert.NoError(t, err)
			assert.False(t, result.IsZero())
		})
	}

	t.Run("invalid format", func(t *testing.T) {
		_, err := parseTimeString("not a valid date")
		assert.Error(t, err)
	})
}

// Test component extraction errors
func TestDateTimeFuncs_ComponentErrors(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	funcs := []string{
		FuncNameYear,
		FuncNameMonth,
		FuncNameDay,
		FuncNameWeekday,
	}

	for _, fn := range funcs {
		t.Run(fn+"_invalid_input", func(t *testing.T) {
			_, err := r.Call(fn, []any{"not a time"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), ErrMsgFuncExpectedTime)
		})
	}
}

// Test comparison function errors
func TestDateTimeFuncs_ComparisonErrors(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	funcs := []string{FuncNameIsAfter, FuncNameIsBefore}

	for _, fn := range funcs {
		t.Run(fn+"_invalid_first", func(t *testing.T) {
			_, err := r.Call(fn, []any{"not a time", time.Now()})
			require.Error(t, err)
			assert.Contains(t, err.Error(), ErrMsgFuncExpectedTime)
		})

		t.Run(fn+"_invalid_second", func(t *testing.T) {
			_, err := r.Call(fn, []any{time.Now(), "not a time"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), ErrMsgFuncExpectedTime)
		})
	}
}

// Test addHours and addMinutes errors
func TestDateTimeFuncs_AddErrors(t *testing.T) {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)

	addFuncs := []string{FuncNameAddHours, FuncNameAddMinutes}

	for _, fn := range addFuncs {
		t.Run(fn+"_invalid_time", func(t *testing.T) {
			_, err := r.Call(fn, []any{"not a time", 5})
			require.Error(t, err)
			assert.Contains(t, err.Error(), ErrMsgFuncExpectedTime)
		})

		t.Run(fn+"_invalid_amount", func(t *testing.T) {
			_, err := r.Call(fn, []any{time.Now(), "five"})
			require.Error(t, err)
			assert.Contains(t, err.Error(), ErrMsgFuncExpectedInteger)
		})
	}
}

// Test that datetime functions are included in RegisterBuiltinFuncs
func TestRegisterBuiltinFuncs_IncludesDateTime(t *testing.T) {
	r := NewFuncRegistry()
	RegisterBuiltinFuncs(r)

	// Check datetime functions are registered
	datetimeFuncs := []string{
		FuncNameNow,
		FuncNameFormatDate,
		FuncNameParseDate,
		FuncNameAddDays,
		FuncNameAddHours,
		FuncNameAddMinutes,
		FuncNameDiffDays,
		FuncNameYear,
		FuncNameMonth,
		FuncNameDay,
		FuncNameWeekday,
		FuncNameIsAfter,
		FuncNameIsBefore,
	}

	for _, name := range datetimeFuncs {
		assert.True(t, r.Has(name), "expected datetime function %s to be registered", name)
	}
}
