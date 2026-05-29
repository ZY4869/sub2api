package repository

import (
	"database/sql"
	"strings"
	"time"
)

func nullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

func nullBool(v *bool) sql.NullBool {
	if v == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *v, Valid: true}
}

func opsNullString(v any) any {
	switch s := v.(type) {
	case nil:
		return sql.NullString{}
	case *string:
		if s == nil || strings.TrimSpace(*s) == "" {
			return sql.NullString{}
		}
		return sql.NullString{String: strings.TrimSpace(*s), Valid: true}
	case string:
		if strings.TrimSpace(s) == "" {
			return sql.NullString{}
		}
		return sql.NullString{String: strings.TrimSpace(s), Valid: true}
	default:
		return sql.NullString{}
	}
}

func opsStringOrEmpty(v any) string {
	switch s := v.(type) {
	case nil:
		return ""
	case *string:
		if s == nil {
			return ""
		}
		return strings.TrimSpace(*s)
	case string:
		return strings.TrimSpace(s)
	default:
		return ""
	}
}

func opsNullInt64(v *int64) any {
	if v == nil || *v == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func opsNullInt(v any) any {
	switch n := v.(type) {
	case nil:
		return sql.NullInt64{}
	case *int:
		if n == nil || *n == 0 {
			return sql.NullInt64{}
		}
		return sql.NullInt64{Int64: int64(*n), Valid: true}
	case *int64:
		if n == nil || *n == 0 {
			return sql.NullInt64{}
		}
		return sql.NullInt64{Int64: *n, Valid: true}
	case int:
		if n == 0 {
			return sql.NullInt64{}
		}
		return sql.NullInt64{Int64: int64(n), Valid: true}
	default:
		return sql.NullInt64{}
	}
}

func opsNullInt16(v *int16) any {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*v), Valid: true}
}
