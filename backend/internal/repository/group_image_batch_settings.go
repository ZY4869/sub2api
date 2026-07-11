package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func saveGroupImageBatchSettings(ctx context.Context, exec sqlExecutor, group *service.Group) error {
	if exec == nil || group == nil || group.ID <= 0 {
		return nil
	}
	providers := service.NormalizeImageBatchAllowList(group.ImageBatchAllowedProviders)
	models := service.NormalizeImageBatchAllowList(group.ImageBatchAllowedModels)
	maxItems := service.NormalizeImageBatchMaxItems(group.ImageBatchMaxItems)
	maxDownloadBytes := service.NormalizeImageBatchMaxDownloadBytes(group.ImageBatchMaxDownloadBytes)
	downloadConcurrency := service.NormalizeImageBatchDownloadConcurrency(group.ImageBatchDownloadConcurrency)
	_, err := exec.ExecContext(ctx, `
		UPDATE groups
		SET image_batch_enabled = $2,
			image_batch_allowed_providers = $3,
			image_batch_allowed_models = $4,
			image_batch_max_items = $5,
			image_batch_max_download_bytes = $6,
			image_batch_download_concurrency = $7
		WHERE id = $1 AND deleted_at IS NULL
	`, group.ID, group.ImageBatchEnabled, pq.Array(providers), pq.Array(models), maxItems, maxDownloadBytes, downloadConcurrency)
	if err != nil && isGroupImageBatchSettingsMissingError(err) && isDefaultGroupImageBatchSettings(group) {
		return nil
	}
	if err != nil {
		return err
	}
	group.ImageBatchAllowedProviders = providers
	group.ImageBatchAllowedModels = models
	group.ImageBatchMaxItems = maxItems
	group.ImageBatchMaxDownloadBytes = maxDownloadBytes
	group.ImageBatchDownloadConcurrency = downloadConcurrency
	return nil
}

func hydrateImageBatchSettingsForGroups(ctx context.Context, exec sqlExecutor, groups []*service.Group) error {
	if exec == nil || len(groups) == 0 {
		return nil
	}
	groupByID := make(map[int64]*service.Group, len(groups))
	ids := make([]int64, 0, len(groups))
	for _, group := range groups {
		if group == nil || group.ID <= 0 {
			continue
		}
		applyDefaultGroupImageBatchSettings(group)
		if _, exists := groupByID[group.ID]; exists {
			continue
		}
		groupByID[group.ID] = group
		ids = append(ids, group.ID)
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := exec.QueryContext(ctx, `
		SELECT id,
			image_batch_enabled,
			image_batch_allowed_providers,
			image_batch_allowed_models,
			image_batch_max_items,
			image_batch_max_download_bytes,
			image_batch_download_concurrency
		FROM groups
		WHERE id = ANY($1) AND deleted_at IS NULL
	`, pq.Array(ids))
	if err != nil {
		if isGroupImageBatchSettingsMissingError(err) {
			return nil
		}
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id int64
		var enabled bool
		var providers []string
		var models []string
		var maxItems int
		var maxDownloadBytes int64
		var downloadConcurrency int
		if err := rows.Scan(
			&id,
			&enabled,
			pq.Array(&providers),
			pq.Array(&models),
			&maxItems,
			&maxDownloadBytes,
			&downloadConcurrency,
		); err != nil {
			return err
		}
		group := groupByID[id]
		if group == nil {
			continue
		}
		group.ImageBatchEnabled = enabled
		group.ImageBatchAllowedProviders = service.NormalizeImageBatchAllowList(providers)
		group.ImageBatchAllowedModels = service.NormalizeImageBatchAllowList(models)
		group.ImageBatchMaxItems = service.NormalizeImageBatchMaxItems(maxItems)
		group.ImageBatchMaxDownloadBytes = service.NormalizeImageBatchMaxDownloadBytes(maxDownloadBytes)
		group.ImageBatchDownloadConcurrency = service.NormalizeImageBatchDownloadConcurrency(downloadConcurrency)
	}
	return rows.Err()
}

func hydrateImageBatchSettingsForGroupValues(ctx context.Context, exec sqlExecutor, groups []service.Group) error {
	if len(groups) == 0 {
		return nil
	}
	ptrs := make([]*service.Group, 0, len(groups))
	for i := range groups {
		ptrs = append(ptrs, &groups[i])
	}
	return hydrateImageBatchSettingsForGroups(ctx, exec, ptrs)
}

func applyDefaultGroupImageBatchSettings(group *service.Group) {
	if group == nil {
		return
	}
	group.ImageBatchAllowedProviders = service.NormalizeImageBatchAllowList(group.ImageBatchAllowedProviders)
	group.ImageBatchAllowedModels = service.NormalizeImageBatchAllowList(group.ImageBatchAllowedModels)
	group.ImageBatchMaxItems = service.NormalizeImageBatchMaxItems(group.ImageBatchMaxItems)
	group.ImageBatchMaxDownloadBytes = service.NormalizeImageBatchMaxDownloadBytes(group.ImageBatchMaxDownloadBytes)
	group.ImageBatchDownloadConcurrency = service.NormalizeImageBatchDownloadConcurrency(group.ImageBatchDownloadConcurrency)
}

func isDefaultGroupImageBatchSettings(group *service.Group) bool {
	if group == nil {
		return true
	}
	return !group.ImageBatchEnabled &&
		len(service.NormalizeImageBatchAllowList(group.ImageBatchAllowedProviders)) == 0 &&
		len(service.NormalizeImageBatchAllowList(group.ImageBatchAllowedModels)) == 0 &&
		service.NormalizeImageBatchMaxItems(group.ImageBatchMaxItems) == service.DefaultImageBatchMaxItems &&
		service.NormalizeImageBatchMaxDownloadBytes(group.ImageBatchMaxDownloadBytes) == service.DefaultImageBatchMaxDownloadBytes &&
		service.NormalizeImageBatchDownloadConcurrency(group.ImageBatchDownloadConcurrency) == service.DefaultImageBatchDownloadConcurrency
}

func isGroupImageBatchSettingsMissingError(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pq.Error
	if !errors.As(err, &pgErr) || pgErr.Code != "42703" {
		return false
	}
	return strings.Contains(strings.ToLower(pgErr.Message), "image_batch_")
}
