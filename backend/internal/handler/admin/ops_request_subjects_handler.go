package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *OpsHandler) BindUsageHandler(usageHandler *UsageHandler) {
	if h == nil {
		return
	}
	h.usageService = nil
	if usageHandler != nil {
		h.usageService = usageHandler.usageService
	}
}

// GetSubjectInsights returns usage insights for account/group/api_key subjects.
// GET /api/v1/admin/ops/request-details/subjects/insights
func (h *OpsHandler) GetSubjectInsights(c *gin.Context) {
	if h == nil || h.usageService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Usage service not available")
		return
	}

	subjectType, subjectID, err := parseUsageSubjectQuery(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	startTime, endTime, err := parseOpsTimeRange(c, "30d")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	insights, err := h.usageService.GetSubjectUsageInsights(c.Request.Context(), service.UsageSubjectInsightsQuery{
		SubjectType: subjectType,
		SubjectID:   subjectID,
		StartTime:   startTime,
		EndTime:     endTime,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, insights)
}

func parseUsageSubjectQuery(c *gin.Context) (service.UsageSubjectType, int64, error) {
	subjectType := service.UsageSubjectType(strings.TrimSpace(c.Query("subject_type")))
	subjectIDRaw := strings.TrimSpace(c.Query("subject_id"))

	switch {
	case subjectType != "" && subjectIDRaw != "":
	case strings.TrimSpace(c.Query("account_id")) != "":
		subjectType = service.UsageSubjectTypeAccount
		subjectIDRaw = strings.TrimSpace(c.Query("account_id"))
	case strings.TrimSpace(c.Query("group_id")) != "":
		subjectType = service.UsageSubjectTypeGroup
		subjectIDRaw = strings.TrimSpace(c.Query("group_id"))
	case strings.TrimSpace(c.Query("api_key_id")) != "":
		subjectType = service.UsageSubjectTypeAPIKey
		subjectIDRaw = strings.TrimSpace(c.Query("api_key_id"))
	default:
		return "", 0, usageSubjectQueryError("subject_type and subject_id are required")
	}

	subjectID, err := strconv.ParseInt(subjectIDRaw, 10, 64)
	if err != nil || subjectID <= 0 {
		return "", 0, usageSubjectQueryError("Invalid subject_id")
	}

	switch subjectType {
	case service.UsageSubjectTypeAccount, service.UsageSubjectTypeGroup, service.UsageSubjectTypeAPIKey:
		return subjectType, subjectID, nil
	default:
		return "", 0, usageSubjectQueryError("subject_type must be account, group, or api_key")
	}
}

type usageSubjectQueryError string

func (e usageSubjectQueryError) Error() string {
	return string(e)
}
