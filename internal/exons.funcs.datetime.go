package internal

import (
	"time"
)

// Date/time function name constants
const (
	FuncNameNow        = "now"
	FuncNameFormatDate = "formatDate"
	FuncNameParseDate  = "parseDate"
	FuncNameAddDays    = "addDays"
	FuncNameAddHours   = "addHours"
	FuncNameAddMinutes = "addMinutes"
	FuncNameDiffDays   = "diffDays"
	FuncNameYear       = "year"
	FuncNameMonth      = "month"
	FuncNameDay        = "day"
	FuncNameWeekday    = "weekday"
	FuncNameIsAfter    = "isAfter"
	FuncNameIsBefore   = "isBefore"
)

// Error messages for date/time functions
const (
	ErrMsgFuncExpectedTime       = "expected time argument"
	ErrMsgFuncExpectedTimeLayout = "expected date format layout string"
	ErrMsgFuncInvalidTimeFormat  = "invalid time format"
	ErrMsgFuncExpectedInteger    = "expected integer argument"
)

// Common date format constants for documentation and examples
const (
	DateFormatISO     = "2006-01-02"
	DateFormatUS      = "01/02/2006"
	DateFormatEU      = "02/01/2006"
	DateTimeFormatISO = "2006-01-02T15:04:05Z07:00"
	TimeFormat24H     = "15:04:05"
	TimeFormat12H     = "3:04:05 PM"
)

// Common time parsing formats tried in order
var commonTimeFormats = []string{
	time.RFC3339,
	time.RFC3339Nano,
	DateTimeFormatISO,
	DateFormatISO,
	time.RFC1123,
	time.RFC1123Z,
	time.RFC822,
	time.RFC822Z,
	DateFormatUS,
	DateFormatEU,
	"2006-01-02 15:04:05",
	"2006/01/02",
	"02-01-2006",
	"Jan 2, 2006",
	"January 2, 2006",
}

// Time component constants
const (
	HoursPerDay    = 24
	MinutesPerHour = 60
)

// registerDateTimeFuncs registers date/time manipulation functions
func registerDateTimeFuncs(r *FuncRegistry) {
	// now() time.Time - returns current timestamp
	r.MustRegister(&Func{
		Name:    FuncNameNow,
		MinArgs: 0,
		MaxArgs: 0,
		Fn: func(args []any) (any, error) {
			return time.Now(), nil
		},
	})

	// formatDate(t time.Time, layout string) string
	r.MustRegister(&Func{
		Name:    FuncNameFormatDate,
		MinArgs: 2,
		MaxArgs: 2,
		Fn: func(args []any) (any, error) {
			t, err := toTime(args[ArgIndexFirst])
			if err != nil {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedTime, FuncNameFormatDate, ArgIndexFirst)
			}
			layout, ok := toString(args[ArgIndexSecond])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedTimeLayout, FuncNameFormatDate, ArgIndexSecond)
			}
			return t.Format(layout), nil
		},
	})

	// parseDate(s string, layout string) time.Time
	r.MustRegister(&Func{
		Name:    FuncNameParseDate,
		MinArgs: 1,
		MaxArgs: 2,
		Fn: func(args []any) (any, error) {
			s, ok := toString(args[ArgIndexFirst])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameParseDate, ArgIndexFirst)
			}

			// If layout provided, use it
			if len(args) > 1 {
				layout, ok := toString(args[ArgIndexSecond])
				if !ok {
					return nil, NewFuncTypeError(ErrMsgFuncExpectedTimeLayout, FuncNameParseDate, ArgIndexSecond)
				}
				t, err := time.Parse(layout, s)
				if err != nil {
					return nil, NewFuncTypeError(ErrMsgFuncInvalidTimeFormat, FuncNameParseDate, ArgIndexFirst)
				}
				return t, nil
			}

			// Try common formats
			t, err := parseTimeString(s)
			if err != nil {
				return nil, NewFuncTypeError(ErrMsgFuncInvalidTimeFormat, FuncNameParseDate, ArgIndexFirst)
			}
			return t, nil
		},
	})

	// addDays(t time.Time, n int) time.Time
	r.MustRegister(&Func{
		Name:    FuncNameAddDays,
		MinArgs: 2,
		MaxArgs: 2,
		Fn: func(args []any) (any, error) {
			t, err := toTime(args[ArgIndexFirst])
			if err != nil {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedTime, FuncNameAddDays, ArgIndexFirst)
			}
			n, err := anyToInt(args[ArgIndexSecond], FuncNameAddDays, ArgIndexSecond)
			if err != nil {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedInteger, FuncNameAddDays, ArgIndexSecond)
			}
			return t.AddDate(0, 0, n), nil
		},
	})

	// addHours(t time.Time, n int) time.Time
	r.MustRegister(&Func{
		Name:    FuncNameAddHours,
		MinArgs: 2,
		MaxArgs: 2,
		Fn: func(args []any) (any, error) {
			t, err := toTime(args[ArgIndexFirst])
			if err != nil {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedTime, FuncNameAddHours, ArgIndexFirst)
			}
			n, err := anyToInt(args[ArgIndexSecond], FuncNameAddHours, ArgIndexSecond)
			if err != nil {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedInteger, FuncNameAddHours, ArgIndexSecond)
			}
			return t.Add(time.Duration(n) * time.Hour), nil
		},
	})

	// addMinutes(t time.Time, n int) time.Time
	r.MustRegister(&Func{
		Name:    FuncNameAddMinutes,
		MinArgs: 2,
		MaxArgs: 2,
		Fn: func(args []any) (any, error) {
			t, err := toTime(args[ArgIndexFirst])
			if err != nil {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedTime, FuncNameAddMinutes, ArgIndexFirst)
			}
			n, err := anyToInt(args[ArgIndexSecond], FuncNameAddMinutes, ArgIndexSecond)
			if err != nil {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedInteger, FuncNameAddMinutes, ArgIndexSecond)
			}
			return t.Add(time.Duration(n) * time.Minute), nil
		},
	})

	// diffDays(t1 time.Time, t2 time.Time) int - returns days between t1 and t2
	r.MustRegister(&Func{
		Name:    FuncNameDiffDays,
		MinArgs: 2,
		MaxArgs: 2,
		Fn: func(args []any) (any, error) {
			t1, err := toTime(args[ArgIndexFirst])
			if err != nil {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedTime, FuncNameDiffDays, ArgIndexFirst)
			}
			t2, err := toTime(args[ArgIndexSecond])
			if err != nil {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedTime, FuncNameDiffDays, ArgIndexSecond)
			}
			diff := t2.Sub(t1)
			return int(diff.Hours() / HoursPerDay), nil
		},
	})

	// year(t time.Time) int
	r.MustRegister(&Func{
		Name:    FuncNameYear,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			t, err := toTime(args[ArgIndexFirst])
			if err != nil {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedTime, FuncNameYear, ArgIndexFirst)
			}
			return t.Year(), nil
		},
	})

	// month(t time.Time) int (1-12)
	r.MustRegister(&Func{
		Name:    FuncNameMonth,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			t, err := toTime(args[ArgIndexFirst])
			if err != nil {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedTime, FuncNameMonth, ArgIndexFirst)
			}
			return int(t.Month()), nil
		},
	})

	// day(t time.Time) int (1-31)
	r.MustRegister(&Func{
		Name:    FuncNameDay,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			t, err := toTime(args[ArgIndexFirst])
			if err != nil {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedTime, FuncNameDay, ArgIndexFirst)
			}
			return t.Day(), nil
		},
	})

	// weekday(t time.Time) string (Monday, Tuesday, etc.)
	r.MustRegister(&Func{
		Name:    FuncNameWeekday,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			t, err := toTime(args[ArgIndexFirst])
			if err != nil {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedTime, FuncNameWeekday, ArgIndexFirst)
			}
			return t.Weekday().String(), nil
		},
	})

	// isAfter(t1 time.Time, t2 time.Time) bool - returns true if t1 is after t2
	r.MustRegister(&Func{
		Name:    FuncNameIsAfter,
		MinArgs: 2,
		MaxArgs: 2,
		Fn: func(args []any) (any, error) {
			t1, err := toTime(args[ArgIndexFirst])
			if err != nil {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedTime, FuncNameIsAfter, ArgIndexFirst)
			}
			t2, err := toTime(args[ArgIndexSecond])
			if err != nil {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedTime, FuncNameIsAfter, ArgIndexSecond)
			}
			return t1.After(t2), nil
		},
	})

	// isBefore(t1 time.Time, t2 time.Time) bool - returns true if t1 is before t2
	r.MustRegister(&Func{
		Name:    FuncNameIsBefore,
		MinArgs: 2,
		MaxArgs: 2,
		Fn: func(args []any) (any, error) {
			t1, err := toTime(args[ArgIndexFirst])
			if err != nil {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedTime, FuncNameIsBefore, ArgIndexFirst)
			}
			t2, err := toTime(args[ArgIndexSecond])
			if err != nil {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedTime, FuncNameIsBefore, ArgIndexSecond)
			}
			return t1.Before(t2), nil
		},
	})
}

// toTime attempts to convert various types to time.Time
func toTime(v any) (time.Time, error) {
	if v == nil {
		return time.Time{}, NewFuncTypeError(ErrMsgFuncExpectedTime, "", 0)
	}

	switch t := v.(type) {
	case time.Time:
		return t, nil
	case *time.Time:
		if t == nil {
			return time.Time{}, NewFuncTypeError(ErrMsgFuncExpectedTime, "", 0)
		}
		return *t, nil
	case string:
		return parseTimeString(t)
	case int64:
		// Treat as Unix timestamp
		return time.Unix(t, 0), nil
	case int:
		// Treat as Unix timestamp
		return time.Unix(int64(t), 0), nil
	case float64:
		// Treat as Unix timestamp with fractional seconds
		sec := int64(t)
		nsec := int64((t - float64(sec)) * 1e9)
		return time.Unix(sec, nsec), nil
	default:
		return time.Time{}, NewFuncTypeError(ErrMsgFuncExpectedTime, "", 0)
	}
}

// parseTimeString attempts to parse a string into time.Time using common formats
func parseTimeString(s string) (time.Time, error) {
	for _, format := range commonTimeFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, NewFuncTypeError(ErrMsgFuncInvalidTimeFormat, "", 0)
}
