package repository

import "context"

type opsRequestTraceSchema struct {
	HasGeminiSurface bool
	HasBillingRuleID bool
	HasProbeAction   bool
	HasUpstreamPath  bool
}

func defaultOpsRequestTraceSchema() opsRequestTraceSchema {
	return opsRequestTraceSchema{
		HasGeminiSurface: true,
		HasBillingRuleID: true,
		HasProbeAction:   true,
		HasUpstreamPath:  true,
	}
}

func (r *opsRepository) getOpsRequestTraceSchema(ctx context.Context) (opsRequestTraceSchema, error) {
	if r == nil || r.db == nil {
		return opsRequestTraceSchema{}, nil
	}

	r.requestTraceSchema.mu.RLock()
	if r.requestTraceSchema.loaded {
		schema := r.requestTraceSchema.value
		r.requestTraceSchema.mu.RUnlock()
		return schema, nil
	}
	r.requestTraceSchema.mu.RUnlock()

	r.requestTraceSchema.mu.Lock()
	defer r.requestTraceSchema.mu.Unlock()
	if r.requestTraceSchema.loaded {
		return r.requestTraceSchema.value, nil
	}

	schema := opsRequestTraceSchema{}
	var err error
	if schema.HasGeminiSurface, err = columnExistsSQL(ctx, r.db, "ops_request_traces", "gemini_surface"); err != nil {
		return opsRequestTraceSchema{}, err
	}
	if schema.HasBillingRuleID, err = columnExistsSQL(ctx, r.db, "ops_request_traces", "billing_rule_id"); err != nil {
		return opsRequestTraceSchema{}, err
	}
	if schema.HasProbeAction, err = columnExistsSQL(ctx, r.db, "ops_request_traces", "probe_action"); err != nil {
		return opsRequestTraceSchema{}, err
	}
	if schema.HasUpstreamPath, err = columnExistsSQL(ctx, r.db, "ops_request_traces", "upstream_path"); err != nil {
		return opsRequestTraceSchema{}, err
	}

	r.requestTraceSchema.value = schema
	r.requestTraceSchema.loaded = true
	return schema, nil
}

func opsRequestTraceOptionalStringExpr(column string, supported bool) string {
	if supported {
		return "COALESCE(" + column + ",'')"
	}
	return "''"
}
