package service

import (
	"context"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

type AvailableTestModel struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	DisplayName  string `json:"display_name"`
	CreatedAt    string `json:"created_at"`
	CanonicalID  string `json:"canonical_id,omitempty"`
	Status       string `json:"status,omitempty"`
	DeprecatedAt string `json:"deprecated_at,omitempty"`
	ReplacedBy   string `json:"replaced_by,omitempty"`
}

type testModelCandidate struct {
	model      AvailableTestModel
	source     string
	uiPriority int
}

func BuildAvailableTestModels(ctx context.Context, account *Account, registry *ModelRegistryService) []AvailableTestModel {
	if account == nil || account.Platform == PlatformSora {
		return []AvailableTestModel{}
	}

	candidates, resolutionEntries := buildRegistryTestModelCandidates(ctx, account, registry)
	if len(candidates) == 0 {
		candidates, resolutionEntries = buildFallbackTestModelCandidates(ctx, account, registry)
	}
	return dedupeAndSortAvailableTestModels(candidates, resolutionEntries)
}

func buildRegistryTestModelCandidates(ctx context.Context, account *Account, registry *ModelRegistryService) ([]testModelCandidate, []modelregistry.ModelEntry) {
	if registry == nil {
		return nil, modelregistry.SeedModels()
	}

	details, err := registry.adminDetails(ctx)
	if err != nil {
		return nil, modelregistry.SeedModels()
	}

	resolutionEntries := make([]modelregistry.ModelEntry, 0, len(details))
	candidates := make([]testModelCandidate, 0, len(details))
	for _, detail := range details {
		if detail.Hidden || detail.Tombstoned || !detail.Available {
			continue
		}
		resolutionEntries = append(resolutionEntries, detail.ModelEntry)
		if !modelregistry.SupportsPlatform(detail.ModelEntry, account.Platform) {
			continue
		}
		if !modelregistry.HasExposure(detail.ModelEntry, "test") {
			continue
		}
		candidates = append(candidates, testModelCandidate{
			model: AvailableTestModel{
				ID:           detail.ID,
				Type:         "model",
				DisplayName:  firstNonEmptyTestModelLabel(detail.DisplayName, detail.ID),
				CreatedAt:    "",
				Status:       strings.TrimSpace(detail.Status),
				DeprecatedAt: strings.TrimSpace(detail.DeprecatedAt),
				ReplacedBy:   strings.TrimSpace(detail.ReplacedBy),
			},
			source:     strings.TrimSpace(detail.Source),
			uiPriority: detail.UIPriority,
		})
	}
	return candidates, resolutionEntries
}

func buildFallbackTestModelCandidates(ctx context.Context, account *Account, registry *ModelRegistryService) ([]testModelCandidate, []modelregistry.ModelEntry) {
	resolutionEntries := modelregistry.SeedModels()
	metadata := map[string]modelregistry.AdminModelDetail{}
	if registry != nil {
		if details, err := registry.adminDetails(ctx); err == nil {
			resolutionEntries = make([]modelregistry.ModelEntry, 0, len(details))
			for _, detail := range details {
				if detail.Hidden || detail.Tombstoned || !detail.Available {
					continue
				}
				resolutionEntries = append(resolutionEntries, detail.ModelEntry)
				metadata[normalizeRegistryID(detail.ID)] = detail
			}
		}
	}

	items := defaultTestModelCatalog(account)
	candidates := make([]testModelCandidate, 0, len(items))
	for _, item := range items {
		detail, ok := metadata[normalizeRegistryID(item.ID)]
		status := "stable"
		deprecatedAt := ""
		replacedBy := ""
		uiPriority := fallbackTestModelPriority(item.ID)
		source := "fallback"
		if ok {
			status = strings.TrimSpace(detail.Status)
			deprecatedAt = strings.TrimSpace(detail.DeprecatedAt)
			replacedBy = strings.TrimSpace(detail.ReplacedBy)
			uiPriority = detail.UIPriority
			source = strings.TrimSpace(detail.Source)
		}
		candidates = append(candidates, testModelCandidate{
			model: AvailableTestModel{
				ID:           item.ID,
				Type:         item.Type,
				DisplayName:  firstNonEmptyTestModelLabel(item.DisplayName, item.ID),
				CreatedAt:    item.CreatedAt,
				Status:       status,
				DeprecatedAt: deprecatedAt,
				ReplacedBy:   replacedBy,
			},
			source:     source,
			uiPriority: uiPriority,
		})
	}
	return candidates, resolutionEntries
}

func dedupeAndSortAvailableTestModels(candidates []testModelCandidate, resolutionEntries []modelregistry.ModelEntry) []AvailableTestModel {
	if len(candidates) == 0 {
		return []AvailableTestModel{}
	}

	indexEntries := resolutionEntries
	if len(indexEntries) == 0 {
		indexEntries = modelregistry.SeedModels()
	}
	index := modelregistry.BuildIndex(indexEntries)

	grouped := make(map[string][]testModelCandidate, len(candidates))
	for _, candidate := range candidates {
		canonicalID := normalizeRegistryID(candidate.model.ID)
		if resolved, ok := index.ResolveCanonicalID(candidate.model.ID); ok && resolved != "" {
			canonicalID = resolved
		}
		candidate.model.CanonicalID = canonicalID
		grouped[canonicalID] = append(grouped[canonicalID], candidate)
	}

	deduped := make([]AvailableTestModel, 0, len(grouped))
	for _, group := range grouped {
		sort.SliceStable(group, func(i, j int) bool {
			return compareTestModelCandidates(group[i], group[j]) < 0
		})
		deduped = append(deduped, group[0].model)
	}

	sort.SliceStable(deduped, func(i, j int) bool {
		return compareAvailableTestModels(deduped[i], deduped[j]) < 0
	})
	return deduped
}

func compareTestModelCandidates(left testModelCandidate, right testModelCandidate) int {
	if left.model.CanonicalID == left.model.ID && right.model.CanonicalID != right.model.ID {
		return -1
	}
	if right.model.CanonicalID == right.model.ID && left.model.CanonicalID != left.model.ID {
		return 1
	}
	if isDeprecatedTestModel(left.model) != isDeprecatedTestModel(right.model) {
		if !isDeprecatedTestModel(left.model) {
			return -1
		}
		return 1
	}
	if sourceRank := compareTestModelSource(left.source, right.source); sourceRank != 0 {
		return sourceRank
	}
	if left.uiPriority != right.uiPriority {
		return left.uiPriority - right.uiPriority
	}
	if len(left.model.ID) != len(right.model.ID) {
		return len(left.model.ID) - len(right.model.ID)
	}
	return strings.Compare(left.model.ID, right.model.ID)
}

func compareAvailableTestModels(left AvailableTestModel, right AvailableTestModel) int {
	if isDeprecatedTestModel(left) != isDeprecatedTestModel(right) {
		if !isDeprecatedTestModel(left) {
			return -1
		}
		return 1
	}
	leftPriority := fallbackTestModelPriority(left.ID)
	rightPriority := fallbackTestModelPriority(right.ID)
	if leftPriority != rightPriority {
		return leftPriority - rightPriority
	}
	leftLabel := strings.ToLower(firstNonEmptyTestModelLabel(left.DisplayName, left.ID))
	rightLabel := strings.ToLower(firstNonEmptyTestModelLabel(right.DisplayName, right.ID))
	if leftLabel != rightLabel {
		return strings.Compare(leftLabel, rightLabel)
	}
	return strings.Compare(left.ID, right.ID)
}

func compareTestModelSource(left string, right string) int {
	return testModelSourceRank(left) - testModelSourceRank(right)
}

func testModelSourceRank(source string) int {
	switch strings.TrimSpace(strings.ToLower(source)) {
	case "seed":
		return 0
	case "runtime":
		return 1
	default:
		return 2
	}
}

func isDeprecatedTestModel(model AvailableTestModel) bool {
	return strings.EqualFold(strings.TrimSpace(model.Status), "deprecated")
}

func fallbackTestModelPriority(modelID string) int {
	if entry, ok := modelregistry.SeedModelByID(modelID); ok && entry.UIPriority > 0 {
		return entry.UIPriority
	}
	return 5000
}

func firstNonEmptyTestModelLabel(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func defaultTestModelCatalog(account *Account) []AvailableTestModel {
	if account == nil {
		return []AvailableTestModel{}
	}

	switch account.Platform {
	case PlatformKiro:
		items := KiroBuiltinModelCatalog()
		result := make([]AvailableTestModel, 0, len(items))
		for _, item := range items {
			result = append(result, AvailableTestModel{
				ID:          item.ID,
				Type:        item.Type,
				DisplayName: item.DisplayName,
				CreatedAt:   item.CreatedAt,
				Status:      "stable",
			})
		}
		return result
	case PlatformGemini:
		result := make([]AvailableTestModel, 0, len(geminicli.DefaultModels))
		for _, item := range geminicli.DefaultModels {
			result = append(result, AvailableTestModel{
				ID:          item.ID,
				Type:        item.Type,
				DisplayName: item.DisplayName,
				CreatedAt:   item.CreatedAt,
				Status:      "stable",
			})
		}
		return result
	case PlatformAntigravity:
		items := antigravity.DefaultModels()
		result := make([]AvailableTestModel, 0, len(items))
		for _, item := range items {
			result = append(result, AvailableTestModel{
				ID:          item.ID,
				Type:        item.Type,
				DisplayName: item.DisplayName,
				CreatedAt:   item.CreatedAt,
				Status:      "stable",
			})
		}
		return result
	case PlatformOpenAI, PlatformCopilot:
		result := make([]AvailableTestModel, 0, len(openai.DefaultModels))
		for _, item := range openai.DefaultModels {
			result = append(result, AvailableTestModel{
				ID:          item.ID,
				Type:        item.Type,
				DisplayName: item.DisplayName,
				Status:      "stable",
			})
		}
		return result
	default:
		result := make([]AvailableTestModel, 0, len(claude.DefaultModels))
		for _, item := range claude.DefaultModels {
			result = append(result, AvailableTestModel{
				ID:          item.ID,
				Type:        item.Type,
				DisplayName: item.DisplayName,
				CreatedAt:   item.CreatedAt,
				Status:      "stable",
			})
		}
		return result
	}
}
