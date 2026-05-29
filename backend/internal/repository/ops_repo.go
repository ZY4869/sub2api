package repository

import (
	"database/sql"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type opsRepository struct {
	db *sql.DB

	requestTraceSchema opsRequestTraceSchemaState
}

type opsRequestTraceSchemaState struct {
	mu     sync.RWMutex
	loaded bool
	value  opsRequestTraceSchema
}

func NewOpsRepository(db *sql.DB) service.OpsRepository {
	return &opsRepository{db: db}
}
