package internal

import (
	"fmt"
	"reflect"
	"strconv"
)

// registerTypeFuncs registers type conversion and inspection functions
func registerTypeFuncs(r *FuncRegistry) {
	// toString(x any) string
	r.MustRegister(&Func{
		Name:    FuncNameToString,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			return anyToString(args[0]), nil
		},
	})

	// toInt(x any) int
	r.MustRegister(&Func{
		Name:    FuncNameToInt,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			return anyToInt(args[ArgIndexFirst], FuncNameToInt, ArgIndexFirst)
		},
	})

	// toFloat(x any) float64
	r.MustRegister(&Func{
		Name:    FuncNameToFloat,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			return anyToFloat(args[ArgIndexFirst], FuncNameToFloat, ArgIndexFirst)
		},
	})

	// toBool(x any) bool
	r.MustRegister(&Func{
		Name:    FuncNameToBool,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			return isTruthy(args[0]), nil
		},
	})

	// typeOf(x any) string
	r.MustRegister(&Func{
		Name:    FuncNameTypeOf,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			if args[ArgIndexFirst] == nil {
				return StringValueNil, nil
			}
			return reflect.TypeOf(args[ArgIndexFirst]).String(), nil
		},
	})

	// isNil(x any) bool
	r.MustRegister(&Func{
		Name:    FuncNameIsNil,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			return args[0] == nil, nil
		},
	})

	// isEmpty(x any) bool
	r.MustRegister(&Func{
		Name:    FuncNameIsEmpty,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			return isEmpty(args[0]), nil
		},
	})
}

// toString attempts to convert any value to a string
func toString(v any) (string, bool) {
	if v == nil {
		return "", true
	}
	switch val := v.(type) {
	case string:
		return val, true
	case fmt.Stringer:
		return val.String(), true
	default:
		return "", false
	}
}

// anyToString converts any value to its string representation
func anyToString(v any) string {
	if v == nil {
		return StringValueEmpty
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return StringValueTrue
		}
		return StringValueFalse
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, IntBase10)
	case float64:
		return strconv.FormatFloat(val, FloatFormatFlag, FloatPrecisionAll, FloatBitSize64)
	case fmt.Stringer:
		return val.String()
	default:
		return fmt.Sprintf("%v", val)
	}
}

// anyToInt converts any value to an integer
func anyToInt(v any, funcName string, argIndex int) (int, error) {
	if v == nil {
		return 0, nil
	}
	switch val := v.(type) {
	case int:
		return val, nil
	case int64:
		return int(val), nil
	case float64:
		return int(val), nil
	case string:
		n, err := strconv.Atoi(val)
		if err != nil {
			return 0, NewFuncTypeError(ErrMsgFuncConversionFailed, funcName, argIndex)
		}
		return n, nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, NewFuncTypeError(ErrMsgFuncConversionFailed, funcName, argIndex)
	}
}

// anyToFloat converts any value to a float64
func anyToFloat(v any, funcName string, argIndex int) (float64, error) {
	if v == nil {
		return 0, nil
	}
	switch val := v.(type) {
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		f, err := strconv.ParseFloat(val, FloatBitSize64)
		if err != nil {
			return 0, NewFuncTypeError(ErrMsgFuncConversionFailed, funcName, argIndex)
		}
		return f, nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, NewFuncTypeError(ErrMsgFuncConversionFailed, funcName, argIndex)
	}
}

// isTruthy determines the truthiness of a value
// Truthiness rules:
// - nil -> false
// - bool -> value
// - string -> len(s) > 0
// - int/float -> n != 0
// - slice/map -> len(x) > 0
func isTruthy(v any) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return len(val) > 0
	case int:
		return val != 0
	case int64:
		return val != 0
	case float64:
		return val != 0
	case []any:
		return len(val) > 0
	case []string:
		return len(val) > 0
	case map[string]any:
		return len(val) > 0
	default:
		// Use reflection for other types
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map:
			return rv.Len() > 0
		case reflect.Ptr, reflect.Interface:
			return !rv.IsNil()
		default:
			return true // Non-nil values are generally truthy
		}
	}
}

// isEmpty checks if a value is empty
func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return len(val) == 0
	case []any:
		return len(val) == 0
	case []string:
		return len(val) == 0
	case map[string]any:
		return len(val) == 0
	default:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
			return rv.Len() == 0
		default:
			return false
		}
	}
}
