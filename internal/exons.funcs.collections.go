package internal

import (
	"reflect"
	"sort"
)

// Collection function error messages (unique to collections)
const (
	ErrMsgFuncIndexRange = "index out of range"
)

// registerCollectionFuncs registers collection manipulation functions
func registerCollectionFuncs(r *FuncRegistry) {
	// len(x any) int - returns length of string, slice, or map
	r.MustRegister(&Func{
		Name:    FuncNameLen,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			return getLength(args[ArgIndexFirst], FuncNameLen, ArgIndexFirst)
		},
	})

	// first(x []any) any - returns first element
	r.MustRegister(&Func{
		Name:    FuncNameFirst,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			slice, err := toSlice(args[ArgIndexFirst], FuncNameFirst, ArgIndexFirst)
			if err != nil {
				return nil, err
			}
			if len(slice) == 0 {
				return nil, nil
			}
			return slice[0], nil
		},
	})

	// last(x []any) any - returns last element
	r.MustRegister(&Func{
		Name:    FuncNameLast,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			slice, err := toSlice(args[ArgIndexFirst], FuncNameLast, ArgIndexFirst)
			if err != nil {
				return nil, err
			}
			if len(slice) == 0 {
				return nil, nil
			}
			return slice[len(slice)-1], nil
		},
	})

	// keys(m map[string]any) []string - returns map keys
	r.MustRegister(&Func{
		Name:    FuncNameKeys,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			m, ok := args[ArgIndexFirst].(map[string]any)
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedMap, FuncNameKeys, ArgIndexFirst)
			}
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			return keys, nil
		},
	})

	// values(m map[string]any) []any - returns map values
	r.MustRegister(&Func{
		Name:    FuncNameValues,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			m, ok := args[ArgIndexFirst].(map[string]any)
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedMap, FuncNameValues, ArgIndexFirst)
			}
			// Get keys in sorted order for deterministic output
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			values := make([]any, len(m))
			for i, k := range keys {
				values[i] = m[k]
			}
			return values, nil
		},
	})

	// has(m map[string]any, key string) bool - checks if map has key
	r.MustRegister(&Func{
		Name:    FuncNameHas,
		MinArgs: 2,
		MaxArgs: 2,
		Fn: func(args []any) (any, error) {
			m, ok := args[ArgIndexFirst].(map[string]any)
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedMap, FuncNameHas, ArgIndexFirst)
			}
			key, ok := toString(args[ArgIndexSecond])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedStringKey, FuncNameHas, ArgIndexSecond)
			}
			_, exists := m[key]
			return exists, nil
		},
	})
}

// getLength returns the length of various types
func getLength(v any, funcName string, argIndex int) (int, error) {
	if v == nil {
		return 0, nil
	}

	switch val := v.(type) {
	case string:
		return len(val), nil
	case []any:
		return len(val), nil
	case []string:
		return len(val), nil
	case []int:
		return len(val), nil
	case []float64:
		return len(val), nil
	case map[string]any:
		return len(val), nil
	case map[string]string:
		return len(val), nil
	default:
		// Use reflection for other slice/map types
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
			return rv.Len(), nil
		default:
			return 0, NewFuncTypeError(ErrMsgFuncExpectedSlice, funcName, argIndex)
		}
	}
}

// toSlice converts various slice types to []any
func toSlice(v any, funcName string, argIndex int) ([]any, error) {
	if v == nil {
		return nil, nil
	}

	switch val := v.(type) {
	case []any:
		return val, nil
	case []string:
		result := make([]any, len(val))
		for i, s := range val {
			result[i] = s
		}
		return result, nil
	case []int:
		result := make([]any, len(val))
		for i, n := range val {
			result[i] = n
		}
		return result, nil
	case []float64:
		result := make([]any, len(val))
		for i, n := range val {
			result[i] = n
		}
		return result, nil
	default:
		// Use reflection for other slice types
		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
			return nil, NewFuncTypeError(ErrMsgFuncExpectedSlice, funcName, argIndex)
		}
		result := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			result[i] = rv.Index(i).Interface()
		}
		return result, nil
	}
}
