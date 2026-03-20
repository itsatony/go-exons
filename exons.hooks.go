package exons

import (
	"context"
	"sync"
	"time"
)

// HookPoint identifies when a hook is called during template operations.
type HookPoint string

// Hook points for template lifecycle events.
const (
	// HookBeforeLoad is called before loading a template from storage.
	HookBeforeLoad HookPoint = "before_load"

	// HookAfterLoad is called after successfully loading a template.
	HookAfterLoad HookPoint = "after_load"

	// HookBeforeExecute is called before executing a template.
	HookBeforeExecute HookPoint = "before_execute"

	// HookAfterExecute is called after template execution (success or failure).
	HookAfterExecute HookPoint = "after_execute"

	// HookBeforeSave is called before saving a template.
	HookBeforeSave HookPoint = "before_save"

	// HookAfterSave is called after successfully saving a template.
	HookAfterSave HookPoint = "after_save"

	// HookBeforeDelete is called before deleting a template.
	HookBeforeDelete HookPoint = "before_delete"

	// HookAfterDelete is called after successfully deleting a template.
	HookAfterDelete HookPoint = "after_delete"

	// HookBeforeValidate is called before validating a template.
	HookBeforeValidate HookPoint = "before_validate"

	// HookAfterValidate is called after template validation.
	HookAfterValidate HookPoint = "after_validate"
)

// Hook is a function called at specific points during template operations.
// Return an error to abort the operation (for "before" hooks).
// Errors from "after" hooks are logged but do not affect the operation result.
type Hook func(ctx context.Context, point HookPoint, data *HookData) error

// HookData carries context information to hooks.
// Simplified from go-prompty: removed Subject, Template (StoredTemplate), and Operation fields.
// Added OperationName as a string for lightweight operation identification.
type HookData struct {
	// OperationName identifies the operation being performed (e.g., "load", "execute", "save").
	OperationName string

	// TemplateName is the name of the template.
	TemplateName string

	// ExecutionData is the data passed to template execution (for execute operations).
	ExecutionData map[string]any

	// Result is the execution result (for after_execute, may be empty on error).
	Result string

	// Error is any error that occurred (for after_* hooks).
	Error error

	// ValidationResult contains validation results (for after_validate).
	ValidationResult *ValidationResult

	// Metadata allows hooks to pass data to each other.
	Metadata map[string]any
}

// NewHookData creates a new HookData with the given parameters.
func NewHookData(operationName, templateName string) *HookData {
	return &HookData{
		OperationName: operationName,
		TemplateName:  templateName,
		Metadata:      make(map[string]any),
	}
}

// WithExecutionData sets the execution data on the hook data.
func (d *HookData) WithExecutionData(data map[string]any) *HookData {
	d.ExecutionData = data
	return d
}

// WithResult sets the execution result.
func (d *HookData) WithResult(result string) *HookData {
	d.Result = result
	return d
}

// WithError sets the error.
func (d *HookData) WithError(err error) *HookData {
	d.Error = err
	return d
}

// WithValidationResult sets the validation result.
func (d *HookData) WithValidationResult(result *ValidationResult) *HookData {
	d.ValidationResult = result
	return d
}

// SetMetadata sets a metadata value.
func (d *HookData) SetMetadata(key string, value any) {
	if d.Metadata == nil {
		d.Metadata = make(map[string]any)
	}
	d.Metadata[key] = value
}

// GetMetadata gets a metadata value.
func (d *HookData) GetMetadata(key string) (any, bool) {
	if d.Metadata == nil {
		return nil, false
	}
	v, ok := d.Metadata[key]
	return v, ok
}

// HookRegistry manages hook registration and execution.
// It is thread-safe for concurrent access.
type HookRegistry struct {
	mu    sync.RWMutex
	hooks map[HookPoint][]Hook
}

// NewHookRegistry creates a new hook registry.
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{
		hooks: make(map[HookPoint][]Hook),
	}
}

// Register adds a hook for the specified point.
func (r *HookRegistry) Register(point HookPoint, hook Hook) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks[point] = append(r.hooks[point], hook)
}

// RegisterMultiple adds a hook for multiple points.
func (r *HookRegistry) RegisterMultiple(hook Hook, points ...HookPoint) {
	for _, point := range points {
		r.Register(point, hook)
	}
}

// Clear removes all hooks for a specific point.
func (r *HookRegistry) Clear(point HookPoint) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.hooks, point)
}

// ClearAll removes all hooks.
func (r *HookRegistry) ClearAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.hooks = make(map[HookPoint][]Hook)
}

// Run executes all hooks for the specified point.
// For "before" hooks, the first error stops execution and returns the error.
// For "after" hooks, all hooks are executed and errors are collected.
func (r *HookRegistry) Run(ctx context.Context, point HookPoint, data *HookData) error {
	r.mu.RLock()
	hooks := r.hooks[point]
	r.mu.RUnlock()

	if len(hooks) == 0 {
		return nil
	}

	isBefore := isBeforeHook(point)

	for _, hook := range hooks {
		err := hook(ctx, point, data)
		if err != nil {
			if isBefore {
				// Before hooks abort on first error
				return err
			}
			// After hooks continue on error (error is logged by caller)
		}
	}

	return nil
}

// RunWithErrors executes all hooks and returns all errors.
// Useful for after hooks where you want to know all errors.
func (r *HookRegistry) RunWithErrors(ctx context.Context, point HookPoint, data *HookData) []error {
	r.mu.RLock()
	hooks := r.hooks[point]
	r.mu.RUnlock()

	if len(hooks) == 0 {
		return nil
	}

	var errors []error
	for _, hook := range hooks {
		if err := hook(ctx, point, data); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// Count returns the number of hooks registered for a point.
func (r *HookRegistry) Count(point HookPoint) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.hooks[point])
}

// HasHooks checks if any hooks are registered for a point.
func (r *HookRegistry) HasHooks(point HookPoint) bool {
	return r.Count(point) > 0
}

// isBeforeHook checks if a hook point is a "before" hook.
func isBeforeHook(point HookPoint) bool {
	switch point {
	case HookBeforeLoad, HookBeforeExecute, HookBeforeSave, HookBeforeDelete, HookBeforeValidate:
		return true
	default:
		return false
	}
}

// LoggingHook creates a hook that logs operations.
// Useful for debugging and audit trails.
func LoggingHook(logFn func(point HookPoint, data *HookData)) Hook {
	return func(ctx context.Context, point HookPoint, data *HookData) error {
		logFn(point, data)
		return nil
	}
}

// Metadata key for timing hook start timestamp.
const metadataKeyTimingStartNs = "_timing_start_ns"

// TimingHook creates a hook that tracks operation timing.
// Call the returned function to get elapsed time in nanoseconds in "after" hooks.
func TimingHook() (Hook, func(*HookData) int64) {
	hook := func(ctx context.Context, point HookPoint, data *HookData) error {
		if isBeforeHook(point) {
			data.SetMetadata(metadataKeyTimingStartNs, nanotime())
		}
		return nil
	}

	getElapsed := func(data *HookData) int64 {
		start, ok := data.GetMetadata(metadataKeyTimingStartNs)
		if !ok {
			return 0
		}
		startNs, ok := start.(int64)
		if !ok {
			return 0
		}
		return nanotime() - startNs
	}

	return hook, getElapsed
}

// nanotime returns current time in nanoseconds.
func nanotime() int64 {
	return time.Now().UnixNano()
}

// Hook error messages.
const (
	ErrMsgHookFailed = "hook execution failed"
)

// Hook error format string constants.
const (
	hookErrFmtPoint     = " (hook: "
	hookErrFmtPointEnd  = ")"
	hookErrFmtCauseSep  = ": "
)

// HookError represents an error from hook execution.
type HookError struct {
	Message string
	Point   HookPoint
	Cause   error
}

// Error implements the error interface.
func (e *HookError) Error() string {
	msg := e.Message
	if e.Point != "" {
		msg += hookErrFmtPoint + string(e.Point) + hookErrFmtPointEnd
	}
	if e.Cause != nil {
		msg += hookErrFmtCauseSep + e.Cause.Error()
	}
	return msg
}

// Unwrap returns the underlying error.
func (e *HookError) Unwrap() error {
	return e.Cause
}

// NewHookError creates a new hook error.
func NewHookError(point HookPoint, cause error) *HookError {
	return &HookError{
		Message: ErrMsgHookFailed,
		Point:   point,
		Cause:   cause,
	}
}
