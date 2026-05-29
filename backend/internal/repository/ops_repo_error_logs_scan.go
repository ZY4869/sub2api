package repository

import (
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type opsErrorLogRow interface {
	Scan(dest ...any) error
}

func scanOpsErrorLogListRow(row opsErrorLogRow) (*service.OpsErrorLog, error) {
	var item service.OpsErrorLog
	var statusCode sql.NullInt64
	var clientIP sql.NullString
	var userID sql.NullInt64
	var apiKeyID sql.NullInt64
	var accountID sql.NullInt64
	var accountName string
	var groupID sql.NullInt64
	var groupName string
	var userEmail string
	var resolvedAt sql.NullTime
	var resolvedBy sql.NullInt64
	var resolvedByName string
	var resolvedRetryID sql.NullInt64
	var requestType sql.NullInt64

	if err := row.Scan(
		&item.ID,
		&item.CreatedAt,
		&item.Phase,
		&item.Type,
		&item.Owner,
		&item.Source,
		&item.Severity,
		&statusCode,
		&item.Platform,
		&item.Model,
		&item.IsRetryable,
		&item.RetryCount,
		&item.Resolved,
		&resolvedAt,
		&resolvedBy,
		&resolvedByName,
		&resolvedRetryID,
		&item.ClientRequestID,
		&item.RequestID,
		&item.Message,
		&userID,
		&userEmail,
		&apiKeyID,
		&accountID,
		&accountName,
		&groupID,
		&groupName,
		&clientIP,
		&item.RequestPath,
		&item.Stream,
		&item.InboundEndpoint,
		&item.UpstreamEndpoint,
		&item.RequestedModel,
		&item.UpstreamModel,
		&requestType,
		&item.UpstreamURL,
		&item.GeminiSurface,
		&item.BillingRuleID,
		&item.ProbeAction,
	); err != nil {
		return nil, err
	}

	applyOpsErrorLogListNulls(&item, statusCode, clientIP, userID, apiKeyID, accountID, groupID, resolvedAt, resolvedBy, resolvedRetryID, requestType)
	item.ResolvedByUserName = resolvedByName
	item.UserEmail = userEmail
	item.AccountName = accountName
	item.GroupName = groupName
	return &item, nil
}

func applyOpsErrorLogListNulls(item *service.OpsErrorLog, statusCode sql.NullInt64, clientIP sql.NullString, userID, apiKeyID, accountID, groupID sql.NullInt64, resolvedAt sql.NullTime, resolvedBy, resolvedRetryID, requestType sql.NullInt64) {
	item.StatusCode = int(statusCode.Int64)
	if clientIP.Valid {
		s := clientIP.String
		item.ClientIP = &s
	}
	if userID.Valid {
		v := userID.Int64
		item.UserID = &v
	}
	if apiKeyID.Valid {
		v := apiKeyID.Int64
		item.APIKeyID = &v
	}
	if accountID.Valid {
		v := accountID.Int64
		item.AccountID = &v
	}
	if groupID.Valid {
		v := groupID.Int64
		item.GroupID = &v
	}
	if resolvedAt.Valid {
		t := resolvedAt.Time
		item.ResolvedAt = &t
	}
	if resolvedBy.Valid {
		v := resolvedBy.Int64
		item.ResolvedByUserID = &v
	}
	if resolvedRetryID.Valid {
		v := resolvedRetryID.Int64
		item.ResolvedRetryID = &v
	}
	if requestType.Valid {
		v := int16(requestType.Int64)
		item.RequestType = &v
	}
}
