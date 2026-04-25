package service

import (
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func ProvideChannelMonitorRunnerService(
	db *sql.DB,
	repo ChannelMonitorRepository,
	historyRepo ChannelMonitorHistoryRepository,
	rollupRepo ChannelMonitorRollupRepository,
	aggRepo ChannelMonitorAggregationRepository,
	settingSvc *SettingService,
	encryptor SecretEncryptor,
	cfg *config.Config,
) *ChannelMonitorRunnerService {
	svc := NewChannelMonitorRunnerService(db, repo, historyRepo, rollupRepo, aggRepo, settingSvc, encryptor, cfg)
	svc.Start()
	return svc
}
