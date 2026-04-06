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

func mustWriteString(buf *bytes.Buffer, value string) {
	if _, err := buf.WriteString(value); err != nil {
		exitWithError(err)
	}
}

func mustWrite(buf *bytes.Buffer, value []byte) {
	if _, err := buf.Write(value); err != nil {
		exitWithError(err)
	}
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
	mustWriteString(&buf, "# Model Registry Snapshot\n\n")
	mustWriteString(&buf, "Generated at "+builtAt+".\n\n")
	for _, provider := range providers {
		items := grouped[provider]
		sort.Slice(items, func(i, j int) bool {
			if items[i].UIPriority == items[j].UIPriority {
				return items[i].ID < items[j].ID
			}
			return items[i].UIPriority < items[j].UIPriority
		})
		mustWriteString(&buf, "## "+title(provider)+"\n\n")
		mustWriteString(&buf, "| ID | Display | Platforms | Status | Replaced By |\n")
		mustWriteString(&buf, "| --- | --- | --- | --- | --- |\n")
		for _, item := range items {
			status := item.Status
			if status == "" {
				status = "stable"
			}
			mustWriteString(&buf, fmt.Sprintf(
				"| `%s` | %s | %s | %s | %s |\n",
				item.ID,
				item.DisplayName,
				strings.Join(item.Platforms, ", "),
				status,
				orDash(item.ReplacedBy),
			))
		}
		mustWriteString(&buf, "\n")
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
	mustWriteString(&buf, "# Model Compatibility Matrix\n\n")
	mustWriteString(&buf, "Generated at "+builtAt+".\n\n")
	mustWriteString(&buf, "| Model | Anthropic | OpenAI | Gemini | Antigravity |\n")
	mustWriteString(&buf, "| --- | --- | --- | --- | --- |\n")
	for _, item := range items {
		mustWriteString(&buf, fmt.Sprintf(
			"| `%s` | %s | %s | %s | %s |\n",
			item.ID,
			platformCheck(item, "anthropic"),
			platformCheck(item, "openai"),
			platformCheck(item, "gemini"),
			platformCheck(item, "antigravity"),
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
	mustWriteString(&buf, "export interface ModelRegistryEntry {\n")
	mustWriteString(&buf, "  id: string\n")
	mustWriteString(&buf, "  display_name: string\n")
	mustWriteString(&buf, "  provider: string\n")
	mustWriteString(&buf, "  platforms: string[]\n")
	mustWriteString(&buf, "  protocol_ids: string[]\n")
	mustWriteString(&buf, "  aliases: string[]\n")
	mustWriteString(&buf, "  pricing_lookup_ids: string[]\n")
	mustWriteString(&buf, "  preferred_protocol_ids?: Record<string, string>\n")
	mustWriteString(&buf, "  modalities: string[]\n")
	mustWriteString(&buf, "  capabilities: string[]\n")
	mustWriteString(&buf, "  ui_priority: number\n")
	mustWriteString(&buf, "  exposed_in: string[]\n")
	mustWriteString(&buf, "  status?: string\n")
	mustWriteString(&buf, "  deprecated_at?: string\n")
	mustWriteString(&buf, "  replaced_by?: string\n")
	mustWriteString(&buf, "  deprecation_notice?: string\n")
	mustWriteString(&buf, "}\n\n")
	mustWriteString(&buf, "export interface ModelRegistryPreset {\n")
	mustWriteString(&buf, "  platform: string\n")
	mustWriteString(&buf, "  label: string\n")
	mustWriteString(&buf, "  from: string\n")
	mustWriteString(&buf, "  to: string\n")
	mustWriteString(&buf, "  color: string\n")
	mustWriteString(&buf, "  order?: number\n")
	mustWriteString(&buf, "}\n\n")
	mustWriteString(&buf, "export interface ModelRegistrySnapshot {\n")
	mustWriteString(&buf, "  etag: string\n")
	mustWriteString(&buf, "  updated_at: string\n")
	mustWriteString(&buf, "  models: ModelRegistryEntry[]\n")
	mustWriteString(&buf, "  presets: ModelRegistryPreset[]\n")
	mustWriteString(&buf, "}\n\n")
	mustWriteString(&buf, fmt.Sprintf("export const generatedModelRegistryBuiltAt = %q\n\n", builtAt))
	mustWriteString(&buf, "export const generatedModelRegistrySnapshot: ModelRegistrySnapshot = ")
	mustWrite(&buf, payload)
	mustWriteString(&buf, "\n")
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
