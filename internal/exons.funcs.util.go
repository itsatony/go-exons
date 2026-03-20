package internal

// registerUtilFuncs registers utility functions
func registerUtilFuncs(r *FuncRegistry) {
	// default(x any, fallback any) any - returns fallback if x is nil or empty
	r.MustRegister(&Func{
		Name:    FuncNameDefault,
		MinArgs: 2,
		MaxArgs: 2,
		Fn: func(args []any) (any, error) {
			if args[0] == nil || isEmpty(args[0]) {
				return args[1], nil
			}
			return args[0], nil
		},
	})

	// coalesce(args ...any) any - returns first non-nil, non-empty value
	r.MustRegister(&Func{
		Name:    FuncNameCoalesce,
		MinArgs: 1,
		MaxArgs: -1, // Variadic
		Fn: func(args []any) (any, error) {
			for _, arg := range args {
				if arg != nil && !isEmpty(arg) {
					return arg, nil
				}
			}
			return nil, nil
		},
	})
}
