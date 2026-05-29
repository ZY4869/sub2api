package service

import (
	"os"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

func (s *PricingService) startUpdateScheduler() {
	// 定期检查哈希更新
	hashInterval := time.Duration(s.cfg.Pricing.HashCheckIntervalMinutes) * time.Minute
	if hashInterval < time.Minute {
		hashInterval = 10 * time.Minute
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(hashInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := s.syncWithRemote(); err != nil {
					logger.LegacyPrintf("service.pricing", "[Pricing] Sync failed: %v", err)
				}
			case <-s.stopCh:
				return
			}
		}
	}()

	logger.LegacyPrintf("service.pricing", "[Pricing] Update scheduler started (check every %v)", hashInterval)
}

// checkAndUpdatePricing 检查并更新价格数据
func (s *PricingService) checkAndUpdatePricing() error {
	pricingFile := s.getPricingFilePath()

	// 检查本地文件是否存在
	if _, err := os.Stat(pricingFile); os.IsNotExist(err) {
		logger.LegacyPrintf("service.pricing", "%s", "[Pricing] Local pricing file not found, downloading...")
		return s.downloadPricingData()
	}

	// 先加载本地文件（确保服务可用），再检查是否需要更新
	if err := s.loadPricingData(pricingFile); err != nil {
		logger.LegacyPrintf("service.pricing", "[Pricing] Failed to load local file, downloading: %v", err)
		return s.downloadPricingData()
	}

	// 如果配置了哈希URL，通过远程哈希检查是否有更新
	if s.cfg.Pricing.HashURL != "" {
		remoteHash, err := s.fetchRemoteHash()
		if err != nil {
			logger.LegacyPrintf("service.pricing", "[Pricing] Failed to fetch remote hash on startup: %v", err)
			return nil // 已加载本地文件，哈希获取失败不影响启动
		}

		s.mu.RLock()
		localHash := s.localHash
		s.mu.RUnlock()

		if localHash == "" || remoteHash != localHash {
			logger.LegacyPrintf("service.pricing", "[Pricing] Remote hash differs on startup (local=%s remote=%s), downloading...",
				localHash[:min(8, len(localHash))], remoteHash[:min(8, len(remoteHash))])
			if err := s.downloadPricingData(); err != nil {
				logger.LegacyPrintf("service.pricing", "[Pricing] Download failed, using existing file: %v", err)
			}
		}
		return nil
	}

	// 没有哈希URL时，基于文件年龄检查
	info, err := os.Stat(pricingFile)
	if err != nil {
		return nil // 已加载本地文件
	}

	fileAge := time.Since(info.ModTime())
	maxAge := time.Duration(s.cfg.Pricing.UpdateIntervalHours) * time.Hour

	if fileAge > maxAge {
		logger.LegacyPrintf("service.pricing", "[Pricing] Local file is %v old, updating...", fileAge.Round(time.Hour))
		if err := s.downloadPricingData(); err != nil {
			logger.LegacyPrintf("service.pricing", "[Pricing] Download failed, using existing file: %v", err)
		}
	}

	return nil
}

// syncWithRemote 与远程同步（基于哈希校验）
func (s *PricingService) syncWithRemote() error {
	// 如果配置了哈希URL，从远程获取哈希进行比对
	if s.cfg.Pricing.HashURL != "" {
		remoteHash, err := s.fetchRemoteHash()
		if err != nil {
			logger.LegacyPrintf("service.pricing", "[Pricing] Failed to fetch remote hash: %v", err)
			return nil // 哈希获取失败不影响正常使用
		}

		s.mu.RLock()
		localHash := s.localHash
		s.mu.RUnlock()

		if localHash == "" || remoteHash != localHash {
			logger.LegacyPrintf("service.pricing", "[Pricing] Remote hash differs (local=%s remote=%s), downloading new version...",
				localHash[:min(8, len(localHash))], remoteHash[:min(8, len(remoteHash))])
			return s.downloadPricingData()
		}
		logger.LegacyPrintf("service.pricing", "%s", "[Pricing] Hash check passed, no update needed")
		return nil
	}

	// 没有哈希URL时，基于时间检查
	pricingFile := s.getPricingFilePath()
	info, err := os.Stat(pricingFile)
	if err != nil {
		return s.downloadPricingData()
	}

	fileAge := time.Since(info.ModTime())
	maxAge := time.Duration(s.cfg.Pricing.UpdateIntervalHours) * time.Hour

	if fileAge > maxAge {
		logger.LegacyPrintf("service.pricing", "[Pricing] File is %v old, downloading...", fileAge.Round(time.Hour))
		return s.downloadPricingData()
	}

	return nil
}

// downloadPricingData 从远程下载价格数据
