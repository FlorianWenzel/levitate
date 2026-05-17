package api

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// pgUUID converts a textual UUID to pgtype.UUID. Returns an invalid value on parse error.
func pgUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	parsed, err := uuid.Parse(s)
	if err != nil {
		return u, err
	}
	u.Bytes = parsed
	u.Valid = true
	return u, nil
}

func uuidString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

// uuidStringPtr returns nil for an unset UUID so JSON serialization emits
// `null` rather than `""`, which matches Float's contract for nullable
// reference fields (e.g. `created_by`, `modified_by`).
func uuidStringPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := uuid.UUID(u.Bytes).String()
	return &s
}

func tsPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	tt := t.Time
	return &tt
}

func ts(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func numericFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return f.Float64
}

func numericFromFloat(f float64) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.Scan(strconv.FormatFloat(f, 'f', -1, 64)); err != nil {
		return n, err
	}
	return n, nil
}
