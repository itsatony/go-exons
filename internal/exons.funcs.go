package internal

import (
	"fmt"
	"sync"
)

// Func represents a callable function in expressions
type Func struct {
	Name    string
	MinArgs int
	MaxArgs int // -1 for variadic
	Fn      func(args []any) (any, error)
}

// FuncRegistry manages registered functions
type FuncRegistry struct {
	funcs map[string]*Func
	mu    sync.RWMutex
}

// NewFuncRegistry creates a new function registry
func NewFuncRegistry() *FuncRegistry {
	return &FuncRegistry{
		funcs: make(map[string]*Func),
	}
}

// Register adds a function to the registry
func (r *FuncRegistry) Register(f *Func) error {
	if f == nil {
		return NewFuncRegistryError(ErrMsgFuncNilFunc, "")
	}
	if f.Name == "" {
		return NewFuncRegistryError(ErrMsgFuncEmptyName, "")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.funcs[f.Name]; exists {
		return NewFuncRegistryError(ErrMsgFuncAlreadyExists, f.Name)
	}

	r.funcs[f.Name] = f
	return nil
}

// MustRegister adds a function and panics on error
func (r *FuncRegistry) MustRegister(f *Func) {
	if err := r.Register(f); err != nil {
		panic(err)
	}
}

// Get retrieves a function by name
func (r *FuncRegistry) Get(name string) (*Func, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f, ok := r.funcs[name]
	return f, ok
}

// Has checks if a function is registered
func (r *FuncRegistry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.funcs[name]
	return ok
}

// Call invokes a function by name with the given arguments
func (r *FuncRegistry) Call(name string, args []any) (any, error) {
	r.mu.RLock()
	f, ok := r.funcs[name]
	r.mu.RUnlock()

	if !ok {
		return nil, NewFuncError(ErrMsgFuncNotFound, name)
	}

	// Check argument count
	argCount := len(args)
	if argCount < f.MinArgs {
		return nil, NewFuncArgError(ErrMsgFuncTooFewArgs, name, f.MinArgs, argCount)
	}
	if f.MaxArgs >= 0 && argCount > f.MaxArgs {
		return nil, NewFuncArgError(ErrMsgFuncTooManyArgs, name, f.MaxArgs, argCount)
	}

	// Call the function
	result, err := f.Fn(args)
	if err != nil {
		return nil, NewFuncExecError(name, err)
	}

	return result, nil
}

// List returns all registered function names
func (r *FuncRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.funcs))
	for name := range r.funcs {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered functions
func (r *FuncRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.funcs)
}

// FuncRegistryError represents a function registry error
type FuncRegistryError struct {
	Message  string
	FuncName string
}

// NewFuncRegistryError creates a new function registry error
func NewFuncRegistryError(message, funcName string) *FuncRegistryError {
	return &FuncRegistryError{
		Message:  message,
		FuncName: funcName,
	}
}

// Error implements the error interface
func (e *FuncRegistryError) Error() string {
	if e.FuncName != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.FuncName)
	}
	return e.Message
}

// FuncError represents a function-related error
type FuncError struct {
	Message  string
	FuncName string
}

// NewFuncError creates a new function error
func NewFuncError(message, funcName string) *FuncError {
	return &FuncError{
		Message:  message,
		FuncName: funcName,
	}
}

// Error implements the error interface
func (e *FuncError) Error() string {
	return fmt.Sprintf("%s: %s", e.Message, e.FuncName)
}

// FuncArgError represents a function argument error
type FuncArgError struct {
	Message  string
	FuncName string
	Expected int
	Actual   int
}

// NewFuncArgError creates a new function argument error
func NewFuncArgError(message, funcName string, expected, actual int) *FuncArgError {
	return &FuncArgError{
		Message:  message,
		FuncName: funcName,
		Expected: expected,
		Actual:   actual,
	}
}

// Error implements the error interface
func (e *FuncArgError) Error() string {
	return fmt.Sprintf("%s: %s (expected %d, got %d)", e.Message, e.FuncName, e.Expected, e.Actual)
}

// FuncExecError represents a function execution error
type FuncExecError struct {
	FuncName string
	Cause    error
}

// NewFuncExecError creates a new function execution error
func NewFuncExecError(funcName string, cause error) *FuncExecError {
	return &FuncExecError{
		FuncName: funcName,
		Cause:    cause,
	}
}

// Error implements the error interface
func (e *FuncExecError) Error() string {
	return fmt.Sprintf("function %s failed: %v", e.FuncName, e.Cause)
}

// Unwrap returns the underlying error
func (e *FuncExecError) Unwrap() error {
	return e.Cause
}

// Function error messages
const (
	ErrMsgFuncNilFunc           = "function cannot be nil"
	ErrMsgFuncEmptyName         = "function name cannot be empty"
	ErrMsgFuncAlreadyExists     = "function already registered"
	ErrMsgFuncNotFound          = "function not found"
	ErrMsgFuncTooFewArgs        = "too few arguments"
	ErrMsgFuncTooManyArgs       = "too many arguments"
	ErrMsgFuncExpectedString    = "expected string argument"
	ErrMsgFuncExpectedSlice     = "expected slice or array argument"
	ErrMsgFuncExpectedMap       = "expected map argument"
	ErrMsgFuncExpectedStringKey = "expected string key"
	ErrMsgFuncConversionFailed  = "type conversion failed"
)

// FuncTypeError represents a type error in function arguments
type FuncTypeError struct {
	Message  string
	FuncName string
	ArgIndex int
}

// NewFuncTypeError creates a new function type error
func NewFuncTypeError(message, funcName string, argIndex int) *FuncTypeError {
	return &FuncTypeError{
		Message:  message,
		FuncName: funcName,
		ArgIndex: argIndex,
	}
}

// Error implements the error interface
func (e *FuncTypeError) Error() string {
	return fmt.Sprintf("%s: %s (argument %d)", e.Message, e.FuncName, e.ArgIndex)
}

// Built-in function names
const (
	FuncNameLen        = "len"
	FuncNameContains   = "contains"
	FuncNameUpper      = "upper"
	FuncNameLower      = "lower"
	FuncNameTrim       = "trim"
	FuncNameTrimPrefix = "trimPrefix"
	FuncNameTrimSuffix = "trimSuffix"
	FuncNameHasPrefix  = "hasPrefix"
	FuncNameHasSuffix  = "hasSuffix"
	FuncNameReplace    = "replace"
	FuncNameSplit      = "split"
	FuncNameJoin       = "join"
	FuncNameFirst      = "first"
	FuncNameLast       = "last"
	FuncNameKeys       = "keys"
	FuncNameValues     = "values"
	FuncNameHas        = "has"
	FuncNameToString   = "toString"
	FuncNameToInt      = "toInt"
	FuncNameToFloat    = "toFloat"
	FuncNameToBool     = "toBool"
	FuncNameTypeOf     = "typeOf"
	FuncNameIsNil      = "isNil"
	FuncNameIsEmpty    = "isEmpty"
	FuncNameDefault    = "default"
	FuncNameCoalesce   = "coalesce"
)

// String value constants for type conversions
const (
	StringValueNil   = "nil"
	StringValueTrue  = "true"
	StringValueFalse = "false"
	// StringValueEmpty is declared in exons.constants.go
)

// Numeric constants for conversions
const (
	FloatFormatFlag   = 'f'
	FloatPrecisionAll = -1
	FloatBitSize64    = 64
	IntBase10         = 10
)

// RegisterBuiltinFuncs registers all built-in functions with the registry
func RegisterBuiltinFuncs(r *FuncRegistry) {
	registerStringFuncs(r)
	registerCollectionFuncs(r)
	registerTypeFuncs(r)
	registerUtilFuncs(r)
	registerDateTimeFuncs(r)
}
