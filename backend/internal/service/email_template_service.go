package service

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"sort"
	"strings"
	"text/template"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	EmailTemplateVerifyCode             = "verify_code"
	EmailTemplatePasswordReset          = "password_reset"
	EmailTemplatePaymentSuccess         = "payment_success"
	EmailTemplateBalanceLow             = "balance_low"
	EmailTemplateSubscriptionExpiring   = "subscription_expiring"
	EmailTemplateScheduledTestResult    = "scheduled_test_result"
	NotificationCategoryPaymentSuccess  = "payment_success"
	NotificationCategoryBalanceLow      = "balance_low"
	NotificationCategorySubscriptionExp = "subscription_expiring"
)

var (
	ErrEmailTemplateNotFound       = infraerrors.NotFound("EMAIL_TEMPLATE_NOT_FOUND", "email template not found")
	ErrEmailTemplateInvalid        = infraerrors.BadRequest("EMAIL_TEMPLATE_INVALID", "invalid email template")
	ErrNotificationCategoryInvalid = infraerrors.BadRequest("NOTIFICATION_CATEGORY_INVALID", "invalid notification category")
)

type EmailTemplate struct {
	ID          int64     `json:"id"`
	TemplateKey string    `json:"key"`
	Locale      string    `json:"locale"`
	Subject     string    `json:"subject"`
	Body        string    `json:"body"`
	Enabled     bool      `json:"enabled"`
	IsCustom    bool      `json:"is_custom"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type EmailTemplateDefinition struct {
	Key       string                   `json:"key"`
	Name      string                   `json:"name"`
	Variables []string                 `json:"variables"`
	BuiltIn   map[string]EmailTemplate `json:"built_in"`
}

type NotificationPreference struct {
	Category string `json:"category"`
	Enabled  bool   `json:"enabled"`
}

type EmailTemplateRepository interface {
	ListEmailTemplates(ctx context.Context) ([]EmailTemplate, error)
	GetEmailTemplate(ctx context.Context, key, locale string) (*EmailTemplate, error)
	UpsertEmailTemplate(ctx context.Context, tmpl *EmailTemplate) error
	DeleteEmailTemplate(ctx context.Context, key, locale string) error
	GetUserNotificationPreference(ctx context.Context, userID int64, category string) (bool, error)
	UpsertUserNotificationPreference(ctx context.Context, userID int64, category string, enabled bool) error
	MarkNotificationSentOnce(ctx context.Context, dedupeKey string, ttl time.Duration) (bool, error)
}

type EmailTemplateService struct {
	repo EmailTemplateRepository
}

func NewEmailTemplateService(repo EmailTemplateRepository) *EmailTemplateService {
	return &EmailTemplateService{repo: repo}
}

func (s *EmailTemplateService) ListTemplates(ctx context.Context) ([]EmailTemplateDefinition, error) {
	defs := EmailTemplateDefinitions()
	stored := map[string]EmailTemplate{}
	if s != nil && s.repo != nil {
		items, err := s.repo.ListEmailTemplates(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			stored[emailTemplateMapKey(item.TemplateKey, item.Locale)] = item
		}
	}
	for i := range defs {
		for locale := range defs[i].BuiltIn {
			if item, ok := stored[emailTemplateMapKey(defs[i].Key, locale)]; ok {
				defs[i].BuiltIn[locale] = item
			}
		}
	}
	return defs, nil
}

func (s *EmailTemplateService) UpsertTemplate(ctx context.Context, key, locale, subject, body string, enabled bool) (*EmailTemplate, error) {
	def, ok := EmailTemplateDefinitionByKey(key)
	if !ok {
		return nil, ErrEmailTemplateInvalid
	}
	locale = NormalizeEmailLocale(locale)
	subject = strings.TrimSpace(subject)
	body = strings.TrimSpace(body)
	if subject == "" || body == "" {
		return nil, ErrEmailTemplateInvalid
	}
	tmpl := &EmailTemplate{TemplateKey: def.Key, Locale: locale, Subject: subject, Body: body, Enabled: enabled, IsCustom: true}
	if s == nil || s.repo == nil {
		return tmpl, nil
	}
	if err := s.repo.UpsertEmailTemplate(ctx, tmpl); err != nil {
		return nil, err
	}
	return s.GetTemplate(ctx, def.Key, locale)
}

func (s *EmailTemplateService) ResetTemplate(ctx context.Context, key, locale string) (*EmailTemplate, error) {
	def, ok := EmailTemplateDefinitionByKey(key)
	if !ok {
		return nil, ErrEmailTemplateInvalid
	}
	locale = NormalizeEmailLocale(locale)
	if s != nil && s.repo != nil {
		if err := s.repo.DeleteEmailTemplate(ctx, def.Key, locale); err != nil {
			return nil, err
		}
	}
	return s.GetTemplate(ctx, def.Key, locale)
}

func (s *EmailTemplateService) GetTemplate(ctx context.Context, key, locale string) (*EmailTemplate, error) {
	def, ok := EmailTemplateDefinitionByKey(key)
	if !ok {
		return nil, ErrEmailTemplateInvalid
	}
	locale = NormalizeEmailLocale(locale)
	if s != nil && s.repo != nil {
		if stored, err := s.repo.GetEmailTemplate(ctx, def.Key, locale); err == nil && stored != nil {
			return stored, nil
		}
	}
	if tmpl, ok := def.BuiltIn[locale]; ok {
		return cloneEmailTemplate(&tmpl), nil
	}
	if tmpl, ok := def.BuiltIn["en"]; ok {
		tmpl.Locale = locale
		return cloneEmailTemplate(&tmpl), nil
	}
	return nil, ErrEmailTemplateNotFound
}

func (s *EmailTemplateService) Render(ctx context.Context, key, locale string, data map[string]string) (subject string, body string, err error) {
	tmpl, err := s.GetTemplate(ctx, key, locale)
	if err != nil {
		return "", "", err
	}
	subject, body, err = renderEmailTemplate(tmpl, data)
	if err == nil {
		return subject, body, nil
	}
	def, ok := EmailTemplateDefinitionByKey(key)
	if !ok {
		return "", "", err
	}
	fallback := def.BuiltIn[NormalizeEmailLocale(locale)]
	if fallback.Subject == "" {
		fallback = def.BuiltIn["en"]
	}
	subject, body, fallbackErr := renderEmailTemplate(&fallback, data)
	if fallbackErr != nil {
		return "", "", err
	}
	return subject, body, nil
}

func (s *EmailTemplateService) UserNotificationEnabled(ctx context.Context, userID int64, category string) bool {
	category = strings.TrimSpace(category)
	if s == nil || s.repo == nil || userID <= 0 || category == "" {
		return true
	}
	enabled, err := s.repo.GetUserNotificationPreference(ctx, userID, category)
	if err != nil {
		return true
	}
	return enabled
}

func (s *EmailTemplateService) ListNotificationPreferences(ctx context.Context, userID int64) ([]NotificationPreference, error) {
	if userID <= 0 {
		return nil, ErrNotificationCategoryInvalid
	}
	categories := NotificationCategories()
	out := make([]NotificationPreference, 0, len(categories))
	for _, category := range categories {
		enabled := true
		if s != nil && s.repo != nil {
			stored, err := s.repo.GetUserNotificationPreference(ctx, userID, category)
			if err != nil {
				return nil, err
			}
			enabled = stored
		}
		out = append(out, NotificationPreference{Category: category, Enabled: enabled})
	}
	return out, nil
}

func (s *EmailTemplateService) UpdateNotificationPreference(ctx context.Context, userID int64, category string, enabled bool) (*NotificationPreference, error) {
	category = strings.TrimSpace(category)
	if userID <= 0 || !IsValidNotificationCategory(category) {
		return nil, ErrNotificationCategoryInvalid
	}
	if s != nil && s.repo != nil {
		if err := s.repo.UpsertUserNotificationPreference(ctx, userID, category, enabled); err != nil {
			return nil, err
		}
	}
	return &NotificationPreference{Category: category, Enabled: enabled}, nil
}

func (s *EmailTemplateService) ShouldSendNotification(ctx context.Context, userID int64, category, resourceID, thresholdOrWindow string, now time.Time) bool {
	if !s.UserNotificationEnabled(ctx, userID, category) {
		return false
	}
	if s == nil || s.repo == nil {
		return true
	}
	day := now.UTC().Format("2006-01-02")
	key := strings.Join([]string{
		strings.TrimSpace(category),
		fmt.Sprintf("%d", userID),
		strings.TrimSpace(resourceID),
		strings.TrimSpace(thresholdOrWindow),
		day,
	}, ":")
	ok, err := s.repo.MarkNotificationSentOnce(ctx, key, 48*time.Hour)
	return err != nil || ok
}

func NotificationCategories() []string {
	return []string{
		NotificationCategoryPaymentSuccess,
		NotificationCategoryBalanceLow,
		NotificationCategorySubscriptionExp,
	}
}

func IsValidNotificationCategory(category string) bool {
	category = strings.TrimSpace(category)
	for _, item := range NotificationCategories() {
		if category == item {
			return true
		}
	}
	return false
}

func renderEmailTemplate(tmpl *EmailTemplate, data map[string]string) (string, string, error) {
	if tmpl == nil || !tmpl.Enabled {
		return "", "", ErrEmailTemplateNotFound
	}
	vars := map[string]string{}
	for k, v := range data {
		vars[strings.TrimSpace(k)] = v
	}
	subject, err := renderTemplateString(tmpl.Subject, vars)
	if err != nil {
		return "", "", err
	}
	body, err := renderTemplateString(tmpl.Body, vars)
	if err != nil {
		return "", "", err
	}
	return subject, body, nil
}

func renderTemplateString(src string, data map[string]string) (string, error) {
	t, err := template.New("email").Option("missingkey=zero").Funcs(template.FuncMap{
		"html": html.EscapeString,
	}).Parse(src)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func EmailTemplateDefinitionByKey(key string) (EmailTemplateDefinition, bool) {
	key = strings.TrimSpace(key)
	for _, def := range EmailTemplateDefinitions() {
		if def.Key == key {
			return def, true
		}
	}
	return EmailTemplateDefinition{}, false
}

func EmailTemplateDefinitions() []EmailTemplateDefinition {
	defs := []EmailTemplateDefinition{
		emailTemplateDefVerifyCode(),
		emailTemplateDefPasswordReset(),
		emailTemplateDefPaymentSuccess(),
		emailTemplateDefBalanceLow(),
		emailTemplateDefSubscriptionExpiring(),
		emailTemplateDefScheduledTestResult(),
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].Key < defs[j].Key })
	return defs
}

func emailTemplateMapKey(key, locale string) string {
	return strings.TrimSpace(key) + ":" + NormalizeEmailLocale(locale)
}

func cloneEmailTemplate(t *EmailTemplate) *EmailTemplate {
	if t == nil {
		return nil
	}
	cp := *t
	return &cp
}
