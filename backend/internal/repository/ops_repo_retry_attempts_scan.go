package repository

import (
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type retryAttemptRow interface {
	Scan(dest ...any) error
}

type retryAttemptNulls struct {
	requestedBy       sql.NullInt64
	pinnedAccountID   sql.NullInt64
	startedAt         sql.NullTime
	finishedAt        sql.NullTime
	durationMs        sql.NullInt64
	success           sql.NullBool
	httpStatusCode    sql.NullInt64
	upstreamRequestID sql.NullString
	usedAccountID     sql.NullInt64
	responsePreview   sql.NullString
	responseTruncated sql.NullBool
	resultRequestID   sql.NullString
	resultErrorID     sql.NullInt64
	errorMessage      sql.NullString
}

func scanRetryAttemptRow(row retryAttemptRow) (*service.OpsRetryAttempt, error) {
	var item service.OpsRetryAttempt
	var pinnedAccountName string
	var usedAccountName string
	var n retryAttemptNulls

	if err := row.Scan(
		&item.ID,
		&item.CreatedAt,
		&n.requestedBy,
		&item.SourceErrorID,
		&item.Mode,
		&n.pinnedAccountID,
		&pinnedAccountName,
		&item.Status,
		&n.startedAt,
		&n.finishedAt,
		&n.durationMs,
		&n.success,
		&n.httpStatusCode,
		&n.upstreamRequestID,
		&n.usedAccountID,
		&usedAccountName,
		&n.responsePreview,
		&n.responseTruncated,
		&n.resultRequestID,
		&n.resultErrorID,
		&n.errorMessage,
	); err != nil {
		return nil, err
	}

	fillRetryAttemptFromNulls(&item, n)
	item.PinnedAccountName = pinnedAccountName
	item.UsedAccountName = usedAccountName
	return &item, nil
}

func fillRetryAttemptFromNulls(item *service.OpsRetryAttempt, n retryAttemptNulls) {
	item.RequestedByUserID = n.requestedBy.Int64
	if n.pinnedAccountID.Valid {
		v := n.pinnedAccountID.Int64
		item.PinnedAccountID = &v
	}
	if n.startedAt.Valid {
		t := n.startedAt.Time
		item.StartedAt = &t
	}
	if n.finishedAt.Valid {
		t := n.finishedAt.Time
		item.FinishedAt = &t
	}
	if n.durationMs.Valid {
		v := n.durationMs.Int64
		item.DurationMs = &v
	}
	if n.success.Valid {
		v := n.success.Bool
		item.Success = &v
	}
	if n.httpStatusCode.Valid {
		v := int(n.httpStatusCode.Int64)
		item.HTTPStatusCode = &v
	}
	if n.upstreamRequestID.Valid {
		item.UpstreamRequestID = &n.upstreamRequestID.String
	}
	if n.usedAccountID.Valid {
		v := n.usedAccountID.Int64
		item.UsedAccountID = &v
	}
	if n.responsePreview.Valid {
		item.ResponsePreview = &n.responsePreview.String
	}
	if n.responseTruncated.Valid {
		v := n.responseTruncated.Bool
		item.ResponseTruncated = &v
	}
	if n.resultRequestID.Valid {
		item.ResultRequestID = &n.resultRequestID.String
	}
	if n.resultErrorID.Valid {
		v := n.resultErrorID.Int64
		item.ResultErrorID = &v
	}
	if n.errorMessage.Valid {
		item.ErrorMessage = &n.errorMessage.String
	}
}
