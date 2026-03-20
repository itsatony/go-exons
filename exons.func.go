package exons

import (
	"github.com/itsatony/go-exons/internal"
)

// Func represents a custom function that can be used in expressions.
// Functions are called within conditional expressions, switch evaluations,
// and other expression contexts.
type Func struct {
	// Name is the function identifier used in expressions (e.g., "myFunc" for myFunc(x))
	Name string
	// MinArgs is the minimum number of arguments required
	MinArgs int
	// MaxArgs is the maximum number of arguments allowed (-1 for variadic)
	MaxArgs int
	// Fn is the function implementation
	Fn func(args []any) (any, error)
}

// RegisterFunc registers a custom function for use in expressions.
// Custom functions can be called within eval expressions, conditionals, and switch statements.
//
// Example:
//
//	engine.RegisterFunc(&exons.Func{
//	    Name:    "double",
//	    MinArgs: 1,
//	    MaxArgs: 1,
//	    Fn: func(args []any) (any, error) {
//	        if n, ok := args[0].(int); ok {
//	            return n * 2, nil
//	        }
//	        return nil, errors.New("expected int argument")
//	    },
//	})
//
// The function can then be used in templates:
//
//	{~exons.if eval="double(count) > 10"~}...{~/exons.if~}
func (e *Engine) RegisterFunc(f *Func) error {
	if f == nil {
		return NewFuncRegistrationError(ErrMsgFuncNilFunc, "")
	}
	if f.Name == "" {
		return NewFuncRegistrationError(ErrMsgFuncEmptyName, "")
	}

	// Convert to internal Func
	internalFunc := &internal.Func{
		Name:    f.Name,
		MinArgs: f.MinArgs,
		MaxArgs: f.MaxArgs,
		Fn:      f.Fn,
	}

	return e.executor.RegisterFunc(internalFunc)
}

// MustRegisterFunc registers a custom function and panics on error.
func (e *Engine) MustRegisterFunc(f *Func) {
	if err := e.RegisterFunc(f); err != nil {
		panic(err)
	}
}

// HasFunc checks if a function is registered with the given name.
func (e *Engine) HasFunc(name string) bool {
	return e.executor.HasFunc(name)
}

// ListFuncs returns all registered function names.
func (e *Engine) ListFuncs() []string {
	return e.executor.ListFuncs()
}

// FuncCount returns the number of registered functions.
func (e *Engine) FuncCount() int {
	return e.executor.FuncCount()
}
