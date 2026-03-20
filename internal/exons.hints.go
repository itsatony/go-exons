package internal

// Hint constants provide actionable guidance appended to error messages.
const (
	HintVarNotFound      = "Hint: Use default=\"value\" to provide a fallback, or onerror=\"default:value\" for error handling"
	HintTemplateNotFound = "Hint: Register the template with engine.RegisterTemplate() or engine.MustRegisterTemplate()"
	HintRefNoResolver    = "Hint: Pass a DocumentResolver via CompileOptions.Resolver to resolve spec references"
	HintRefNotFound      = "Hint: Ensure the slug exists in your DocumentResolver and check for typos"
	HintSeparator        = "\n"
)

// ShouldShowHint returns true if the attributes do not contain a default or onerror
// attribute, meaning the user has not already configured a workaround for the error.
func ShouldShowHint(attrs Attributes) bool {
	if attrs == nil {
		return true
	}
	if attrs.Has(AttrDefault) {
		return false
	}
	if attrs.Has(AttrOnError) {
		return false
	}
	return true
}

// AppendHint appends a hint to a message with a newline separator.
// Returns the original message if hint is empty.
func AppendHint(msg, hint string) string {
	if hint == "" {
		return msg
	}
	return msg + HintSeparator + hint
}
