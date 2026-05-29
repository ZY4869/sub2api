package repository

import (
	"database/sql"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type apiKeyRepository struct {
	client *dbent.Client
	sql    sqlExecutor
}

func NewAPIKeyRepository(client *dbent.Client, sqlDB *sql.DB) service.APIKeyRepository {
	return newAPIKeyRepositoryWithSQL(client, sqlDB)
}

func newAPIKeyRepositoryWithSQL(client *dbent.Client, sqlq sqlExecutor) *apiKeyRepository {
	return &apiKeyRepository{client: client, sql: sqlq}
}

func (r *apiKeyRepository) BillingHoldRepository() service.BillingHoldRepository {
	db, ok := r.sql.(*sql.DB)
	if !ok || db == nil {
		return nil
	}
	return NewBillingHoldRepository(db)
}

func (r *apiKeyRepository) activeQuery() *dbent.APIKeyQuery {
	// 默认过滤已软删除记录，避免删除后仍被查询到。
	return r.client.APIKey.Query().Where(apikey.DeletedAtIsNil())
}
