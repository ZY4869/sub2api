package repository

import (
	"context"
	"database/sql"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/proxy"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type sqlQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type proxyRepository struct {
	client *dbent.Client
	sql    sqlQuerier
}

func NewProxyRepository(client *dbent.Client, sqlDB *sql.DB) service.ProxyRepository {
	return newProxyRepositoryWithSQL(client, sqlDB)
}

func newProxyRepositoryWithSQL(client *dbent.Client, sqlq sqlQuerier) *proxyRepository {
	return &proxyRepository{client: client, sql: sqlq}
}

func (r *proxyRepository) Create(ctx context.Context, proxyIn *service.Proxy) error {
	builder := r.client.Proxy.Create().
		SetName(proxyIn.Name).
		SetProtocol(proxyIn.Protocol).
		SetHost(proxyIn.Host).
		SetPort(proxyIn.Port).
		SetStatus(proxyIn.Status).
		SetExpiryRemindDays(proxyIn.ExpiryRemindDays)
	if proxyIn.Username != "" {
		builder.SetUsername(proxyIn.Username)
	}
	if proxyIn.Password != "" {
		builder.SetPassword(proxyIn.Password)
	}
	if proxyIn.ExpiresAt != nil {
		builder.SetExpiresAt(*proxyIn.ExpiresAt)
	}
	if proxyIn.FallbackProxyID != nil {
		builder.SetFallbackProxyID(*proxyIn.FallbackProxyID)
	}

	created, err := builder.Save(ctx)
	if err == nil {
		applyProxyEntityToService(proxyIn, created)
	}
	return err
}

func (r *proxyRepository) GetByID(ctx context.Context, id int64) (*service.Proxy, error) {
	m, err := r.client.Proxy.Get(ctx, id)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrProxyNotFound
		}
		return nil, err
	}
	return proxyEntityToService(m), nil
}

func (r *proxyRepository) ListByIDs(ctx context.Context, ids []int64) ([]service.Proxy, error) {
	if len(ids) == 0 {
		return []service.Proxy{}, nil
	}

	proxies, err := r.client.Proxy.Query().
		Where(proxy.IDIn(ids...)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]service.Proxy, 0, len(proxies))
	for i := range proxies {
		out = append(out, *proxyEntityToService(proxies[i]))
	}
	return out, nil
}

func (r *proxyRepository) Update(ctx context.Context, proxyIn *service.Proxy) error {
	builder := r.client.Proxy.UpdateOneID(proxyIn.ID).
		SetName(proxyIn.Name).
		SetProtocol(proxyIn.Protocol).
		SetHost(proxyIn.Host).
		SetPort(proxyIn.Port).
		SetStatus(proxyIn.Status).
		SetExpiryRemindDays(proxyIn.ExpiryRemindDays)
	if proxyIn.Username != "" {
		builder.SetUsername(proxyIn.Username)
	} else {
		builder.ClearUsername()
	}
	if proxyIn.Password != "" {
		builder.SetPassword(proxyIn.Password)
	} else {
		builder.ClearPassword()
	}
	if proxyIn.ExpiresAt != nil {
		builder.SetExpiresAt(*proxyIn.ExpiresAt)
	} else {
		builder.ClearExpiresAt()
	}
	if proxyIn.FallbackProxyID != nil {
		builder.SetFallbackProxyID(*proxyIn.FallbackProxyID)
	} else {
		builder.ClearFallbackProxyID()
	}

	updated, err := builder.Save(ctx)
	if err == nil {
		applyProxyEntityToService(proxyIn, updated)
		return nil
	}
	if dbent.IsNotFound(err) {
		return service.ErrProxyNotFound
	}
	return err
}

func (r *proxyRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.client.Proxy.Delete().Where(proxy.IDEQ(id)).Exec(ctx)
	return err
}

func (r *proxyRepository) List(ctx context.Context, params pagination.PaginationParams) ([]service.Proxy, *pagination.PaginationResult, error) {
	return r.ListWithFilters(ctx, params, "", "", "")
}

// ListWithFilters lists proxies with optional filtering by protocol, status, and search query
func (r *proxyRepository) ListWithFilters(ctx context.Context, params pagination.PaginationParams, protocol, status, search string) ([]service.Proxy, *pagination.PaginationResult, error) {
	q := r.client.Proxy.Query()
	if protocol != "" {
		q = q.Where(proxy.ProtocolEQ(protocol))
	}
	if status != "" {
		q = q.Where(proxy.StatusEQ(status))
	}
	if search != "" {
		q = q.Where(proxy.NameContainsFold(search))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	proxies, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(proxy.FieldID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	outProxies := make([]service.Proxy, 0, len(proxies))
	for i := range proxies {
		outProxies = append(outProxies, *proxyEntityToService(proxies[i]))
	}

	return outProxies, paginationResultFromTotal(int64(total), params), nil
}

// ListWithFiltersAndAccountCount lists proxies with filters and includes account count per proxy
func (r *proxyRepository) ListWithFiltersAndAccountCount(ctx context.Context, params pagination.PaginationParams, protocol, status, search string) ([]service.ProxyWithAccountCount, *pagination.PaginationResult, error) {
	q := r.client.Proxy.Query()
	if protocol != "" {
		q = q.Where(proxy.ProtocolEQ(protocol))
	}
	if status != "" {
		q = q.Where(proxy.StatusEQ(status))
	}
	if search != "" {
		q = q.Where(proxy.NameContainsFold(search))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	proxies, err := q.
		Offset(params.Offset()).
		Limit(params.Limit()).
		Order(dbent.Desc(proxy.FieldID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Get account counts
	counts, err := r.GetAccountCountsForProxies(ctx)
	if err != nil {
		return nil, nil, err
	}

	outProxies := make([]service.Proxy, 0, len(proxies))
	for i := range proxies {
		proxyOut := proxyEntityToService(proxies[i])
		if proxyOut == nil {
			continue
		}
		outProxies = append(outProxies, *proxyOut)
	}

	// Build result with account counts
	result := make([]service.ProxyWithAccountCount, 0, len(outProxies))
	for i := range outProxies {
		proxyOut := outProxies[i]
		result = append(result, service.ProxyWithAccountCount{
			Proxy:        proxyOut,
			AccountCount: counts[proxyOut.ID],
		})
	}

	return result, paginationResultFromTotal(int64(total), params), nil
}

func (r *proxyRepository) ListActive(ctx context.Context) ([]service.Proxy, error) {
	proxies, err := r.client.Proxy.Query().
		Where(proxy.StatusEQ(service.StatusActive)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	outProxies := make([]service.Proxy, 0, len(proxies))
	for i := range proxies {
		outProxies = append(outProxies, *proxyEntityToService(proxies[i]))
	}
	return outProxies, nil
}

// ExistsByHostPortAuth checks if a proxy with the same host, port, username, and password exists
func (r *proxyRepository) ExistsByHostPortAuth(ctx context.Context, host string, port int, username, password string) (bool, error) {
	q := r.client.Proxy.Query().
		Where(proxy.HostEQ(host), proxy.PortEQ(port))

	if username == "" {
		q = q.Where(proxy.Or(proxy.UsernameIsNil(), proxy.UsernameEQ("")))
	} else {
		q = q.Where(proxy.UsernameEQ(username))
	}
	if password == "" {
		q = q.Where(proxy.Or(proxy.PasswordIsNil(), proxy.PasswordEQ("")))
	} else {
		q = q.Where(proxy.PasswordEQ(password))
	}

	count, err := q.Count(ctx)
	return count > 0, err
}

// CountAccountsByProxyID returns the number of accounts using a specific proxy
func (r *proxyRepository) CountAccountsByProxyID(ctx context.Context, proxyID int64) (int64, error) {
	var count int64
	if err := scanSingleRow(ctx, r.sql, "SELECT COUNT(*) FROM accounts WHERE proxy_id = $1 AND deleted_at IS NULL", []any{proxyID}, &count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *proxyRepository) ListAccountSummariesByProxyID(ctx context.Context, proxyID int64) ([]service.ProxyAccountSummary, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, name, platform, COALESCE(extra->>'gateway_protocol', ''), type, notes
		FROM accounts
		WHERE proxy_id = $1 AND deleted_at IS NULL
		ORDER BY id DESC
	`, proxyID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make([]service.ProxyAccountSummary, 0)
	for rows.Next() {
		var (
			id              int64
			name            string
			platform        string
			gatewayProtocol string
			accType         string
			notes           sql.NullString
		)
		if err := rows.Scan(&id, &name, &platform, &gatewayProtocol, &accType, &notes); err != nil {
			return nil, err
		}
		var notesPtr *string
		if notes.Valid {
			notesPtr = &notes.String
		}
		out = append(out, service.ProxyAccountSummary{
			ID:              id,
			Name:            name,
			Platform:        platform,
			GatewayProtocol: gatewayProtocol,
			Type:            accType,
			Notes:           notesPtr,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// GetAccountCountsForProxies returns a map of proxy ID to account count for all proxies
func (r *proxyRepository) GetAccountCountsForProxies(ctx context.Context) (counts map[int64]int64, err error) {
	rows, err := r.sql.QueryContext(ctx, "SELECT proxy_id, COUNT(*) AS count FROM accounts WHERE proxy_id IS NOT NULL AND deleted_at IS NULL GROUP BY proxy_id")
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			counts = nil
		}
	}()

	counts = make(map[int64]int64)
	for rows.Next() {
		var proxyID, count int64
		if err = rows.Scan(&proxyID, &count); err != nil {
			return nil, err
		}
		counts[proxyID] = count
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}

// ListActiveWithAccountCount returns all active proxies with account count, sorted by creation time descending
func (r *proxyRepository) ListActiveWithAccountCount(ctx context.Context) ([]service.ProxyWithAccountCount, error) {
	proxies, err := r.client.Proxy.Query().
		Where(proxy.StatusEQ(service.StatusActive)).
		Order(dbent.Desc(proxy.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	// Get account counts
	counts, err := r.GetAccountCountsForProxies(ctx)
	if err != nil {
		return nil, err
	}

	outProxies := make([]service.Proxy, 0, len(proxies))
	for i := range proxies {
		proxyOut := proxyEntityToService(proxies[i])
		if proxyOut == nil {
			continue
		}
		outProxies = append(outProxies, *proxyOut)
	}

	// Build result with account counts
	result := make([]service.ProxyWithAccountCount, 0, len(outProxies))
	for i := range outProxies {
		proxyOut := outProxies[i]
		result = append(result, service.ProxyWithAccountCount{
			Proxy:        proxyOut,
			AccountCount: counts[proxyOut.ID],
		})
	}

	return result, nil
}

func proxyEntityToService(m *dbent.Proxy) *service.Proxy {
	if m == nil {
		return nil
	}
	out := &service.Proxy{
		ID:               m.ID,
		Name:             m.Name,
		Protocol:         m.Protocol,
		Host:             m.Host,
		Port:             m.Port,
		Status:           m.Status,
		ExpiresAt:        m.ExpiresAt,
		ExpiryRemindDays: m.ExpiryRemindDays,
		FallbackProxyID:  m.FallbackProxyID,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
	if m.Username != nil {
		out.Username = *m.Username
	}
	if m.Password != nil {
		out.Password = *m.Password
	}
	return out
}

func applyProxyEntityToService(dst *service.Proxy, src *dbent.Proxy) {
	if dst == nil || src == nil {
		return
	}
	dst.ID = src.ID
	dst.ExpiresAt = src.ExpiresAt
	dst.ExpiryRemindDays = src.ExpiryRemindDays
	dst.FallbackProxyID = src.FallbackProxyID
	dst.CreatedAt = src.CreatedAt
	dst.UpdatedAt = src.UpdatedAt
}

func (r *proxyRepository) ListExpiredProxies(ctx context.Context, now time.Time) ([]service.Proxy, error) {
	proxies, err := r.client.Proxy.Query().
		Where(
			proxy.StatusEQ(service.StatusActive),
			proxy.ExpiresAtNotNil(),
			proxy.ExpiresAtLTE(now.UTC()),
		).
		Order(proxy.ByExpiresAt(), proxy.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]service.Proxy, 0, len(proxies))
	for i := range proxies {
		out = append(out, *proxyEntityToService(proxies[i]))
	}
	return out, nil
}

func (r *accountRepository) SwitchExpiredProxyAccounts(ctx context.Context, expired service.Proxy, fallback service.Proxy, switchedAt time.Time) ([]int64, error) {
	if expired.ID <= 0 || fallback.ID <= 0 || expired.ID == fallback.ID {
		return []int64{}, nil
	}
	rows, err := r.sql.QueryContext(ctx, `
		UPDATE accounts
		SET proxy_id = $2,
			original_proxy_id = COALESCE(original_proxy_id, $1),
			original_proxy_name = COALESCE(NULLIF(original_proxy_name, ''), $3),
			updated_at = NOW()
		WHERE proxy_id = $1
			AND deleted_at IS NULL
		RETURNING id
	`, expired.ID, fallback.ID, expired.Name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return ids, nil
	}
	payload := map[string]any{
		"account_ids":       ids,
		"expired_proxy_id":  expired.ID,
		"fallback_proxy_id": fallback.ID,
		"switched_at":       switchedAt.UTC().Format(time.RFC3339),
	}
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountBulkChanged, nil, nil, payload); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue proxy fallback failed: expired_proxy=%d fallback_proxy=%d err=%v", expired.ID, fallback.ID, err)
	}
	r.syncSchedulerAccountSnapshots(ctx, ids)
	return ids, nil
}

func (r *accountRepository) RestoreAccountOriginalProxy(ctx context.Context, accountID int64) (*service.AccountProxyRestoreResult, error) {
	if accountID <= 0 {
		return nil, service.ErrAccountNotFound
	}
	var result service.AccountProxyRestoreResult
	var previousFallbackID sql.NullInt64
	var previousFallbackName sql.NullString
	if err := scanSingleRow(ctx, r.sql, `
		WITH current_account AS (
			SELECT a.id, a.proxy_id, a.original_proxy_id, a.original_proxy_name, fp.name AS fallback_name
			FROM accounts a
			LEFT JOIN proxies fp ON fp.id = a.proxy_id AND fp.deleted_at IS NULL
			WHERE a.id = $1 AND a.deleted_at IS NULL
		),
		valid_original AS (
			SELECT ca.id AS account_id,
				ca.proxy_id AS fallback_id,
				ca.fallback_name,
				p.id AS original_id,
				p.name AS original_name
			FROM current_account ca
			JOIN proxies p ON p.id = ca.original_proxy_id AND p.deleted_at IS NULL
			WHERE ca.original_proxy_id IS NOT NULL
		),
		updated AS (
			UPDATE accounts a
			SET proxy_id = vo.original_id,
				original_proxy_id = NULL,
				original_proxy_name = NULL,
				updated_at = NOW()
			FROM valid_original vo
			WHERE a.id = vo.account_id
			RETURNING a.id, vo.original_id, vo.original_name, vo.fallback_id, vo.fallback_name
		)
		SELECT id, original_id, original_name, fallback_id, fallback_name
		FROM updated
	`, []any{accountID}, &result.AccountID, &result.RestoredProxyID, &result.RestoredProxyName, &previousFallbackID, &previousFallbackName); err != nil {
		if err == sql.ErrNoRows {
			exists, existsErr := r.accountExists(ctx, accountID)
			if existsErr != nil {
				return nil, existsErr
			}
			if !exists {
				return nil, service.ErrAccountNotFound
			}
			return nil, service.ErrProxyOriginalNotFound
		}
		return nil, err
	}
	if previousFallbackID.Valid {
		v := previousFallbackID.Int64
		result.PreviousFallbackID = &v
	}
	if previousFallbackName.Valid {
		result.PreviousFallbackName = previousFallbackName.String
	}
	if err := enqueueSchedulerOutbox(ctx, r.sql, service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil); err != nil {
		logger.LegacyPrintf("repository.account", "[SchedulerOutbox] enqueue restore original proxy failed: account=%d err=%v", accountID, err)
	}
	r.syncSchedulerAccountSnapshot(ctx, accountID)
	return &result, nil
}

func (r *accountRepository) accountExists(ctx context.Context, accountID int64) (bool, error) {
	var exists int
	if err := scanSingleRow(ctx, r.sql, `
		SELECT 1
		FROM accounts
		WHERE id = $1 AND deleted_at IS NULL
	`, []any{accountID}, &exists); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return exists == 1, nil
}
