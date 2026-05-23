package admin

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type EmailTemplateHandler struct {
	templateService *service.EmailTemplateService
	emailService    *service.EmailService
	settingService  *service.SettingService
}

func NewEmailTemplateHandler(templateService *service.EmailTemplateService, emailService *service.EmailService, settingService *service.SettingService) *EmailTemplateHandler {
	return &EmailTemplateHandler{templateService: templateService, emailService: emailService, settingService: settingService}
}

func (h *EmailTemplateHandler) List(c *gin.Context) {
	if h.templateService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Email template service not available")
		return
	}
	items, err := h.templateService.ListTemplates(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

type updateEmailTemplateRequest struct {
	Subject string `json:"subject" binding:"required"`
	Body    string `json:"body" binding:"required"`
	Enabled *bool  `json:"enabled"`
}

func (h *EmailTemplateHandler) Update(c *gin.Context) {
	if h.templateService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Email template service not available")
		return
	}
	var req updateEmailTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	tmpl, err := h.templateService.UpsertTemplate(c.Request.Context(), c.Param("key"), c.Param("locale"), req.Subject, req.Body, enabled)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tmpl)
}

func (h *EmailTemplateHandler) Reset(c *gin.Context) {
	if h.templateService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Email template service not available")
		return
	}
	tmpl, err := h.templateService.ResetTemplate(c.Request.Context(), c.Param("key"), c.Param("locale"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, tmpl)
}

type testEmailTemplateRequest struct {
	Email string            `json:"email" binding:"required,email"`
	Data  map[string]string `json:"data"`
}

func (h *EmailTemplateHandler) Test(c *gin.Context) {
	if h.templateService == nil || h.emailService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Email template service not available")
		return
	}
	var req testEmailTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	data := defaultEmailTemplateTestData(h.siteName(c), req.Data)
	subject, body, err := h.templateService.Render(c.Request.Context(), c.Param("key"), c.Param("locale"), data)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if err := h.emailService.SendEmail(c.Request.Context(), strings.TrimSpace(req.Email), subject, body); err != nil {
		response.BadRequest(c, "Failed to send test email: "+err.Error())
		return
	}
	response.Success(c, gin.H{"message": "Test email sent successfully"})
}

func (h *EmailTemplateHandler) siteName(c *gin.Context) string {
	if h.settingService == nil {
		return "Sub2API"
	}
	return h.settingService.GetSiteName(c.Request.Context())
}

func defaultEmailTemplateTestData(siteName string, extra map[string]string) map[string]string {
	data := map[string]string{
		"SiteName":            siteName,
		"Code":                "123456",
		"ResetURL":            "https://example.com/reset-password?token=test",
		"OrderNo":             "pay_test",
		"ProductType":         "balance_topup",
		"Amount":              "10.00",
		"Currency":            "USD",
		"Balance":             "2.00",
		"Threshold":           "5.00",
		"GroupName":           "Default",
		"ExpiresAt":           "2026-06-01 09:00:00",
		"DaysLeft":            "3",
		"AccountName":         "Example account",
		"AccountID":           "1",
		"Platform":            "openai",
		"PlanID":              "1",
		"ResultID":            "1",
		"Model":               "gpt-5",
		"Status":              "success",
		"LatencyMs":           "1200",
		"Error":               "-",
		"ConsecutiveFailures": "0",
		"CompletedAt":         "2026-05-23 09:00:00",
		"NextRun":             "2026-05-23 10:00:00",
	}
	for k, v := range extra {
		if strings.TrimSpace(k) != "" {
			data[k] = v
		}
	}
	return data
}
