package repository

import "github.com/Wei-Shaw/sub2api/internal/service"

func platformFilterValues(platform string) []string {
	canonical := service.CanonicalizePlatformValue(platform)
	if canonical == "" {
		return nil
	}
	if canonical == service.PlatformBaiduDocumentAI {
		return []string{service.PlatformBaiduDocumentAI, "baidu"}
	}
	return []string{canonical}
}
