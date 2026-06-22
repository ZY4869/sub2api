export default {
    defaults: {
        title: "用户默认设置",
        description: "新用户的默认值",
        defaultBalance: "默认余额",
        defaultBalanceHint: "新用户的初始余额",
        defaultConcurrency: "默认并发数",
        defaultConcurrencyHint: "新用户的最大并发请求数",
        defaultApiKeyModelBindingMode: "新用户密钥创建默认模式",
        defaultApiKeyModelBindingModeHint: "决定新注册用户第一次创建或编辑 Key 时，默认按分组授权还是按对外模型选择授权。",
        defaultApiKeyModeGroup: "默认通过分组创建密钥",
        defaultApiKeyModeGroupHint: "新用户可按分组绑定创建 Key，这是推荐默认值。",
        defaultApiKeyModePublicModel: "选择对外展示模型创建密钥",
        defaultApiKeyModePublicModelHint: "新用户创建 Key 时默认从已发布的对外模型中选择。",
        defaultSubscriptions: "默认订阅列表",
        defaultSubscriptionsHint: "新用户创建或注册时自动分配这些订阅",
        addDefaultSubscription: "添加默认订阅",
        defaultSubscriptionsEmpty: "未配置默认订阅。新用户不会自动获得订阅套餐。",
        defaultSubscriptionsDuplicate: "默认订阅存在重复分组：{groupId}。每个分组只能出现一次。",
        subscriptionGroup: "订阅分组",
        subscriptionValidityDays: "有效期（天）",
    }
}
