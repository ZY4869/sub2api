package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type geminiPublicPathMetadata struct {
	version      string
	resource     string
	aliasUsed    bool
	upstreamPath string
}

func applyGeminiPublicPathMetadata(c *gin.Context, upstreamPathOverride string) {
	if c == nil || c.Request == nil || c.Request.URL == nil {
		return
	}
	ctx := service.EnsureRequestMetadata(c.Request.Context())
	meta := deriveGeminiPublicPathMetadata(c.Request.URL.Path)
	if trimmed := strings.TrimSpace(upstreamPathOverride); trimmed != "" {
		meta.upstreamPath = trimmed
	}
	service.SetGeminiPublicVersionMetadata(ctx, meta.version)
	service.SetGeminiPublicResourceMetadata(ctx, meta.resource)
	service.SetGeminiAliasUsedMetadata(ctx, meta.aliasUsed)
	service.SetGeminiUpstreamPathMetadata(ctx, meta.upstreamPath)
	c.Request = c.Request.WithContext(ctx)
}

func deriveGeminiPublicPathMetadata(path string) geminiPublicPathMetadata {
	trimmedOriginal := strings.TrimSpace(path)
	normalized := strings.ToLower(strings.Trim(trimmedOriginal, "/"))
	normalized = strings.TrimPrefix(normalized, "antigravity/")

	meta := geminiPublicPathMetadata{upstreamPath: trimmedOriginal}
	switch {
	case strings.HasPrefix(normalized, "v1/models"):
		meta.version = "v1"
		meta.resource = "models"
	case strings.HasPrefix(normalized, "v1alpha/authtokens"):
		meta.version = "v1alpha"
		meta.resource = "live_auth_tokens"
	case strings.HasPrefix(normalized, "v1beta/live/auth-token"),
		strings.HasPrefix(normalized, "v1beta/live/auth-tokens"),
		strings.HasPrefix(normalized, "v1beta/live/authtokens"):
		meta.version = "v1beta"
		meta.resource = "live_auth_tokens"
		meta.aliasUsed = true
	case strings.HasPrefix(normalized, "v1beta/live"):
		meta.version = "v1beta"
		meta.resource = "live"
	case strings.HasPrefix(normalized, "upload/v1beta/filesearchstores/"):
		meta.version = "v1beta"
		if strings.Contains(normalized, ":uploadtofilesearchstore") {
			meta.resource = "file_search_upload_operations"
		} else {
			meta.resource = "file_search_stores"
		}
	case strings.HasPrefix(normalized, "v1beta/filesearchstores/") && strings.Contains(normalized, "/upload/operations/"):
		meta.version = "v1beta"
		meta.resource = "file_search_upload_operations"
	case strings.HasPrefix(normalized, "v1beta/filesearchstores/") && strings.Contains(normalized, "/documents"):
		meta.version = "v1beta"
		meta.resource = "file_search_documents"
	case strings.HasPrefix(normalized, "v1beta/filesearchstores/") && strings.Contains(normalized, "/operations/"):
		meta.version = "v1beta"
		meta.resource = "file_search_operations"
	case strings.HasPrefix(normalized, "v1beta/filesearchstores"):
		meta.version = "v1beta"
		if strings.Contains(normalized, ":uploadtofilesearchstore") {
			meta.resource = "file_search_upload_operations"
		} else {
			meta.resource = "file_search_stores"
		}
	case strings.HasPrefix(normalized, "v1beta/documents"):
		meta.version = "v1beta"
		meta.resource = "file_search_documents"
		meta.aliasUsed = true
	case strings.HasPrefix(normalized, "v1beta/operations"):
		meta.version = "v1beta"
		meta.resource = "file_search_operations"
		meta.aliasUsed = true
	case strings.HasPrefix(normalized, "v1beta/files"):
		meta.version = "v1beta"
		meta.resource = "files"
	case strings.HasPrefix(normalized, "upload/v1beta/files"):
		meta.version = "v1beta"
		meta.resource = "files_upload"
	case strings.HasPrefix(normalized, "download/v1beta/files"):
		meta.version = "v1beta"
		meta.resource = "files_download"
	case strings.HasPrefix(normalized, "v1beta/batches"):
		meta.version = "v1beta"
		meta.resource = "batches"
	case strings.HasPrefix(normalized, "v1beta/cachedcontents"):
		meta.version = "v1beta"
		meta.resource = "cached_contents"
	case strings.HasPrefix(normalized, "v1beta/embeddings"):
		meta.version = "v1beta"
		meta.resource = "embeddings"
	case strings.HasPrefix(normalized, "v1beta/interactions"):
		meta.version = "v1beta"
		meta.resource = "interactions"
	case strings.HasPrefix(normalized, "v1beta/corpora/") && strings.Contains(normalized, "/permissions/"):
		meta.version = "v1beta"
		meta.resource = "corpora_permissions"
	case strings.HasPrefix(normalized, "v1beta/corpora/") && (strings.HasSuffix(normalized, "/operations") || strings.Contains(normalized, "/operations/")):
		meta.version = "v1beta"
		meta.resource = "corpora_operations"
	case strings.HasPrefix(normalized, "v1beta/corpora"):
		meta.version = "v1beta"
		meta.resource = "corpora"
	case strings.HasPrefix(normalized, "v1beta/dynamic"):
		meta.version = "v1beta"
		meta.resource = "dynamic"
	case strings.HasPrefix(normalized, "v1beta/generatedfiles/") && (strings.HasSuffix(normalized, "/operations") || strings.Contains(normalized, "/operations/")):
		meta.version = "v1beta"
		meta.resource = "generated_files_operations"
	case strings.HasPrefix(normalized, "v1beta/generatedfiles"):
		meta.version = "v1beta"
		meta.resource = "generated_files"
	case strings.HasPrefix(normalized, "v1beta/models/") && strings.Contains(normalized, "/operations"):
		meta.version = "v1beta"
		meta.resource = "model_operations"
	case strings.HasPrefix(normalized, "v1beta/tunedmodels/") && strings.Contains(normalized, "/permissions/"):
		meta.version = "v1beta"
		meta.resource = "tuned_models_permissions"
	case strings.HasPrefix(normalized, "v1beta/tunedmodels/") && strings.Contains(normalized, "/operations"):
		meta.version = "v1beta"
		meta.resource = "tuned_models_operations"
	case strings.HasPrefix(normalized, "v1beta/tunedmodels"):
		meta.version = "v1beta"
		meta.resource = "tuned_models"
	case strings.HasPrefix(normalized, "v1beta/openai/"):
		meta.version = "v1beta"
		meta.resource = "openai_compat"
	case strings.HasPrefix(normalized, "v1/projects/"):
		meta.version = "v1"
		if strings.Contains(normalized, "/publishers/google/models/") {
			meta.resource = "vertex_models"
		} else {
			meta.resource = "vertex_batch_jobs"
		}
	case strings.HasPrefix(normalized, "v1beta/models"):
		meta.version = "v1beta"
		meta.resource = "models"
	}
	return meta
}
