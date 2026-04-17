export default {
  ui: {
    routeTitles: {
      home: '首页',
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
      description: '以标准 Markdown 展示当前网关的请求契约、兼容范围和接入规则，方便人类与 AI 共同阅读。',
      protocolsTitle: '支持协议',
      pageTocTitle: '本页内容',
      articleEyebrow: '标准 Markdown',
      articleTitle: '当前生效文档',
      articleDescription: '这里渲染的是登录后文档接口返回的当前生效 Markdown 原文。',
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
