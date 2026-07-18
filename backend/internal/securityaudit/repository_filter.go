package securityaudit

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
)

func eventSelectSQL(includeFullText bool) string {
	fullPrompt := "''"
	if includeFullText {
		fullPrompt = "full_prompt"
	}
	return `SELECT id,job_id,request_id,user_id,username_snapshot,user_email_snapshot,api_key_id,
api_key_name_snapshot,group_id,group_name,provider,endpoint,protocol,model,prompt_hash,
redacted_preview,` + fullPrompt + `,stage,decision,risk_level,action,categories,matched_scanners,
scanner_scores,scanner_evidence,scanner_backend,scanner_version,guard_endpoint_id,policy_id,
policy_version,config_version,chunk_total,latency_ms,created_at FROM prompt_audit_events`
}

func normalizeEventFilter(filter EventFilter) EventFilter {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	if filter.PageSize > 100 {
		filter.PageSize = 100
	}
	filter.Decision = strings.TrimSpace(filter.Decision)
	filter.RiskLevel = strings.TrimSpace(filter.RiskLevel)
	filter.PromptHash = strings.TrimSpace(filter.PromptHash)
	filter.RequestID = strings.TrimSpace(filter.RequestID)
	filter.Protocol = strings.TrimSpace(filter.Protocol)
	filter.Endpoint = strings.TrimSpace(filter.Endpoint)
	filter.Keyword = strings.TrimSpace(filter.Keyword)
	return filter
}

func buildEventWhere(filter EventFilter) (string, []any) {
	builder := eventWhereBuilder{clauses: []string{"1=1"}}
	builder.addIf("decision", filter.Decision)
	builder.addIf("risk_level", filter.RiskLevel)
	builder.addPtr("user_id", filter.UserID)
	builder.addPtr("api_key_id", filter.APIKeyID)
	builder.addPtr("group_id", filter.GroupID)
	builder.addIf("prompt_hash", filter.PromptHash)
	builder.addIf("request_id", filter.RequestID)
	builder.addIf("protocol", filter.Protocol)
	builder.addIf("endpoint", filter.Endpoint)
	builder.addKeyword(filter.Keyword)
	builder.addTime("created_at >=", filter.StartAt)
	builder.addTime("created_at <=", filter.EndAt)
	return " WHERE " + strings.Join(builder.clauses, " AND "), builder.args
}

func filterHash(filter EventFilter, snapshotMaxID int64) string {
	type stable struct {
		Filter EventFilter `json:"filter"`
		MaxID  int64       `json:"max_id"`
	}
	raw, _ := json.Marshal(stable{Filter: filter, MaxID: snapshotMaxID})
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum[:8])
}
