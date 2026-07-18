package securityaudit

import (
	"context"
	"database/sql"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
)

func ProvidePromptService(
	settings service.SettingRepository,
	encryptor service.SecretEncryptor,
	db *sql.DB,
	redisClient *redis.Client,
) *PromptService {
	config := NewConfigManager(settings, encryptor)
	repo := NewRepository(db)
	payload := NewRedisPayloadStore(redisClient)
	scanner := NewOpenAICompatibleScanner()
	metrics := NewAtomicMetrics()
	svc := NewPromptService(config, repo, payload, scanner, metrics)
	svc.SetConfigInvalidator(NewRedisConfigInvalidator(redisClient))
	svc.Start(context.Background())
	return svc
}

var ProviderSet = wire.NewSet(
	ProvidePromptService,
	NewAdminHandler,
)
