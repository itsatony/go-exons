package internal

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockNowContext is a minimal ContextAccessor that returns a single seeded key,
// used to pin the reference time for deterministic {~exons.now~} assertions.
type mockNowContext struct {
	values map[string]any
}

func (m *mockNowContext) Get(path string) (any, bool) { v, ok := m.values[path]; return v, ok }
func (m *mockNowContext) GetString(path string) string {
	if v, ok := m.values[path].(string); ok {
		return v
	}
	return ""
}
func (m *mockNowContext) GetStringDefault(path, def string) string {
	if v, ok := m.values[path].(string); ok {
		return v
	}
	return def
}
func (m *mockNowContext) Has(path string) bool { _, ok := m.values[path]; return ok }

// refInstant is the pinned reference time: 2026-07-21T14:30:00Z (a Tuesday).
var refInstant = time.Date(2026, time.July, 21, 14, 30, 0, 0, time.UTC)

func seededCtx() *mockNowContext {
	return &mockNowContext{values: map[string]any{ContextKeyReferenceTime: refInstant}}
}

func TestNowResolver_TagName(t *testing.T) {
	assert.Equal(t, TagNameNow, NewNowResolver().TagName())
}

func TestNowResolver_ValidateAlwaysOK(t *testing.T) {
	r := NewNowResolver()
	assert.NoError(t, r.Validate(nil))
	assert.NoError(t, r.Validate(Attributes{AttrFormat: "date"}))
	assert.NoError(t, r.Validate(Attributes{AttrFormat: "bogus"}))
}

func TestNowResolver_NamedFormats(t *testing.T) {
	r := NewNowResolver()
	ctx := context.Background()
	execCtx := seededCtx()

	cases := []struct {
		format string
		want   string
	}{
		{"", "2026-07-21T14:30:00Z"},
		{NowFormatISO, "2026-07-21T14:30:00Z"},
		{NowFormatDate, "2026-07-21"},
		{NowFormatDateTime, "2026-07-21 14:30:00"},
		{NowFormatTime, "14:30:00"},
		{NowFormatYear, "2026"},
		{NowFormatMonth, "07"},
		{NowFormatDay, "21"},
		{NowFormatWeekday, "Tuesday"},
		{NowFormatUnix, "1784644200"},
		{NowFormatRFC1123, "Tue, 21 Jul 2026 14:30:00 UTC"},
		{NowFormatDateDE, "21.07.2026"},
	}
	for _, c := range cases {
		attrs := Attributes{}
		if c.format != "" {
			attrs[AttrFormat] = c.format
		}
		out, err := r.Resolve(ctx, execCtx, attrs)
		require.NoError(t, err, "format=%q", c.format)
		assert.Equal(t, c.want, out, "format=%q", c.format)
	}
}

func TestNowResolver_Timezone(t *testing.T) {
	r := NewNowResolver()
	// 14:30 UTC is 16:30 in Berlin (CEST, +02:00) on 2026-07-21.
	out, err := r.Resolve(context.Background(), seededCtx(),
		Attributes{AttrFormat: NowFormatTime, AttrTz: "Europe/Berlin"})
	require.NoError(t, err)
	assert.Equal(t, "16:30:00", out)
}

func TestNowResolver_InvalidTimezone(t *testing.T) {
	r := NewNowResolver()
	_, err := r.Resolve(context.Background(), seededCtx(),
		Attributes{AttrTz: "Mars/Olympus_Mons"})
	require.Error(t, err)
}

func TestNowResolver_RawLayoutOverridesFormat(t *testing.T) {
	r := NewNowResolver()
	// layout= wins over format=.
	out, err := r.Resolve(context.Background(), seededCtx(),
		Attributes{AttrFormat: NowFormatDate, AttrLayout: "Mon Jan 2 2006"})
	require.NoError(t, err)
	assert.Equal(t, "Tue Jul 21 2026", out)
}

func TestNowResolver_UnknownFormat(t *testing.T) {
	r := NewNowResolver()
	_, err := r.Resolve(context.Background(), seededCtx(), Attributes{AttrFormat: "eon"})
	require.Error(t, err)
}

func TestNowResolver_FallsBackToWallClockWhenUnseeded(t *testing.T) {
	r := NewNowResolver()
	// No seeded reference time and no ContextAccessor at all → time.Now() path.
	out, err := r.Resolve(context.Background(), nil, Attributes{AttrFormat: NowFormatYear})
	require.NoError(t, err)
	assert.Equal(t, time.Now().UTC().Format(LayoutYear), out)
}
