package service

import "log/slog"

func (s *defaultOpenAIAccountScheduler) logLoadBalanceSelection(
	phase string,
	req OpenAIAccountScheduleRequest,
	candidate openAIAccountCandidateScore,
	candidateCount int,
	topK int,
	loadSkew float64,
) {
	if candidate.account == nil {
		return
	}
	window, utilization, resetAt := accountUsagePressureLogValues(candidate.pressure)
	concurrencyUtil := clamp01(calcConcurrencyUtilization(candidate.loadInfo.CurrentConcurrency, candidate.account.Concurrency))
	slog.Debug(
		"openai_ws_account_scheduler_selection",
		"phase", phase,
		"group_id", derefGroupID(req.GroupID),
		"model", req.RequestedModel,
		"session", shortSessionHash(req.SessionHash),
		"account_id", candidate.account.ID,
		"account_type", candidate.account.Type,
		"priority", candidate.account.Priority,
		"selection_concurrency", resolveOpenAIAccountSelectionConcurrency(candidate.account),
		"plan_type", candidate.planType,
		"plan_rank", candidate.planRank,
		"candidate_count", candidateCount,
		"top_k", topK,
		"load_skew", loadSkew,
		"score", candidate.score,
		"load_rate", candidate.loadInfo.LoadRate,
		"current_concurrency", candidate.loadInfo.CurrentConcurrency,
		"max_concurrency", candidate.account.Concurrency,
		"concurrency_utilization", concurrencyUtil,
		"waiting_count", candidate.loadInfo.WaitingCount,
		"pressure_scope", candidate.pressureScope,
		"pressure_window", window,
		"pressure_utilization", utilization,
		"pressure_reset_at", resetAt,
	)
}
