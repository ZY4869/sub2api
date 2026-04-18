export default {
  ui: {
    routeTitles: {
      home: '首页',
      models: '模型库',
      apiDocs: 'API 文档',
      oauthCallback: '授权回调',
      linuxDoOAuthCallback: 'LinuxDo 授权回调',
      notFound: '页面不存在'
    },
    oauthCallback: {
      title: '授权回调',
      description: '请将 code 和 state 复制回管理端授权流程。',
      copy: '复制'
    },
    apiDocs: {
      eyebrow: '协议基线',
      title: 'API 文档',
      description: '以标准 Markdown 展示当前网关的请求契约、兼容范围和接入规则，方便人工与 AI 协同阅读。',
      protocolsTitle: '支持协议',
      pageTocTitle: '本页内容',
      articleEyebrow: '标准 Markdown',
      articleTitle: '当前生效文档',
      articleDescription: '这里展示的是登录后文档接口返回的当前生效 Markdown 原文。',
      tocTitle: '目录',
      copy: '一键复制 Markdown',
      copySuccess: 'Markdown 已复制',
      loading: '正在加载 API 文档...',
      loadFailed: '加载 API 文档失败',
      summary: {
        protocols: {
          label: '协议面',
          value: '7 个协议子页',
          description: '按通用接入、OpenAI、Claude、Gemini、Grok、Antigravity、Vertex / Batch 分页阅读。'
        },
        auth: {
          label: '认证',
          value: 'Bearer + Google 头',
          description: '明确不同客户端应使用的认证优先级、废弃参数和接入建议。'
        },
        sync: {
          label: '同步',
          value: '仓库基线 + 运行时覆盖',
          description: '仓库模板负责版本同步，管理员可以通过后台保存运行时覆盖内容。'
        }
      }
    },
    modelCatalog: {
      eyebrow: '公开模型库',
      description: '浏览当前可对外售卖且可用的模型目录，价格统一按出售价格层与倍率后的有效售价展示。',
      modelCount: '{count} 个模型',
      refresh: '刷新目录',
      refreshing: '刷新中...',
      loadFailed: '加载公开模型目录失败',
      empty: '当前筛选条件下没有可展示的模型。',
      filters: {
        provider: '供应商',
        protocol: '请求协议',
        multiplier: '倍率',
        all: '全部'
      },
      multiplier: {
        disabled: '未启用倍率',
        mixed: '混合倍率'
      },
      priceFields: {
        input: '输入',
        output: '输出',
        cache: '缓存',
        inputTier: '输入阶梯',
        outputTier: '输出阶梯',
        batchInput: 'Batch 输入',
        batchOutput: 'Batch 输出',
        batchCache: 'Batch 缓存',
        groundingSearch: 'Grounding Search',
        groundingMaps: 'Grounding Maps',
        embedding: 'Embedding',
        retrieval: 'Retrieval'
      },
      units: {
        perMillionTokens: '/ 1M Tokens',
        perImage: '/ 张',
        perRequest: '/ 次'
      }
    },
    usageWindow: {
      fiveHour: '5H',
      daily: '日',
      weekly: '周',
      total: '总',
      pro: 'Pro',
      flash: 'Flash'
    },
    platformType: {
      oauth: 'OAuth',
      token: '令牌',
      key: '密钥',
      sso: 'SSO',
      aws: 'AWS',
      privacy: '隐私',
      fail: '失败'
    },
    opsDimensions: {
      platform: '平台',
      groupId: '分组 ID',
      region: '地区'
    }
  }
}
