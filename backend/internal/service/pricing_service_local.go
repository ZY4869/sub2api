package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	pricingbundle "github.com/Wei-Shaw/sub2api/resources/model-pricing"
)

func (s *PricingService) loadPricingDataFromBytes(source string, data []byte, modTime time.Time) error {
	if len(data) == 0 {
		return fmt.Errorf("pricing data is empty")
	}

	pricingData, err := s.parsePricingData(data)
	if err != nil {
		return fmt.Errorf("parse pricing data: %w", err)
	}

	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	if modTime.IsZero() {
		modTime = time.Now()
	}

	s.mu.Lock()
	s.pricingData = pricingData
	s.localHash = hashStr
	s.lastUpdated = modTime
	s.mu.Unlock()

	logger.LegacyPrintf("service.pricing", "[Pricing] Loaded %d models from %s", len(pricingData), source)
	return nil
}

// loadPricingData 从本地文件加载价格数据
func (s *PricingService) loadPricingData(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file failed: %w", err)
	}

	info, _ := os.Stat(filePath)
	modTime := time.Now()
	if info != nil {
		modTime = info.ModTime()
	}
	return s.loadPricingDataFromBytes(filePath, data, modTime)
}

// useFallbackPricing 使用回退价格文件
func (s *PricingService) useFallbackPricing() error {
	fallbackFile := strings.TrimSpace(s.cfg.Pricing.FallbackFile)
	if fallbackFile != "" {
		data, err := os.ReadFile(fallbackFile)
		if err == nil {
			logger.LegacyPrintf("service.pricing", "[Pricing] Using fallback file: %s", fallbackFile)

			pricingFile := s.getPricingFilePath()
			if err := os.WriteFile(pricingFile, data, 0644); err != nil {
				logger.LegacyPrintf("service.pricing", "[Pricing] Failed to copy fallback: %v", err)
			}

			modTime := time.Now()
			if info, err := os.Stat(fallbackFile); err == nil && info != nil {
				modTime = info.ModTime()
			}

			if err := s.loadPricingDataFromBytes(fallbackFile, data, modTime); err == nil {
				return nil
			}
			logger.LegacyPrintf("service.pricing", "[Pricing] Failed to load fallback file, using embedded: %v", err)
		} else {
			logger.LegacyPrintf("service.pricing", "[Pricing] Fallback file unavailable, using embedded: %v", err)
		}
	}

	if len(pricingbundle.FallbackPricingJSON) == 0 {
		return fmt.Errorf("embedded fallback pricing data is empty")
	}

	logger.LegacyPrintf("service.pricing", "%s", "[Pricing] Using embedded fallback pricing data")

	pricingFile := s.getPricingFilePath()
	if err := os.WriteFile(pricingFile, pricingbundle.FallbackPricingJSON, 0644); err != nil {
		logger.LegacyPrintf("service.pricing", "[Pricing] Failed to copy embedded fallback: %v", err)
	}

	return s.loadPricingDataFromBytes(
		"embedded://resources/model-pricing/model_prices_and_context_window.json",
		pricingbundle.FallbackPricingJSON,
		time.Now(),
	)
}
