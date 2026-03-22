package service

import (
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

const modelRegistrySortModeCategoryLatest = "category_latest"

type modelRegistryCategory string

const (
	modelRegistryCategoryText  modelRegistryCategory = "text"
	modelRegistryCategoryImage modelRegistryCategory = "image"
	modelRegistryCategoryVideo modelRegistryCategory = "video"
	modelRegistryCategoryAudio modelRegistryCategory = "audio"
	modelRegistryCategoryOther modelRegistryCategory = "other"
)

var modelRegistryCategoryOrder = map[modelRegistryCategory]int{
	modelRegistryCategoryText:  0,
	modelRegistryCategoryImage: 1,
	modelRegistryCategoryVideo: 2,
	modelRegistryCategoryAudio: 3,
	modelRegistryCategoryOther: 4,
}

func sortModelRegistryDetails(details []modelregistry.AdminModelDetail, sortMode string) {
	if strings.TrimSpace(strings.ToLower(sortMode)) != modelRegistrySortModeCategoryLatest {
		sort.Slice(details, func(i, j int) bool {
			if details[i].UIPriority == details[j].UIPriority {
				return details[i].ID < details[j].ID
			}
			return details[i].UIPriority < details[j].UIPriority
		})
		return
	}
	sort.Slice(details, func(i, j int) bool {
		leftCategory := modelRegistryDetailCategory(details[i])
		rightCategory := modelRegistryDetailCategory(details[j])
		if leftCategory != rightCategory {
			return modelRegistryCategoryOrder[leftCategory] < modelRegistryCategoryOrder[rightCategory]
		}
		if details[i].UIPriority == details[j].UIPriority {
			return details[i].ID < details[j].ID
		}
		return details[i].UIPriority < details[j].UIPriority
	})
}

func modelRegistryDetailCategory(detail modelregistry.AdminModelDetail) modelRegistryCategory {
	if containsAnyRegistryValue(detail.Capabilities, "video_generation", "video_understanding") {
		return modelRegistryCategoryVideo
	}
	if containsAnyRegistryValue(detail.Capabilities, "audio_generation", "audio_understanding") {
		return modelRegistryCategoryAudio
	}
	if containsAnyRegistryValue(detail.Capabilities, "image_generation", "vision") || containsAnyRegistryValue(detail.Modalities, "image") {
		return modelRegistryCategoryImage
	}
	if containsAnyRegistryValue(detail.Capabilities, "text") {
		return modelRegistryCategoryText
	}
	return modelRegistryCategoryOther
}

func containsAnyRegistryValue(values []string, targets ...string) bool {
	if len(values) == 0 || len(targets) == 0 {
		return false
	}
	lookup := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		target = strings.TrimSpace(strings.ToLower(target))
		if target == "" {
			continue
		}
		lookup[target] = struct{}{}
	}
	for _, value := range values {
		normalized := strings.TrimSpace(strings.ToLower(value))
		if _, exists := lookup[normalized]; exists {
			return true
		}
	}
	return false
}
