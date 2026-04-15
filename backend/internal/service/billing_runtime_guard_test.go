package service

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBillingRuntimeCostEntryPointsStayBehindResolver(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)

	root := filepath.Dir(currentFile)
	allowed := map[string]struct{}{
		"billing_service.go":          {},
		"billing_runtime_resolver.go": {},
	}
	pattern := regexp.MustCompile(`\.(Calculate(?:Cost(?:WithServiceTier|WithLongContext)?|ImageCost(?:WithServiceTier)?|VideoRequestCost))\(`)

	var matches []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if _, ok := allowed[filepath.Base(path)]; ok {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		lines := strings.Split(string(content), "\n")
		for index, line := range lines {
			if pattern.MatchString(line) {
				matches = append(matches, filepath.Base(path)+":"+strconv.Itoa(index+1)+": "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	require.NoError(t, err)
	require.Empty(t, matches, "resolver 外不应再直接调用底层 Calculate* 计费函数")
}
