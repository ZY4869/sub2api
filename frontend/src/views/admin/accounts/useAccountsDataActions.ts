export function useAccountsDataActions(ctx: any) {
  const {
    adminAPI, appStore, t, groups, reload, refreshArchivedPanel,
    showImportData, showCreate, showArchiveSelected, clearSelection,
    includeProxyOnExport, showExportDataDialog, exportingData, selIds, params
  } = ctx

  const refreshGroups = async () => {
    try {
      groups.value = await adminAPI.groups.getAll()
    } catch (error) {
      console.error('Failed to refresh groups:', error)
    }
  }
  const refreshListAndArchivedPanel = async () => {
    refreshArchivedPanel()
    await reload()
  }
  const handleReloadRequested = async () => {
    await refreshListAndArchivedPanel()
  }
  const handleDataImported = async () => {
    showImportData.value = false
    await refreshGroups()
    await refreshListAndArchivedPanel()
  }
  const handleCreated = async () => {
    showCreate.value = false
    await reload()
  }
  const handleArchivedAccounts = async () => {
    showArchiveSelected.value = false
    clearSelection()
    await refreshGroups()
    await refreshListAndArchivedPanel()
  }
  const formatExportTimestamp = () => {
    const now = new Date()
    const pad2 = (value: number) => String(value).padStart(2, '0')
    return `${now.getFullYear()}${pad2(now.getMonth() + 1)}${pad2(now.getDate())}${pad2(now.getHours())}${pad2(now.getMinutes())}${pad2(now.getSeconds())}`
  }
  const openExportDataDialog = () => {
    includeProxyOnExport.value = true
    showExportDataDialog.value = true
  }
  const handleExportData = async () => {
    if (exportingData.value) return
    exportingData.value = true
    try {
      const dataPayload = await adminAPI.accounts.exportData(
        selIds.value.length > 0
          ? { ids: selIds.value, includeProxies: includeProxyOnExport.value }
          : {
              includeProxies: includeProxyOnExport.value,
              filters: {
                platform: params.platform,
                type: params.type,
                status: params.status,
                search: params.search
              }
            }
      )
      const blob = new Blob([JSON.stringify(dataPayload, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `sub2api-account-${formatExportTimestamp()}.json`
      link.click()
      URL.revokeObjectURL(url)
      appStore.showSuccess(t('admin.accounts.dataExported'))
    } catch (error: any) {
      appStore.showError(error?.message || t('admin.accounts.dataExportFailed'))
    } finally {
      exportingData.value = false
      showExportDataDialog.value = false
    }
  }

  return {
    refreshGroups, refreshListAndArchivedPanel, handleReloadRequested,
    handleDataImported, handleCreated, handleArchivedAccounts,
    openExportDataDialog, handleExportData
  }
}
