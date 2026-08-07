package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/redis/go-redis/v9"
)

type passkeySessionStore struct {
	rdb *redis.Client
}

func NewPasskeySessionStore(rdb *redis.Client) service.PasskeySessionStore {
	return &passkeySessionStore{rdb: rdb}
}

func (s *passkeySessionStore) Store(ctx context.Context, purpose, sessionID string, data webauthn.SessionData, ttl time.Duration) error {
	if s == nil || s.rdb == nil {
		return service.ErrPasskeySession
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return service.ErrPasskeySession
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, passkeySessionKey(purpose, sessionID), raw, ttl).Err()
}

func (s *passkeySessionStore) Take(ctx context.Context, purpose, sessionID string) (*webauthn.SessionData, error) {
	if s == nil || s.rdb == nil {
		return nil, service.ErrPasskeySession
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, service.ErrPasskeySession
	}
	raw, err := s.rdb.GetDel(ctx, passkeySessionKey(purpose, sessionID)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, service.ErrPasskeySession
		}
		return nil, err
	}
	var data webauthn.SessionData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("decode passkey session: %w", err)
	}
	return &data, nil
}

func passkeySessionKey(purpose, sessionID string) string {
	return "passkey:" + strings.TrimSpace(purpose) + ":" + strings.TrimSpace(sessionID)
}
