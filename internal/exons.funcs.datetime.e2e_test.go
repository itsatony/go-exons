package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Now Function Tests
// =============================================================================

// createDateTimeRegistry creates a new registry with datetime functions registered
func createDateTimeRegistry() *FuncRegistry {
	r := NewFuncRegistry()
	registerDateTimeFuncs(r)
	return r
}

func TestDateTime_E2E_NowConsistency(t *testing.T) {
	registry := createDateTimeRegistry()

	t.Run("MultipleCalls", func(t *testing.T) {
		nowFunc, ok := registry.Get(FuncNameNow)
		require.True(t, ok)

		// Call now() multiple times in quick succession
		var times []time.Time
		for i := 0; i < 10; i++ {
			result, err := nowFunc.Fn(nil)
			require.NoError(t, err)
			times = append(times, result.(time.Time))
		}

		// All times should be within a reasonable window (1 second)
		first := times[0]
		last := times[len(times)-1]
		assert.True(t, last.Sub(first) < time.Second,
			"Multiple now() calls should be within 1 second")
	})

	t.Run("ReturnsCurrentTime", func(t *testing.T) {
		nowFunc, _ := registry.Get(FuncNameNow)
		before := time.Now()
		result, err := nowFunc.Fn(nil)
		after := time.Now()

		require.NoError(t, err)
		resultTime := result.(time.Time)

		// Result should be between before and after
		assert.True(t, !resultTime.Before(before), "now() should not be before call")
		assert.True(t, !resultTime.After(after), "now() should not be after call")
	})
}

// =============================================================================
// Format Date Tests
// =============================================================================

func TestDateTime_E2E_FormatAllLayouts(t *testing.T) {
	registry := createDateTimeRegistry()
	formatFunc, ok := registry.Get(FuncNameFormatDate)
	require.True(t, ok)

	// Fixed test time: 2024-06-15 14:30:45 UTC
	testTime := time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)

	testCases := []struct {
		layout   string
		expected string
	}{
		{DateFormatISO, "2024-06-15"},
		{DateFormatUS, "06/15/2024"},
		{DateFormatEU, "15/06/2024"},
		{DateTimeFormatISO, "2024-06-15T14:30:45Z"},
		{TimeFormat24H, "14:30:45"},
		{TimeFormat12H, "2:30:45 PM"},
	}

	for _, tc := range testCases {
		t.Run(tc.layout, func(t *testing.T) {
			result, err := formatFunc.Fn([]any{testTime, tc.layout})
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDateTime_E2E_FormatEdgeCases(t *testing.T) {
	registry := createDateTimeRegistry()
	formatFunc, _ := registry.Get(FuncNameFormatDate)

	t.Run("MidnightTime", func(t *testing.T) {
		midnight := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		result, err := formatFunc.Fn([]any{midnight, TimeFormat24H})
		require.NoError(t, err)
		assert.Equal(t, "00:00:00", result)
	})

	t.Run("EndOfDay", func(t *testing.T) {
		endOfDay := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
		result, err := formatFunc.Fn([]any{endOfDay, DateTimeFormatISO})
		require.NoError(t, err)
		assert.Contains(t, result.(string), "23:59:59")
	})

	t.Run("CustomFormat", func(t *testing.T) {
		testTime := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
		result, err := formatFunc.Fn([]any{testTime, "Monday, January 2, 2006"})
		require.NoError(t, err)
		assert.Equal(t, "Saturday, June 15, 2024", result)
	})
}

// =============================================================================
// Parse Date Tests
// =============================================================================

func TestDateTime_E2E_ParseAutoDetect(t *testing.T) {
	registry := createDateTimeRegistry()
	parseFunc, ok := registry.Get(FuncNameParseDate)
	require.True(t, ok)

	testCases := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{"ISO", "2024-06-15", time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)},
		{"US", "06/15/2024", time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)},
		{"EU", "15/06/2024", time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)},
		{"RFC3339", "2024-06-15T14:30:45Z", time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)},
		{"DateTimeSpace", "2024-06-15 14:30:45", time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)},
		{"SlashDate", "2024/06/15", time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)},
		{"NamedMonth", "Jun 15, 2024", time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)},
		{"FullNamedMonth", "June 15, 2024", time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseFunc.Fn([]any{tc.input})
			require.NoError(t, err, "Failed to parse: %s", tc.input)

			resultTime := result.(time.Time)
			assert.Equal(t, tc.expected.Year(), resultTime.Year())
			assert.Equal(t, tc.expected.Month(), resultTime.Month())
			assert.Equal(t, tc.expected.Day(), resultTime.Day())
		})
	}
}

func TestDateTime_E2E_ParseWithExplicitLayout(t *testing.T) {
	registry := createDateTimeRegistry()
	parseFunc, _ := registry.Get(FuncNameParseDate)

	t.Run("CustomLayout", func(t *testing.T) {
		result, err := parseFunc.Fn([]any{"15-Jun-2024", "02-Jan-2006"})
		require.NoError(t, err)

		resultTime := result.(time.Time)
		assert.Equal(t, 2024, resultTime.Year())
		assert.Equal(t, time.June, resultTime.Month())
		assert.Equal(t, 15, resultTime.Day())
	})

	t.Run("InvalidFormat", func(t *testing.T) {
		_, err := parseFunc.Fn([]any{"not-a-date"})
		require.Error(t, err)
	})

	t.Run("WrongLayoutFormat", func(t *testing.T) {
		_, err := parseFunc.Fn([]any{"2024-06-15", "02/01/2006"})
		require.Error(t, err)
	})
}

// =============================================================================
// Arithmetic Tests
// =============================================================================

func TestDateTime_E2E_ArithmeticBoundaries(t *testing.T) {
	registry := createDateTimeRegistry()
	addDaysFunc, _ := registry.Get(FuncNameAddDays)
	addHoursFunc, _ := registry.Get(FuncNameAddHours)
	addMinutesFunc, _ := registry.Get(FuncNameAddMinutes)

	t.Run("AddDays_MonthCrossing", func(t *testing.T) {
		// Jan 31 + 1 day = Feb 1
		jan31 := time.Date(2024, 1, 31, 12, 0, 0, 0, time.UTC)
		result, err := addDaysFunc.Fn([]any{jan31, 1})
		require.NoError(t, err)

		resultTime := result.(time.Time)
		assert.Equal(t, time.February, resultTime.Month())
		assert.Equal(t, 1, resultTime.Day())
	})

	t.Run("AddDays_YearCrossing", func(t *testing.T) {
		// Dec 31, 2024 + 1 day = Jan 1, 2025
		dec31 := time.Date(2024, 12, 31, 12, 0, 0, 0, time.UTC)
		result, err := addDaysFunc.Fn([]any{dec31, 1})
		require.NoError(t, err)

		resultTime := result.(time.Time)
		assert.Equal(t, 2025, resultTime.Year())
		assert.Equal(t, time.January, resultTime.Month())
		assert.Equal(t, 1, resultTime.Day())
	})

	t.Run("AddDays_LeapYear", func(t *testing.T) {
		// Feb 28, 2024 (leap year) + 1 = Feb 29
		feb28Leap := time.Date(2024, 2, 28, 12, 0, 0, 0, time.UTC)
		result, err := addDaysFunc.Fn([]any{feb28Leap, 1})
		require.NoError(t, err)

		resultTime := result.(time.Time)
		assert.Equal(t, time.February, resultTime.Month())
		assert.Equal(t, 29, resultTime.Day())

		// Feb 28, 2023 (non-leap year) + 1 = Mar 1
		feb28NonLeap := time.Date(2023, 2, 28, 12, 0, 0, 0, time.UTC)
		result, err = addDaysFunc.Fn([]any{feb28NonLeap, 1})
		require.NoError(t, err)

		resultTime = result.(time.Time)
		assert.Equal(t, time.March, resultTime.Month())
		assert.Equal(t, 1, resultTime.Day())
	})

	t.Run("AddDays_Negative", func(t *testing.T) {
		// Jan 1 - 1 day = Dec 31 (previous year)
		jan1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		result, err := addDaysFunc.Fn([]any{jan1, -1})
		require.NoError(t, err)

		resultTime := result.(time.Time)
		assert.Equal(t, 2023, resultTime.Year())
		assert.Equal(t, time.December, resultTime.Month())
		assert.Equal(t, 31, resultTime.Day())
	})

	t.Run("AddHours_DayCrossing", func(t *testing.T) {
		// 23:00 + 2 hours = 01:00 next day
		late := time.Date(2024, 6, 15, 23, 0, 0, 0, time.UTC)
		result, err := addHoursFunc.Fn([]any{late, 2})
		require.NoError(t, err)

		resultTime := result.(time.Time)
		assert.Equal(t, 16, resultTime.Day())
		assert.Equal(t, 1, resultTime.Hour())
	})

	t.Run("AddMinutes_HourCrossing", func(t *testing.T) {
		// 12:45 + 30 minutes = 13:15
		test := time.Date(2024, 6, 15, 12, 45, 0, 0, time.UTC)
		result, err := addMinutesFunc.Fn([]any{test, 30})
		require.NoError(t, err)

		resultTime := result.(time.Time)
		assert.Equal(t, 13, resultTime.Hour())
		assert.Equal(t, 15, resultTime.Minute())
	})

	t.Run("LargeAddition", func(t *testing.T) {
		// Add 365 days
		start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		result, err := addDaysFunc.Fn([]any{start, 365})
		require.NoError(t, err)

		resultTime := result.(time.Time)
		assert.Equal(t, 2024, resultTime.Year()) // 2024 is leap year, so still 2024
		assert.Equal(t, time.December, resultTime.Month())
		assert.Equal(t, 31, resultTime.Day())
	})
}

// =============================================================================
// DiffDays Tests
// =============================================================================

func TestDateTime_E2E_DiffDays(t *testing.T) {
	registry := createDateTimeRegistry()
	diffFunc, _ := registry.Get(FuncNameDiffDays)

	t.Run("PositiveDiff", func(t *testing.T) {
		t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC)

		result, err := diffFunc.Fn([]any{t1, t2})
		require.NoError(t, err)
		assert.Equal(t, 10, result)
	})

	t.Run("NegativeDiff", func(t *testing.T) {
		t1 := time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		result, err := diffFunc.Fn([]any{t1, t2})
		require.NoError(t, err)
		assert.Equal(t, -10, result)
	})

	t.Run("SameDay", func(t *testing.T) {
		t1 := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 6, 15, 23, 59, 59, 0, time.UTC)

		result, err := diffFunc.Fn([]any{t1, t2})
		require.NoError(t, err)
		assert.Equal(t, 0, result) // Less than 24 hours
	})

	t.Run("CrossYear", func(t *testing.T) {
		t1 := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		result, err := diffFunc.Fn([]any{t1, t2})
		require.NoError(t, err)
		assert.Equal(t, 1, result)
	})
}

// =============================================================================
// Component Extraction Tests
// =============================================================================

func TestDateTime_E2E_ComponentExtraction(t *testing.T) {
	registry := createDateTimeRegistry()
	yearFunc, _ := registry.Get(FuncNameYear)
	monthFunc, _ := registry.Get(FuncNameMonth)
	dayFunc, _ := registry.Get(FuncNameDay)
	weekdayFunc, _ := registry.Get(FuncNameWeekday)

	testTime := time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC) // Saturday

	t.Run("Year", func(t *testing.T) {
		result, err := yearFunc.Fn([]any{testTime})
		require.NoError(t, err)
		assert.Equal(t, 2024, result)
	})

	t.Run("Month", func(t *testing.T) {
		result, err := monthFunc.Fn([]any{testTime})
		require.NoError(t, err)
		assert.Equal(t, 6, result)
	})

	t.Run("Day", func(t *testing.T) {
		result, err := dayFunc.Fn([]any{testTime})
		require.NoError(t, err)
		assert.Equal(t, 15, result)
	})

	t.Run("Weekday", func(t *testing.T) {
		result, err := weekdayFunc.Fn([]any{testTime})
		require.NoError(t, err)
		assert.Equal(t, "Saturday", result)
	})

	t.Run("AllWeekdays", func(t *testing.T) {
		weekdays := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

		// June 2024: 1st is Saturday
		for i, expected := range weekdays {
			// Sunday (i=0) is June 16, Monday (i=1) is June 17, etc.
			day := time.Date(2024, 6, 16+i, 0, 0, 0, 0, time.UTC)
			result, err := weekdayFunc.Fn([]any{day})
			require.NoError(t, err)
			assert.Equal(t, expected, result, "Day %d should be %s", 16+i, expected)
		}
	})
}

// =============================================================================
// Comparison Tests
// =============================================================================

func TestDateTime_E2E_ComparisonEdgeCases(t *testing.T) {
	registry := createDateTimeRegistry()
	isAfterFunc, _ := registry.Get(FuncNameIsAfter)
	isBeforeFunc, _ := registry.Get(FuncNameIsBefore)

	t.Run("SameTime", func(t *testing.T) {
		t1 := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

		result, err := isAfterFunc.Fn([]any{t1, t2})
		require.NoError(t, err)
		assert.False(t, result.(bool), "Same time should not be after")

		result, err = isBeforeFunc.Fn([]any{t1, t2})
		require.NoError(t, err)
		assert.False(t, result.(bool), "Same time should not be before")
	})

	t.Run("MicrosecondDifference", func(t *testing.T) {
		t1 := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 6, 15, 12, 0, 0, 1000, time.UTC) // 1 microsecond later

		result, err := isAfterFunc.Fn([]any{t2, t1})
		require.NoError(t, err)
		assert.True(t, result.(bool), "Later time should be after")

		result, err = isBeforeFunc.Fn([]any{t1, t2})
		require.NoError(t, err)
		assert.True(t, result.(bool), "Earlier time should be before")
	})

	t.Run("ClearlyBefore", func(t *testing.T) {
		t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

		result, err := isBeforeFunc.Fn([]any{t1, t2})
		require.NoError(t, err)
		assert.True(t, result.(bool))

		result, err = isAfterFunc.Fn([]any{t2, t1})
		require.NoError(t, err)
		assert.True(t, result.(bool))
	})

	t.Run("DifferentYears", func(t *testing.T) {
		t1 := time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)
		t2 := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

		result, err := isBeforeFunc.Fn([]any{t1, t2})
		require.NoError(t, err)
		assert.True(t, result.(bool), "2023 should be before 2024")
	})
}

// =============================================================================
// Timezone Handling Tests
// =============================================================================

func TestDateTime_E2E_TimezoneHandling(t *testing.T) {
	registry := createDateTimeRegistry()
	formatFunc, _ := registry.Get(FuncNameFormatDate)
	parseFunc, _ := registry.Get(FuncNameParseDate)

	t.Run("UTCFormat", func(t *testing.T) {
		utcTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
		result, err := formatFunc.Fn([]any{utcTime, time.RFC3339})
		require.NoError(t, err)
		assert.Contains(t, result.(string), "Z")
	})

	t.Run("PreservesTimezone", func(t *testing.T) {
		// Create time in a specific timezone
		loc, err := time.LoadLocation("America/New_York")
		if err != nil {
			t.Skip("Timezone not available")
		}
		nyTime := time.Date(2024, 6, 15, 12, 0, 0, 0, loc)

		// Format should reflect the timezone
		result, err := formatFunc.Fn([]any{nyTime, time.RFC3339})
		require.NoError(t, err)
		// NYC in June is -04:00 (EDT)
		assert.Contains(t, result.(string), "-04:00")
	})

	t.Run("ParseWithTimezone", func(t *testing.T) {
		// Parse RFC3339 with timezone
		result, err := parseFunc.Fn([]any{"2024-06-15T12:00:00-04:00"})
		require.NoError(t, err)

		resultTime := result.(time.Time)
		// The parsed time should represent the same instant
		expectedUTC := time.Date(2024, 6, 15, 16, 0, 0, 0, time.UTC) // 12:00 EDT = 16:00 UTC
		assert.True(t, resultTime.Equal(expectedUTC),
			"Parsed time should equal expected UTC time")
	})
}

// =============================================================================
// Integration with Template Execution
// =============================================================================

func TestDateTime_E2E_InTemplateExecution(t *testing.T) {
	registry := createDateTimeRegistry()

	// Simulate expression evaluation context
	t.Run("FormatInExpression", func(t *testing.T) {
		formatFunc, _ := registry.Get(FuncNameFormatDate)
		testTime := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)

		// This simulates what would happen in a template like:
		// {~exons.if eval="formatDate(now(), '2006') == '2024'"~}
		result, err := formatFunc.Fn([]any{testTime, "2006"})
		require.NoError(t, err)
		assert.Equal(t, "2024", result)
	})

	t.Run("ComparisonInExpression", func(t *testing.T) {
		isAfterFunc, _ := registry.Get(FuncNameIsAfter)

		now := time.Now()
		past := now.Add(-24 * time.Hour)

		// This simulates: {~exons.if eval="isAfter(now(), deadline)"~}
		result, err := isAfterFunc.Fn([]any{now, past})
		require.NoError(t, err)
		assert.True(t, result.(bool))
	})

	t.Run("ArithmeticInExpression", func(t *testing.T) {
		addDaysFunc, _ := registry.Get(FuncNameAddDays)
		isAfterFunc, _ := registry.Get(FuncNameIsAfter)

		now := time.Now()

		// Add 7 days
		futureResult, err := addDaysFunc.Fn([]any{now, 7})
		require.NoError(t, err)
		futureTime := futureResult.(time.Time)

		// Future should be after now
		result, err := isAfterFunc.Fn([]any{futureTime, now})
		require.NoError(t, err)
		assert.True(t, result.(bool))
	})
}

// =============================================================================
// Type Conversion Tests
// =============================================================================

func TestDateTime_E2E_TypeConversion(t *testing.T) {
	registry := createDateTimeRegistry()
	yearFunc, _ := registry.Get(FuncNameYear)

	t.Run("TimeValue", func(t *testing.T) {
		tm := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		result, err := yearFunc.Fn([]any{tm})
		require.NoError(t, err)
		assert.Equal(t, 2024, result)
	})

	t.Run("TimePointer", func(t *testing.T) {
		tm := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		result, err := yearFunc.Fn([]any{&tm})
		require.NoError(t, err)
		assert.Equal(t, 2024, result)
	})

	t.Run("UnixTimestamp", func(t *testing.T) {
		// Unix timestamp for 2024-06-15 00:00:00 UTC
		ts := int64(1718409600)
		result, err := yearFunc.Fn([]any{ts})
		require.NoError(t, err)
		assert.Equal(t, 2024, result)
	})

	t.Run("UnixTimestampInt", func(t *testing.T) {
		ts := int(1718409600)
		result, err := yearFunc.Fn([]any{ts})
		require.NoError(t, err)
		assert.Equal(t, 2024, result)
	})

	t.Run("UnixTimestampFloat", func(t *testing.T) {
		ts := float64(1718409600.5)
		result, err := yearFunc.Fn([]any{ts})
		require.NoError(t, err)
		assert.Equal(t, 2024, result)
	})

	t.Run("DateString", func(t *testing.T) {
		result, err := yearFunc.Fn([]any{"2024-06-15"})
		require.NoError(t, err)
		assert.Equal(t, 2024, result)
	})

	t.Run("InvalidType", func(t *testing.T) {
		_, err := yearFunc.Fn([]any{[]string{"not", "a", "time"}})
		require.Error(t, err)
	})

	t.Run("NilValue", func(t *testing.T) {
		_, err := yearFunc.Fn([]any{nil})
		require.Error(t, err)
	})
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestDateTime_E2E_ErrorHandling(t *testing.T) {
	registry := createDateTimeRegistry()

	t.Run("FormatDate_WrongArgCount", func(t *testing.T) {
		// Use registry.Call which validates argument counts
		_, err := registry.Call(FuncNameFormatDate, []any{time.Now()})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "few")
	})

	t.Run("FormatDate_InvalidTimeArg", func(t *testing.T) {
		_, err := registry.Call(FuncNameFormatDate, []any{"not a time", DateFormatISO})
		require.Error(t, err)
	})

	t.Run("FormatDate_InvalidLayoutArg", func(t *testing.T) {
		_, err := registry.Call(FuncNameFormatDate, []any{time.Now(), 12345})
		require.Error(t, err)
	})

	t.Run("ParseDate_InvalidString", func(t *testing.T) {
		_, err := registry.Call(FuncNameParseDate, []any{"definitely-not-a-date"})
		require.Error(t, err)
	})

	t.Run("AddDays_NonIntegerDays", func(t *testing.T) {
		_, err := registry.Call(FuncNameAddDays, []any{time.Now(), "five"})
		require.Error(t, err)
	})

	t.Run("DiffDays_InvalidFirstArg", func(t *testing.T) {
		_, err := registry.Call(FuncNameDiffDays, []any{"not time", time.Now()})
		require.Error(t, err)
	})

	t.Run("DiffDays_InvalidSecondArg", func(t *testing.T) {
		_, err := registry.Call(FuncNameDiffDays, []any{time.Now(), "not time"})
		require.Error(t, err)
	})
}
