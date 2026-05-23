package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type emailTemplateRepository struct {
	db *sql.DB
}

func NewEmailTemplateRepository(db *sql.DB) service.EmailTemplateRepository {
	return &emailTemplateRepository{db: db}
}

func (r *emailTemplateRepository) ListEmailTemplates(ctx context.Context) ([]service.EmailTemplate, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil email template repository")
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT id, template_key, locale, subject, body, enabled, is_custom, created_at, updated_at
FROM email_templates
ORDER BY template_key ASC, locale ASC`)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var out []service.EmailTemplate
	for rows.Next() {
		var item service.EmailTemplate
		if err := rows.Scan(&item.ID, &item.TemplateKey, &item.Locale, &item.Subject, &item.Body, &item.Enabled, &item.IsCustom, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *emailTemplateRepository) GetEmailTemplate(ctx context.Context, key, locale string) (*service.EmailTemplate, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil email template repository")
	}
	var item service.EmailTemplate
	err := r.db.QueryRowContext(ctx, `
SELECT id, template_key, locale, subject, body, enabled, is_custom, created_at, updated_at
FROM email_templates
WHERE template_key = $1 AND locale = $2
LIMIT 1`, strings.TrimSpace(key), service.NormalizeEmailLocale(locale)).
		Scan(&item.ID, &item.TemplateKey, &item.Locale, &item.Subject, &item.Body, &item.Enabled, &item.IsCustom, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrEmailTemplateNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *emailTemplateRepository) UpsertEmailTemplate(ctx context.Context, tmpl *service.EmailTemplate) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil email template repository")
	}
	if tmpl == nil {
		return fmt.Errorf("nil email template")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO email_templates (template_key, locale, subject, body, enabled, is_custom, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, TRUE, NOW(), NOW())
ON CONFLICT (template_key, locale)
DO UPDATE SET subject = EXCLUDED.subject, body = EXCLUDED.body, enabled = EXCLUDED.enabled, is_custom = TRUE, updated_at = NOW()`,
		strings.TrimSpace(tmpl.TemplateKey),
		service.NormalizeEmailLocale(tmpl.Locale),
		tmpl.Subject,
		tmpl.Body,
		tmpl.Enabled,
	)
	return err
}

func (r *emailTemplateRepository) DeleteEmailTemplate(ctx context.Context, key, locale string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil email template repository")
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM email_templates WHERE template_key = $1 AND locale = $2`, strings.TrimSpace(key), service.NormalizeEmailLocale(locale))
	return err
}

func (r *emailTemplateRepository) GetUserNotificationPreference(ctx context.Context, userID int64, category string) (bool, error) {
	if r == nil || r.db == nil {
		return true, fmt.Errorf("nil email template repository")
	}
	var enabled bool
	err := r.db.QueryRowContext(ctx, `
SELECT enabled FROM user_notification_preferences
WHERE user_id = $1 AND category = $2
LIMIT 1`, userID, strings.TrimSpace(category)).Scan(&enabled)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	return enabled, err
}

func (r *emailTemplateRepository) UpsertUserNotificationPreference(ctx context.Context, userID int64, category string, enabled bool) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil email template repository")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO user_notification_preferences (user_id, category, enabled, created_at, updated_at)
VALUES ($1, $2, $3, NOW(), NOW())
ON CONFLICT (user_id, category)
DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = NOW()`,
		userID, strings.TrimSpace(category), enabled)
	return err
}

func (r *emailTemplateRepository) MarkNotificationSentOnce(ctx context.Context, dedupeKey string, ttl time.Duration) (bool, error) {
	if r == nil || r.db == nil {
		return true, fmt.Errorf("nil email template repository")
	}
	if ttl <= 0 {
		ttl = 48 * time.Hour
	}
	expiresAt := time.Now().UTC().Add(ttl)
	var inserted int
	err := r.db.QueryRowContext(ctx, `
INSERT INTO notification_dedupe_keys (dedupe_key, expires_at, created_at)
VALUES ($1, $2, NOW())
ON CONFLICT (dedupe_key) DO NOTHING
RETURNING 1`, strings.TrimSpace(dedupeKey), expiresAt).Scan(&inserted)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}
