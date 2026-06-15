package service

import (
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type ChannelMonitorAccountDependencyBinding struct{}

func ProvideChannelMonitorService(
	db *sql.DB,
	repo ChannelMonitorRepository,
	historyRepo ChannelMonitorHistoryRepository,
	rollupRepo ChannelMonitorRollupRepository,
	settingSvc *SettingService,
	encryptor SecretEncryptor,
	cfg *config.Config,
	templateRepo ChannelMonitorTemplateRepository,
) *ChannelMonitorService {
	svc := NewChannelMonitorService(repo, historyRepo, rollupRepo, settingSvc, encryptor, cfg)
	svc.SetTemplateRepository(db, templateRepo)
	return svc
}

func BindChannelMonitorAccountDependencies(
	svc *ChannelMonitorService,
	accountRepo AccountRepository,
	testRunner channelMonitorAccountTestRunner,
) ChannelMonitorAccountDependencyBinding {
	svc.SetAccountMonitorDependencies(accountRepo, testRunner)
	return ChannelMonitorAccountDependencyBinding{}
}

func ProvideChannelMonitorRunnerService(
	db *sql.DB,
	repo ChannelMonitorRepository,
	historyRepo ChannelMonitorHistoryRepository,
	rollupRepo ChannelMonitorRollupRepository,
	aggRepo ChannelMonitorAggregationRepository,
	settingSvc *SettingService,
	encryptor SecretEncryptor,
	cfg *config.Config,
	accountRepo AccountRepository,
	accountTestService *AccountTestService,
) *ChannelMonitorRunnerService {
	svc := NewChannelMonitorRunnerService(db, repo, historyRepo, rollupRepo, aggRepo, settingSvc, encryptor, cfg)
	svc.SetAccountMonitorDependencies(accountRepo, accountTestService)
	svc.Start()
	return svc
}
