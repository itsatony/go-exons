package internal

import (
	"strings"
)

// Argument index constants for error reporting
const (
	ArgIndexFirst  = 0
	ArgIndexSecond = 1
	ArgIndexThird  = 2
)

// registerStringFuncs registers string manipulation functions
func registerStringFuncs(r *FuncRegistry) {
	// upper(s string) string
	r.MustRegister(&Func{
		Name:    FuncNameUpper,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			s, ok := toString(args[ArgIndexFirst])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameUpper, ArgIndexFirst)
			}
			return strings.ToUpper(s), nil
		},
	})

	// lower(s string) string
	r.MustRegister(&Func{
		Name:    FuncNameLower,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			s, ok := toString(args[ArgIndexFirst])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameLower, ArgIndexFirst)
			}
			return strings.ToLower(s), nil
		},
	})

	// trim(s string) string
	r.MustRegister(&Func{
		Name:    FuncNameTrim,
		MinArgs: 1,
		MaxArgs: 1,
		Fn: func(args []any) (any, error) {
			s, ok := toString(args[ArgIndexFirst])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameTrim, ArgIndexFirst)
			}
			return strings.TrimSpace(s), nil
		},
	})

	// trimPrefix(s, prefix string) string
	r.MustRegister(&Func{
		Name:    FuncNameTrimPrefix,
		MinArgs: 2,
		MaxArgs: 2,
		Fn: func(args []any) (any, error) {
			s, ok := toString(args[ArgIndexFirst])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameTrimPrefix, ArgIndexFirst)
			}
			prefix, ok := toString(args[ArgIndexSecond])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameTrimPrefix, ArgIndexSecond)
			}
			return strings.TrimPrefix(s, prefix), nil
		},
	})

	// trimSuffix(s, suffix string) string
	r.MustRegister(&Func{
		Name:    FuncNameTrimSuffix,
		MinArgs: 2,
		MaxArgs: 2,
		Fn: func(args []any) (any, error) {
			s, ok := toString(args[ArgIndexFirst])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameTrimSuffix, ArgIndexFirst)
			}
			suffix, ok := toString(args[ArgIndexSecond])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameTrimSuffix, ArgIndexSecond)
			}
			return strings.TrimSuffix(s, suffix), nil
		},
	})

	// hasPrefix(s, prefix string) bool
	r.MustRegister(&Func{
		Name:    FuncNameHasPrefix,
		MinArgs: 2,
		MaxArgs: 2,
		Fn: func(args []any) (any, error) {
			s, ok := toString(args[ArgIndexFirst])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameHasPrefix, ArgIndexFirst)
			}
			prefix, ok := toString(args[ArgIndexSecond])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameHasPrefix, ArgIndexSecond)
			}
			return strings.HasPrefix(s, prefix), nil
		},
	})

	// hasSuffix(s, suffix string) bool
	r.MustRegister(&Func{
		Name:    FuncNameHasSuffix,
		MinArgs: 2,
		MaxArgs: 2,
		Fn: func(args []any) (any, error) {
			s, ok := toString(args[ArgIndexFirst])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameHasSuffix, ArgIndexFirst)
			}
			suffix, ok := toString(args[ArgIndexSecond])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameHasSuffix, ArgIndexSecond)
			}
			return strings.HasSuffix(s, suffix), nil
		},
	})

	// contains(s, substr string) bool - also handles slice contains
	r.MustRegister(&Func{
		Name:    FuncNameContains,
		MinArgs: 2,
		MaxArgs: 2,
		Fn: func(args []any) (any, error) {
			// Handle both string contains and slice contains
			switch v := args[ArgIndexFirst].(type) {
			case string:
				substr, ok := toString(args[ArgIndexSecond])
				if !ok {
					return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameContains, ArgIndexSecond)
				}
				return strings.Contains(v, substr), nil
			case []any:
				for _, item := range v {
					if item == args[ArgIndexSecond] {
						return true, nil
					}
				}
				return false, nil
			case []string:
				search, ok := toString(args[ArgIndexSecond])
				if !ok {
					return false, nil
				}
				for _, item := range v {
					if item == search {
						return true, nil
					}
				}
				return false, nil
			default:
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameContains, ArgIndexFirst)
			}
		},
	})

	// replace(s, old, new string) string
	r.MustRegister(&Func{
		Name:    FuncNameReplace,
		MinArgs: 3,
		MaxArgs: 3,
		Fn: func(args []any) (any, error) {
			s, ok := toString(args[ArgIndexFirst])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameReplace, ArgIndexFirst)
			}
			old, ok := toString(args[ArgIndexSecond])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameReplace, ArgIndexSecond)
			}
			newStr, ok := toString(args[ArgIndexThird])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameReplace, ArgIndexThird)
			}
			return strings.ReplaceAll(s, old, newStr), nil
		},
	})

	// split(s, sep string) []string
	r.MustRegister(&Func{
		Name:    FuncNameSplit,
		MinArgs: 2,
		MaxArgs: 2,
		Fn: func(args []any) (any, error) {
			s, ok := toString(args[ArgIndexFirst])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameSplit, ArgIndexFirst)
			}
			sep, ok := toString(args[ArgIndexSecond])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameSplit, ArgIndexSecond)
			}
			return strings.Split(s, sep), nil
		},
	})

	// join(items []string, sep string) string
	r.MustRegister(&Func{
		Name:    FuncNameJoin,
		MinArgs: 2,
		MaxArgs: 2,
		Fn: func(args []any) (any, error) {
			sep, ok := toString(args[ArgIndexSecond])
			if !ok {
				return nil, NewFuncTypeError(ErrMsgFuncExpectedString, FuncNameJoin, ArgIndexSecond)
			}

			switch v := args[ArgIndexFirst].(type) {
			case []string:
				return strings.Join(v, sep), nil
			case []any:
				strs := make([]string, len(v))
				for i, item := range v {
					s, ok := toString(item)
					if !ok {
						s = StringValueEmpty
					}
					strs[i] = s
				}
				return strings.Join(strs, sep), nil
			default:
				return nil, NewFuncTypeError(ErrMsgFuncExpectedSlice, FuncNameJoin, ArgIndexFirst)
			}
		},
	})
}
