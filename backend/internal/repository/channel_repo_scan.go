package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/lib/pq"
)

type channelScanner interface {
	Scan(dest ...any) error
}

func scanChannelRows(rows *sql.Rows) ([]*model.Channel, error) {
	result := make([]*model.Channel, 0)
	for rows.Next() {
		channel, err := scanChannelRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, channel)
	}
	return result, rows.Err()
}

func scanChannelRow(scanner channelScanner) (*model.Channel, error) {
	channel := &model.Channel{}
	var mappingRaw []byte
	if err := scanner.Scan(
		&channel.ID,
		&channel.Name,
		&channel.Description,
		&channel.Status,
		&channel.RestrictModels,
		&channel.BillingModelSource,
		&mappingRaw,
		&channel.CreatedAt,
		&channel.UpdatedAt,
	); err != nil {
		return nil, err
	}
	channel.ModelMapping = map[string]map[string]string{}
	if len(mappingRaw) > 0 {
		_ = json.Unmarshal(mappingRaw, &channel.ModelMapping)
	}
	channel.GroupIDs = []int64{}
	channel.ModelPricing = []model.ChannelModelPricing{}
	return channel, nil
}

func (r *channelRepository) hydrateChannels(ctx context.Context, channels []*model.Channel) error {
	if len(channels) == 0 {
		return nil
	}

	channelByID := make(map[int64]*model.Channel, len(channels))
	channelIDs := make([]int64, 0, len(channels))
	for _, channel := range channels {
		channelByID[channel.ID] = channel
		channelIDs = append(channelIDs, channel.ID)
		channel.GroupIDs = []int64{}
		channel.ModelPricing = []model.ChannelModelPricing{}
	}

	groupRows, err := r.db.QueryContext(ctx, `
SELECT channel_id, group_id
FROM channel_groups
WHERE channel_id = ANY($1)
ORDER BY channel_id, group_id
`, pq.Array(channelIDs))
	if err != nil {
		return err
	}
	defer func() {
		_ = groupRows.Close()
	}()

	for groupRows.Next() {
		var channelID, groupID int64
		if err := groupRows.Scan(&channelID, &groupID); err != nil {
			return err
		}
		if channel := channelByID[channelID]; channel != nil {
			channel.GroupIDs = append(channel.GroupIDs, groupID)
		}
	}
	if err := groupRows.Err(); err != nil {
		return err
	}

	pricingRows, err := r.db.QueryContext(ctx, `
SELECT
	id,
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
FROM channel_model_pricing
WHERE channel_id = ANY($1)
ORDER BY channel_id, sort_order, id
`, pq.Array(channelIDs))
	if err != nil {
		return err
	}
	defer func() {
		_ = pricingRows.Close()
	}()

	pricingByID := make(map[int64]*model.ChannelModelPricing)
	pricingIDs := make([]int64, 0)
	for pricingRows.Next() {
		var pricing model.ChannelModelPricing
		var models []string
		var inputPrice sql.NullFloat64
		var outputPrice sql.NullFloat64
		var cacheWritePrice sql.NullFloat64
		var cacheReadPrice sql.NullFloat64
		var imageOutputPrice sql.NullFloat64
		var perRequestPrice sql.NullFloat64
		if err := pricingRows.Scan(
			&pricing.ID,
			&pricing.ChannelID,
			&pricing.Platform,
			pq.Array(&models),
			&pricing.BillingMode,
			&inputPrice,
			&outputPrice,
			&cacheWritePrice,
			&cacheReadPrice,
			&imageOutputPrice,
			&perRequestPrice,
			&pricing.SortOrder,
		); err != nil {
			return err
		}
		pricing.Models = models
		pricing.InputPrice = nullableFloatPtr(inputPrice)
		pricing.OutputPrice = nullableFloatPtr(outputPrice)
		pricing.CacheWritePrice = nullableFloatPtr(cacheWritePrice)
		pricing.CacheReadPrice = nullableFloatPtr(cacheReadPrice)
		pricing.ImageOutputPrice = nullableFloatPtr(imageOutputPrice)
		pricing.PerRequestPrice = nullableFloatPtr(perRequestPrice)
		pricing.Intervals = []model.ChannelPricingInterval{}

		channel := channelByID[pricing.ChannelID]
		if channel == nil {
			continue
		}
		channel.ModelPricing = append(channel.ModelPricing, pricing)
		pricingPtr := &channel.ModelPricing[len(channel.ModelPricing)-1]
		pricingByID[pricingPtr.ID] = pricingPtr
		pricingIDs = append(pricingIDs, pricingPtr.ID)
	}
	if err := pricingRows.Err(); err != nil {
		return err
	}
	if len(pricingIDs) == 0 {
		return nil
	}

	intervalRows, err := r.db.QueryContext(ctx, `
SELECT
	id,
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
FROM channel_pricing_intervals
WHERE pricing_id = ANY($1)
ORDER BY pricing_id, sort_order, id
`, pq.Array(pricingIDs))
	if err != nil {
		return err
	}
	defer func() {
		_ = intervalRows.Close()
	}()

	for intervalRows.Next() {
		var interval model.ChannelPricingInterval
		var maxTokens sql.NullInt64
		var inputPrice sql.NullFloat64
		var outputPrice sql.NullFloat64
		var cacheWritePrice sql.NullFloat64
		var cacheReadPrice sql.NullFloat64
		var perRequestPrice sql.NullFloat64
		if err := intervalRows.Scan(
			&interval.ID,
			&interval.PricingID,
			&interval.MinTokens,
			&maxTokens,
			&interval.TierLabel,
			&inputPrice,
			&outputPrice,
			&cacheWritePrice,
			&cacheReadPrice,
			&perRequestPrice,
			&interval.SortOrder,
		); err != nil {
			return err
		}
		interval.MaxTokens = nullableInt64Ptr(maxTokens)
		interval.InputPrice = nullableFloatPtr(inputPrice)
		interval.OutputPrice = nullableFloatPtr(outputPrice)
		interval.CacheWritePrice = nullableFloatPtr(cacheWritePrice)
		interval.CacheReadPrice = nullableFloatPtr(cacheReadPrice)
		interval.PerRequestPrice = nullableFloatPtr(perRequestPrice)
		if pricing := pricingByID[interval.PricingID]; pricing != nil {
			pricing.Intervals = append(pricing.Intervals, interval)
		}
	}
	return intervalRows.Err()
}

func nullableFloatPtr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	v := value.Float64
	return &v
}

func nullableInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	v := value.Int64
	return &v
}
