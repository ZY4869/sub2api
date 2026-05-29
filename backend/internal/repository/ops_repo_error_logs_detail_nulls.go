package repository

import (
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type opsErrorLogDetailNulls struct {
	statusCode         sql.NullInt64
	upstreamStatusCode sql.NullInt64
	resolvedAt         sql.NullTime
	resolvedBy         sql.NullInt64
	resolvedRetryID    sql.NullInt64
	clientIP           sql.NullString
	userID             sql.NullInt64
	apiKeyID           sql.NullInt64
	accountID          sql.NullInt64
	groupID            sql.NullInt64
	authLatency        sql.NullInt64
	routingLatency     sql.NullInt64
	upstreamLatency    sql.NullInt64
	responseLatency    sql.NullInt64
	ttft               sql.NullInt64
	requestBodyBytes   sql.NullInt64
	requestType        sql.NullInt64
}

func applyOpsErrorLogDetailNulls(out *service.OpsErrorLogDetail, n opsErrorLogDetailNulls) {
	out.StatusCode = int(n.statusCode.Int64)
	if n.resolvedAt.Valid {
		t := n.resolvedAt.Time
		out.ResolvedAt = &t
	}
	if n.resolvedBy.Valid {
		v := n.resolvedBy.Int64
		out.ResolvedByUserID = &v
	}
	if n.resolvedRetryID.Valid {
		v := n.resolvedRetryID.Int64
		out.ResolvedRetryID = &v
	}
	if n.clientIP.Valid {
		s := n.clientIP.String
		out.ClientIP = &s
	}
	if n.upstreamStatusCode.Valid && n.upstreamStatusCode.Int64 > 0 {
		v := int(n.upstreamStatusCode.Int64)
		out.UpstreamStatusCode = &v
	}
	if n.userID.Valid {
		v := n.userID.Int64
		out.UserID = &v
	}
	if n.apiKeyID.Valid {
		v := n.apiKeyID.Int64
		out.APIKeyID = &v
	}
	if n.accountID.Valid {
		v := n.accountID.Int64
		out.AccountID = &v
	}
	if n.groupID.Valid {
		v := n.groupID.Int64
		out.GroupID = &v
	}
	if n.authLatency.Valid {
		v := n.authLatency.Int64
		out.AuthLatencyMs = &v
	}
	if n.routingLatency.Valid {
		v := n.routingLatency.Int64
		out.RoutingLatencyMs = &v
	}
	if n.upstreamLatency.Valid {
		v := n.upstreamLatency.Int64
		out.UpstreamLatencyMs = &v
	}
	if n.responseLatency.Valid {
		v := n.responseLatency.Int64
		out.ResponseLatencyMs = &v
	}
	if n.ttft.Valid {
		v := n.ttft.Int64
		out.TimeToFirstTokenMs = &v
	}
	if n.requestBodyBytes.Valid {
		v := int(n.requestBodyBytes.Int64)
		out.RequestBodyBytes = &v
	}
	if n.requestType.Valid {
		v := int16(n.requestType.Int64)
		out.RequestType = &v
	}
}
