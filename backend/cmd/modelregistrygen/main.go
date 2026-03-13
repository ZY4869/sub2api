package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

type tsSnapshot struct {
	ETag      string                        `json:"etag"`
	UpdatedAt string                        `json:"updated_at"`
	Models    []modelregistry.ModelEntry    `json:"models"`
	Presets   []modelregistry.PresetMapping `json:"presets"`
}

func main() {
	root, err := resolveRepoRoot()
	if err != nil {
		exitWithError(err)
	}

	models := modelregistry.SeedModels()
	presets := modelregistry.SeedPresets()
	builtAt := time.Now().UTC().Format(time.RFC3339)
	snapshot := tsSnapshot{
		ETag:      computeETag(models, presets),
		UpdatedAt: builtAt,
		Models:    normalizeModels(models),
		Presets:   append([]modelregistry.PresetMapping(nil), presets...),
	}

	writes := []struct {
		path    string
		content []byte
	}{
		{
			path:    filepath.Join(root, "docs", "models.md"),
			content: renderModelsMarkdown(models, builtAt),
		},
		{
			path:    filepath.Join(root, "docs", "compatibility.md"),
			content: renderCompatibilityMarkdown(models, builtAt),
		},
		{
			path:    filepath.Join(root, "frontend", "src", "generated", "modelRegistry.ts"),
			content: renderTypeScriptSnapshot(snapshot, builtAt),
		},
	}

	for _, write := range writes {
		if err := os.MkdirAll(filepath.Dir(write.path), 0o755); err != nil {
			exitWithError(err)
		}
		if err := os.WriteFile(write.path, write.content, 0o644); err != nil {
			exitWithError(err)
		}
		fmt.Println(write.path)
	}
}

func resolveRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if filepath.Base(wd) == "backend" {
		return filepath.Dir(wd), nil
	}
	if _, err := os.Stat(filepath.Join(wd, "backend")); err == nil {
		return wd, nil
	}
	return "", fmt.Errorf("unable to resolve repo root from %s", wd)
}

func computeETag(models []modelregistry.ModelEntry, presets []modelregistry.PresetMapping) string {
	payload, _ := json.Marshal(struct {
		Models  []modelregistry.ModelEntry    `json:"models"`
		Presets []modelregistry.PresetMapping `json:"presets"`
	}{
		Models:  models,
		Presets: presets,
	})
	sum := sha256.Sum256(payload)
	return `W/"` + hex.EncodeToString(sum[:]) + `"`
}

func renderModelsMarkdown(models []modelregistry.ModelEntry, builtAt string) []byte {
	grouped := map[string][]modelregistry.ModelEntry{}
	for _, model := range models {
		grouped[model.Provider] = append(grouped[model.Provider], model)
	}
	providers := make([]string, 0, len(grouped))
	for provider := range grouped {
		providers = append(providers, provider)
	}
	sort.Strings(providers)

	var buf bytes.Buffer
	buf.WriteString("# Model Registry Snapshot\n\n")
	buf.WriteString("Generated at " + builtAt + ".\n\n")
	for _, provider := range providers {
		items := grouped[provider]
		sort.Slice(items, func(i, j int) bool {
			if items[i].UIPriority == items[j].UIPriority {
				return items[i].ID < items[j].ID
			}
			return items[i].UIPriority < items[j].UIPriority
		})
		buf.WriteString("## " + title(provider) + "\n\n")
		buf.WriteString("| ID | Display | Platforms | Status | Replaced By |\n")
		buf.WriteString("| --- | --- | --- | --- | --- |\n")
		for _, item := range items {
			status := item.Status
			if status == "" {
				status = "stable"
			}
			buf.WriteString(fmt.Sprintf(
				"| `%s` | %s | %s | %s | %s |\n",
				item.ID,
				item.DisplayName,
				strings.Join(item.Platforms, ", "),
				status,
				orDash(item.ReplacedBy),
			))
		}
		buf.WriteString("\n")
	}
	return buf.Bytes()
}

func renderCompatibilityMarkdown(models []modelregistry.ModelEntry, builtAt string) []byte {
	items := append([]modelregistry.ModelEntry(nil), models...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].Provider == items[j].Provider {
			return items[i].ID < items[j].ID
		}
		return items[i].Provider < items[j].Provider
	})

	var buf bytes.Buffer
	buf.WriteString("# Model Compatibility Matrix\n\n")
	buf.WriteString("Generated at " + builtAt + ".\n\n")
	buf.WriteString("| Model | Anthropic | OpenAI | Gemini | Antigravity | Sora |\n")
	buf.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, item := range items {
		buf.WriteString(fmt.Sprintf(
			"| `%s` | %s | %s | %s | %s | %s |\n",
			item.ID,
			platformCheck(item, "anthropic"),
			platformCheck(item, "openai"),
			platformCheck(item, "gemini"),
			platformCheck(item, "antigravity"),
			platformCheck(item, "sora"),
		))
	}
	return buf.Bytes()
}

func renderTypeScriptSnapshot(snapshot tsSnapshot, builtAt string) []byte {
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		exitWithError(err)
	}

	var buf bytes.Buffer
	buf.WriteString("export interface ModelRegistryEntry {\n")
	buf.WriteString("  id: string\n")
	buf.WriteString("  display_name: string\n")
	buf.WriteString("  provider: string\n")
	buf.WriteString("  platforms: string[]\n")
	buf.WriteString("  protocol_ids: string[]\n")
	buf.WriteString("  aliases: string[]\n")
	buf.WriteString("  pricing_lookup_ids: string[]\n")
	buf.WriteString("  preferred_protocol_ids?: Record<string, string>\n")
	buf.WriteString("  modalities: string[]\n")
	buf.WriteString("  capabilities: string[]\n")
	buf.WriteString("  ui_priority: number\n")
	buf.WriteString("  exposed_in: string[]\n")
	buf.WriteString("  status?: string\n")
	buf.WriteString("  deprecated_at?: string\n")
	buf.WriteString("  replaced_by?: string\n")
	buf.WriteString("  deprecation_notice?: string\n")
	buf.WriteString("}\n\n")
	buf.WriteString("export interface ModelRegistryPreset {\n")
	buf.WriteString("  platform: string\n")
	buf.WriteString("  label: string\n")
	buf.WriteString("  from: string\n")
	buf.WriteString("  to: string\n")
	buf.WriteString("  color: string\n")
	buf.WriteString("  order?: number\n")
	buf.WriteString("}\n\n")
	buf.WriteString("export interface ModelRegistrySnapshot {\n")
	buf.WriteString("  etag: string\n")
	buf.WriteString("  updated_at: string\n")
	buf.WriteString("  models: ModelRegistryEntry[]\n")
	buf.WriteString("  presets: ModelRegistryPreset[]\n")
	buf.WriteString("}\n\n")
	buf.WriteString(fmt.Sprintf("export const generatedModelRegistryBuiltAt = %q\n\n", builtAt))
	buf.WriteString("export const generatedModelRegistrySnapshot: ModelRegistrySnapshot = ")
	buf.Write(payload)
	buf.WriteString("\n")
	return buf.Bytes()
}

func normalizeModels(models []modelregistry.ModelEntry) []modelregistry.ModelEntry {
	items := make([]modelregistry.ModelEntry, len(models))
	for index, model := range models {
		items[index] = modelregistry.ModelEntry{
			ID:                   model.ID,
			DisplayName:          model.DisplayName,
			Provider:             model.Provider,
			Platforms:            append([]string{}, model.Platforms...),
			ProtocolIDs:          append([]string{}, model.ProtocolIDs...),
			Aliases:              append([]string{}, model.Aliases...),
			PricingLookupIDs:     append([]string{}, model.PricingLookupIDs...),
			PreferredProtocolIDs: clonePreferredProtocolIDs(model.PreferredProtocolIDs),
			Modalities:           append([]string{}, model.Modalities...),
			Capabilities:         append([]string{}, model.Capabilities...),
			UIPriority:           model.UIPriority,
			ExposedIn:            append([]string{}, model.ExposedIn...),
			Status:               model.Status,
			DeprecatedAt:         model.DeprecatedAt,
			ReplacedBy:           model.ReplacedBy,
			DeprecationNotice:    model.DeprecationNotice,
		}
	}
	return items
}

func clonePreferredProtocolIDs(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func platformCheck(entry modelregistry.ModelEntry, platform string) string {
	for _, current := range entry.Platforms {
		if modelregistry.NormalizePlatform(current) == platform {
			return "Y"
		}
	}
	return "-"
}

func title(value string) string {
	if value == "" {
		return "Unknown"
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func orDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
