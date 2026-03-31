package service

func (a *Account) IsBatchArchiveEnabled() bool {
	return a.getExtraBool("batch_archive_enabled")
}

func (a *Account) IsBatchArchiveAutoPrefetchEnabled() bool {
	return a.getExtraBool("batch_archive_auto_prefetch_enabled")
}

func (a *Account) GetBatchArchiveRetentionDays() int {
	if value := a.getExtraInt("batch_archive_retention_days"); value > 0 {
		return value
	}
	return googleBatchArchiveDefaultRetentionDays
}

func (a *Account) GetBatchArchiveBillingMode() string {
	switch a.getExtraString("batch_archive_billing_mode") {
	case GoogleBatchArchiveBillingModeArchiveCharge:
		return GoogleBatchArchiveBillingModeArchiveCharge
	default:
		return GoogleBatchArchiveBillingModeLogOnly
	}
}

func (a *Account) GetBatchArchiveDownloadPriceUSD() float64 {
	value := a.getExtraFloat64("batch_archive_download_price_usd")
	if value < 0 {
		return 0
	}
	return value
}

func (a *Account) AllowVertexBatchOverflow() bool {
	return a.getExtraBool("allow_vertex_batch_overflow")
}

func (a *Account) AcceptAIStudioBatchOverflow() bool {
	return a.getExtraBool("accept_aistudio_batch_overflow")
}
