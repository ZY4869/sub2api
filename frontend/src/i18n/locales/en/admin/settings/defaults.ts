export default {
    defaults: {
        title: "Default User Settings",
        description: "Default values for new users",
        defaultBalance: "Default Balance",
        defaultBalanceHint: "Initial balance for new users",
        defaultConcurrency: "Default Concurrency",
        defaultConcurrencyHint: "Maximum concurrent requests for new users",
        defaultApiKeyModelBindingMode: "Default API key creation mode",
        defaultApiKeyModelBindingModeHint: "Controls whether newly registered users default to group-based keys or public-model selection.",
        defaultApiKeyModeGroup: "Create keys by group",
        defaultApiKeyModeGroupHint: "New users can create keys by binding groups. This is the recommended default.",
        defaultApiKeyModePublicModel: "Create keys by public model",
        defaultApiKeyModePublicModelHint: "New users default to selecting from published public models when creating keys.",
        defaultSubscriptions: "Default Subscriptions",
        defaultSubscriptionsHint: "Auto-assign these subscriptions when a new user is created or registered",
        addDefaultSubscription: "Add Default Subscription",
        defaultSubscriptionsEmpty: "No default subscriptions configured.",
        defaultSubscriptionsDuplicate: "Duplicate subscription group: {groupId}. Each group can only appear once.",
        subscriptionGroup: "Subscription Group",
        subscriptionValidityDays: "Validity (days)",
    }
}
