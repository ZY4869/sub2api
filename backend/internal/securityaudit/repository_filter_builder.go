package securityaudit

import (
	"fmt"
	"time"
)

type eventWhereBuilder struct {
	clauses []string
	args    []any
}

func (b *eventWhereBuilder) add(column string, value any) {
	b.args = append(b.args, value)
	b.clauses = append(b.clauses, fmt.Sprintf("%s=$%d", column, len(b.args)))
}

func (b *eventWhereBuilder) addIf(column string, value string) {
	if value != "" {
		b.add(column, value)
	}
}

func (b *eventWhereBuilder) addPtr(column string, value *int64) {
	if value != nil {
		b.add(column, *value)
	}
}

func (b *eventWhereBuilder) addKeyword(keyword string) {
	if keyword == "" {
		return
	}
	b.args = append(b.args, "%"+keyword+"%")
	b.clauses = append(b.clauses, fmt.Sprintf("redacted_preview ILIKE $%d", len(b.args)))
}

func (b *eventWhereBuilder) addTime(column string, value *time.Time) {
	if value != nil {
		b.add(column, value.UTC())
	}
}
