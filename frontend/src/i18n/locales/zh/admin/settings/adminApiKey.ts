export default {
    adminApiKey: {
        title: "管理员 API Key",
        description: "用于外部系统集成的全局 API Key，拥有完整的管理员权限",
        notConfigured: "尚未配置管理员 API Key",
        configured: "管理员 API Key 已启用",
        currentKey: "当前密钥",
        regenerate: "重新生成",
        regenerating: "生成中...",
        delete: "删除",
        deleting: "删除中...",
        create: "创建密钥",
        creating: "创建中...",
        regenerateConfirm: "确定要重新生成吗？当前密钥将立即失效。",
        deleteConfirm: "确定要删除管理员 API Key 吗？外部集成将停止工作。",
        keyGenerated: "新的管理员 API Key 已生成",
        keyDeleted: "管理员 API Key 已删除",
        copyKey: "复制密钥",
        keyCopied: "密钥已复制到剪贴板",
        keyWarning: "此密钥仅显示一次，请立即复制保存。",
        securityWarning: "警告：此密钥拥有完整的管理员权限，请妥善保管。",
        usage: "使用方法：在请求头中添加 x-api-key: <your-admin-api-key>",
    }
}
