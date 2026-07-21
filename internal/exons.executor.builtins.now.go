package internal

import (
	"context"
	"strconv"
	"time"
)

// NowResolver handles the exons.now built-in OUTPUT tag: it prints a formatted
// reference time directly into body text.
//
// This is deliberately distinct from the date/time EXPRESSION functions (now,
// formatDate, year, …) registered for eval= expressions on control-flow tags:
// those have no output path, so before this tag an author could not simply drop
// "today's date" into a prompt. {~exons.now format="date" /~} fills that gap.
//
// The reference time is seeded once per render by the caller under the reserved
// ContextKeyReferenceTime data key, so every {~exons.now~} in one render agrees
// (deterministic within a render) and tests can pin an exact instant. Absent a
// seed it falls back to time.Now(). Like EnvResolver this is attribute-only — no
// spec resolver, no I/O — so it mirrors that shape rather than RefResolver's.
//
// Usage:
//
//	{~exons.now /~}                                  -> 2026-07-21T14:30:00Z (RFC-3339 UTC)
//	{~exons.now format="date" /~}                    -> 2026-07-21
//	{~exons.now format="date-de" /~}                 -> 21.07.2026
//	{~exons.now format="datetime" tz="Europe/Berlin" /~}
//	{~exons.now layout="Mon Jan 2" /~}               -> raw Go layout (power users)
type NowResolver struct{}

// NewNowResolver creates a new NowResolver.
func NewNowResolver() *NowResolver { return &NowResolver{} }

// TagName returns the tag name for this resolver.
func (r *NowResolver) TagName() string { return TagNameNow }

// Validate accepts the tag unconditionally: every attribute is optional. An
// unknown format or timezone is reported from Resolve (not here), so the engine's
// configured error strategy governs it — a parse-time Validate error would bypass
// that and hard-fail a template a lenient strategy would otherwise render.
func (r *NowResolver) Validate(attrs Attributes) error { return nil }

// Resolve renders the seeded reference time under the requested format/timezone.
func (r *NowResolver) Resolve(_ context.Context, execCtx interface{}, attrs Attributes) (string, error) {
	ref := referenceTime(execCtx)

	if tz, ok := attrs.Get(AttrTz); ok && tz != "" {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return "", NewBuiltinError(ErrMsgNowInvalidTimezone, TagNameNow)
		}
		ref = ref.In(loc)
	} else {
		// Default timezone is UTC (the product spans locales; a naked wall clock
		// would be ambiguous). A named format then formats this UTC instant.
		ref = ref.UTC()
	}

	// An explicit raw Go layout wins over a named format — the power-user escape
	// hatch for a format the curated set does not cover.
	if layout, ok := attrs.Get(AttrLayout); ok && layout != "" {
		return ref.Format(layout), nil
	}

	format, _ := attrs.Get(AttrFormat)
	out, ok := formatReferenceTime(ref, format)
	if !ok {
		return "", NewBuiltinError(ErrMsgNowUnknownFormat, TagNameNow)
	}
	return out, nil
}

// referenceTime returns the caller-seeded reference time (the reserved data key),
// or time.Now() when unseeded. Seeding is what makes one render's timestamps agree
// and lets tests assert an exact instant.
func referenceTime(execCtx interface{}) time.Time {
	if accessor, ok := execCtx.(ContextAccessor); ok {
		if v, found := accessor.Get(ContextKeyReferenceTime); found {
			if t, isTime := v.(time.Time); isTime {
				return t
			}
		}
	}
	return time.Now()
}

// formatReferenceTime renders ref under a named format. The empty format and
// NowFormatISO both yield RFC-3339. Returns ok=false for an unrecognized name so
// the caller can raise a builtin error routed through the engine's error strategy.
func formatReferenceTime(ref time.Time, format string) (string, bool) {
	switch format {
	case "", NowFormatISO:
		return ref.Format(time.RFC3339), true
	case NowFormatDate:
		return ref.Format(DateFormatISO), true
	case NowFormatDateTime:
		return ref.Format(DateTimeFormatSpaced), true
	case NowFormatTime:
		return ref.Format(TimeFormat24H), true
	case NowFormatYear:
		return ref.Format(LayoutYear), true
	case NowFormatMonth:
		return ref.Format(LayoutMonthNumeric), true
	case NowFormatDay:
		return ref.Format(LayoutDayNumeric), true
	case NowFormatWeekday:
		return ref.Weekday().String(), true
	case NowFormatUnix:
		return strconv.FormatInt(ref.Unix(), 10), true
	case NowFormatRFC1123:
		return ref.Format(time.RFC1123), true
	case NowFormatDateDE:
		return ref.Format(DateFormatDE), true
	default:
		return "", false
	}
}
