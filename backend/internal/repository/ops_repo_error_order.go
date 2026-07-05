package repository

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func opsErrorLogsOrderBy(filter *service.OpsErrorLogFilter) string {
	sortBy := ""
	sortOrder := "desc"
	if filter != nil {
		sortBy = strings.ToLower(strings.TrimSpace(filter.SortBy))
		if strings.EqualFold(strings.TrimSpace(filter.SortOrder), "asc") {
			sortOrder = "asc"
		}
	}

	column := "e.created_at"
	switch sortBy {
	case "model":
		column = "COALESCE(NULLIF(TRIM(e.requested_model), ''), e.model, '')"
	case "status_code":
		column = "COALESCE(e.upstream_status_code, e.status_code, 0)"
	}

	return column + " " + sortOrder + ", e.id " + sortOrder
}
