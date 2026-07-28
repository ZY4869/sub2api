export default {
  panelRateLimit: {
    title: '面板 API 限流',
    description: '控制后台、用户面板和公开面板接口的每分钟请求上限。',
    enabled: '启用面板限流',
    enabledHint: '默认关闭；开启后 Redis 故障时不阻断面板请求。',
    userRpm: '用户接口 RPM',
    heavyRpm: '聚合接口 RPM',
    publicIpRpm: '公开 IP RPM',
    exemptAdmin: '管理员豁免',
    exemptAdminHint: '开启后管理员账号不受用户和聚合接口限流影响。',
    saved: '面板限流设置保存成功',
    loadFailed: '加载面板限流设置失败',
    saveFailed: '保存面板限流设置失败',
  },
}
