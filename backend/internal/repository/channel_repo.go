package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/model"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type channelRepository struct {
	db *sql.DB
}

func NewChannelRepository(db *sql.DB) service.ChannelRepository {
	return &channelRepository{db: db}
}

func (r *channelRepository) List(ctx context.Context, params pagination.PaginationParams, filters service.ChannelListFilters) ([]*model.Channel, *pagination.PaginationResult, error) {
	whereSQL, args := buildChannelListWhere(filters)
	countQuery := "SELECT COUNT(*) FROM channels c" + whereSQL

	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, nil, err
	}

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, params.Limit(), params.Offset())

	rows, err := r.db.QueryContext(ctx, `
SELECT
	c.id,
	c.name,
	COALESCE(c.description, ''),
	c.status,
	c.restrict_models,
	c.billing_model_source,
	c.model_mapping,
	c.created_at,
	c.updated_at
FROM channels c`+whereSQL+`
ORDER BY c.created_at DESC, c.id DESC
LIMIT $`+fmt.Sprint(len(queryArgs)-1)+` OFFSET $`+fmt.Sprint(len(queryArgs)),
		queryArgs...,
	)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	channels, err := scanChannelRows(rows)
	if err != nil {
		return nil, nil, err
	}
	if err := r.hydrateChannels(ctx, channels); err != nil {
		return nil, nil, err
	}

	return channels, &pagination.PaginationResult{
		Total:    total,
		Page:     params.Page,
		PageSize: params.Limit(),
		Pages:    paginationPages(total, params.Limit()),
	}, nil
}

func (r *channelRepository) GetByID(ctx context.Context, id int64) (*model.Channel, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT
	c.id,
	c.name,
	COALESCE(c.description, ''),
	c.status,
	c.restrict_models,
	c.billing_model_source,
	c.model_mapping,
	c.created_at,
	c.updated_at
FROM channels c
WHERE c.id = $1
`, id)

	channel, err := scanChannelRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrChannelNotFound
		}
		return nil, err
	}
	if err := r.hydrateChannels(ctx, []*model.Channel{channel}); err != nil {
		return nil, err
	}
	return channel, nil
}

func (r *channelRepository) GetActiveByGroupID(ctx context.Context, groupID int64) (*model.Channel, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT
	c.id,
	c.name,
	COALESCE(c.description, ''),
	c.status,
	c.restrict_models,
	c.billing_model_source,
	c.model_mapping,
	c.created_at,
	c.updated_at
FROM channels c
INNER JOIN channel_groups cg ON cg.channel_id = c.id
WHERE cg.group_id = $1
  AND c.status = $2
LIMIT 1
`, groupID, model.ChannelStatusActive)

	channel, err := scanChannelRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if err := r.hydrateChannels(ctx, []*model.Channel{channel}); err != nil {
		return nil, err
	}
	return channel, nil
}

func (r *channelRepository) Create(ctx context.Context, channel *model.Channel) (*model.Channel, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackIfNeeded(tx)

	modelMapping, err := json.Marshal(channel.ModelMapping)
	if err != nil {
		return nil, infraerrors.BadRequest("CHANNEL_MODEL_MAPPING_INVALID", "invalid channel model_mapping")
	}

	if err := tx.QueryRowContext(ctx, `
INSERT INTO channels (
	name,
	description,
	status,
	restrict_models,
	billing_model_source,
	model_mapping
)
VALUES ($1, NULLIF($2, ''), $3, $4, $5, $6::jsonb)
RETURNING id, created_at, updated_at
`,
		channel.Name,
		channel.Description,
		channel.Status,
		channel.RestrictModels,
		channel.BillingModelSource,
		string(modelMapping),
	).Scan(&channel.ID, &channel.CreatedAt, &channel.UpdatedAt); err != nil {
		return nil, translateChannelSQLError(err)
	}

	if err := r.replaceChannelRelations(ctx, tx, channel); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, translateChannelSQLError(err)
	}
	return r.GetByID(ctx, channel.ID)
}

func (r *channelRepository) Update(ctx context.Context, channel *model.Channel) (*model.Channel, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollbackIfNeeded(tx)

	modelMapping, err := json.Marshal(channel.ModelMapping)
	if err != nil {
		return nil, infraerrors.BadRequest("CHANNEL_MODEL_MAPPING_INVALID", "invalid channel model_mapping")
	}

	result, err := tx.ExecContext(ctx, `
UPDATE channels
SET
	name = $2,
	description = NULLIF($3, ''),
	status = $4,
	restrict_models = $5,
	billing_model_source = $6,
	model_mapping = $7::jsonb,
	updated_at = NOW()
WHERE id = $1
`,
		channel.ID,
		channel.Name,
		channel.Description,
		channel.Status,
		channel.RestrictModels,
		channel.BillingModelSource,
		string(modelMapping),
	)
	if err != nil {
		return nil, translateChannelSQLError(err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, service.ErrChannelNotFound
	}

	if err := r.replaceChannelRelations(ctx, tx, channel); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, translateChannelSQLError(err)
	}
	return r.GetByID(ctx, channel.ID)
}

func (r *channelRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM channels WHERE id = $1", id)
	if err != nil {
		return translateChannelSQLError(err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return service.ErrChannelNotFound
	}
	return nil
}

func (r *channelRepository) replaceChannelRelations(ctx context.Context, tx *sql.Tx, channel *model.Channel) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM channel_groups WHERE channel_id = $1", channel.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM channel_model_pricing WHERE channel_id = $1", channel.ID); err != nil {
		return err
	}

	for _, groupID := range channel.GroupIDs {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO channel_groups (channel_id, group_id)
VALUES ($1, $2)
`, channel.ID, groupID); err != nil {
			return translateChannelSQLError(err)
		}
	}

	for i := range channel.ModelPricing {
		pricing := &channel.ModelPricing[i]
		pricing.SortOrder = i

		if err := tx.QueryRowContext(ctx, `
INSERT INTO channel_model_pricing (
	channel_id,
	platform,
	models,
	billing_mode,
	input_price,
	output_price,
	cache_write_price,
	cache_read_price,
	image_output_price,
	per_request_price,
	sort_order
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id
`,
			channel.ID,
			pricing.Platform,
			pq.Array(pricing.Models),
			pricing.BillingMode,
			pricing.InputPrice,
			pricing.OutputPrice,
			pricing.CacheWritePrice,
			pricing.CacheReadPrice,
			pricing.ImageOutputPrice,
			pricing.PerRequestPrice,
			pricing.SortOrder,
		).Scan(&pricing.ID); err != nil {
			return translateChannelSQLError(err)
		}

		for j := range pricing.Intervals {
			interval := &pricing.Intervals[j]
			interval.SortOrder = j
			if _, err := tx.ExecContext(ctx, `
INSERT INTO channel_pricing_intervals (
	pricing_id,
	min_tokens,
	max_tokens,
	tier_label,
	input_price,
	output_price,
	cache_write_price,
	cache_read_price,
	per_request_price,
	sort_order
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
`,
				pricing.ID,
				interval.MinTokens,
				interval.MaxTokens,
				interval.TierLabel,
				interval.InputPrice,
				interval.OutputPrice,
				interval.CacheWritePrice,
				interval.CacheReadPrice,
				interval.PerRequestPrice,
				interval.SortOrder,
			); err != nil {
				return translateChannelSQLError(err)
			}
		}
	}

	return nil
}

func buildChannelListWhere(filters service.ChannelListFilters) (string, []any) {
	conditions := make([]string, 0, 2)
	args := make([]any, 0, 2)

	if status := strings.TrimSpace(filters.Status); status != "" {
		args = append(args, status)
		conditions = append(conditions, fmt.Sprintf("c.status = $%d", len(args)))
	}
	if search := strings.TrimSpace(filters.Search); search != "" {
		args = append(args, "%"+search+"%")
		conditions = append(conditions, fmt.Sprintf("(c.name ILIKE $%d OR COALESCE(c.description, '') ILIKE $%d)", len(args), len(args)))
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func rollbackIfNeeded(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

func paginationPages(total int64, pageSize int) int {
	if total == 0 {
		return 0
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	pages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		pages++
	}
	return pages
}

func translateChannelSQLError(err error) error {
	if err == nil {
		return nil
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Constraint {
		case "channels_unique_name":
			return service.ErrChannelAlreadyExists
		case "channel_groups_unique_group":
			return service.ErrChannelGroupConflict
		case "channel_groups_group_id_fkey":
			return infraerrors.BadRequest("CHANNEL_GROUP_NOT_FOUND", "one or more groups do not exist")
		}
	}
	return err
}
