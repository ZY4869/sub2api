package repository

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

const opsSystemLogInsertSQL = `
INSERT INTO ops_system_logs (
  created_at,
  level,
  component,
  message,
  request_id,
  client_request_id,
  user_id,
  api_key_id,
  account_id,
  platform,
  model,
  extra
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
`

func (r *opsRepository) BatchInsertSystemLogs(ctx context.Context, inputs []*service.OpsInsertSystemLogInput) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil ops repository")
	}
	if len(inputs) == 0 {
		return 0, nil
	}

	inserted, err := r.batchInsertSystemLogsCopy(ctx, inputs)
	if err == nil {
		return inserted, nil
	}
	if errors.Is(err, driver.ErrSkip) {
		return r.batchInsertSystemLogsFallback(ctx, inputs)
	}
	return inserted, err
}

func (r *opsRepository) batchInsertSystemLogsCopy(ctx context.Context, inputs []*service.OpsInsertSystemLogInput) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn(
		"ops_system_logs",
		"created_at",
		"level",
		"component",
		"message",
		"request_id",
		"client_request_id",
		"user_id",
		"api_key_id",
		"account_id",
		"platform",
		"model",
		"extra",
	))
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}

	var inserted int64
	for _, input := range inputs {
		args, ok := buildSystemLogInsertArgs(input)
		if !ok {
			continue
		}
		if _, err := stmt.ExecContext(ctx, args...); err != nil {
			_ = stmt.Close()
			_ = tx.Rollback()
			return inserted, err
		}
		inserted++
	}

	if _, err := stmt.ExecContext(ctx); err != nil {
		_ = stmt.Close()
		_ = tx.Rollback()
		return inserted, err
	}
	if err := stmt.Close(); err != nil {
		_ = tx.Rollback()
		return inserted, err
	}
	if err := tx.Commit(); err != nil {
		return inserted, err
	}
	return inserted, nil
}

func (r *opsRepository) batchInsertSystemLogsFallback(ctx context.Context, inputs []*service.OpsInsertSystemLogInput) (int64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}

	var inserted int64
	for _, input := range inputs {
		args, ok := buildSystemLogInsertArgs(input)
		if !ok {
			continue
		}
		if _, err := tx.ExecContext(ctx, opsSystemLogInsertSQL, args...); err != nil {
			_ = tx.Rollback()
			return inserted, err
		}
		inserted++
	}

	if err := tx.Commit(); err != nil {
		return inserted, err
	}
	return inserted, nil
}

func buildSystemLogInsertArgs(input *service.OpsInsertSystemLogInput) ([]any, bool) {
	if input == nil {
		return nil, false
	}
	createdAt := input.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	level := strings.ToLower(strings.TrimSpace(input.Level))
	message := strings.TrimSpace(input.Message)
	if level == "" || message == "" {
		return nil, false
	}
	component := strings.TrimSpace(input.Component)
	if component == "" {
		component = "app"
	}
	extra := strings.TrimSpace(input.ExtraJSON)
	if extra == "" {
		extra = "{}"
	}
	return []any{
		createdAt.UTC(),
		level,
		component,
		message,
		opsNullString(input.RequestID),
		opsNullString(input.ClientRequestID),
		opsNullInt64(input.UserID),
		opsNullInt64(input.APIKeyID),
		opsNullInt64(input.AccountID),
		opsNullString(input.Platform),
		opsNullString(input.Model),
		extra,
	}, true
}
