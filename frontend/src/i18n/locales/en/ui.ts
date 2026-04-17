export default {
  ui: {
    routeTitles: {
      home: 'Home',
      apiDocs: 'API Docs',
      oauthCallback: 'OAuth Callback',
      linuxDoOAuthCallback: 'LinuxDo OAuth Callback',
      notFound: '404 Not Found'
    },
    oauthCallback: {
      title: 'OAuth Callback',
      description: 'Copy the code and state back to the admin authorization flow.',
      copy: 'Copy'
    },
    apiDocs: {
      eyebrow: 'Protocol Reference',
      title: 'API Docs',
      description: 'Read the current gateway request contract, compatibility surface, and onboarding rules in standard Markdown.',
      protocolsTitle: 'Protocols',
      pageTocTitle: 'On This Page',
      articleEyebrow: 'Standard Markdown',
      articleTitle: 'Current Effective Document',
      articleDescription: 'This page renders the effective Markdown exactly as the authenticated API exposes it.',
      tocTitle: 'Contents',
      copy: 'Copy Markdown',
      copySuccess: 'Markdown copied',
      loading: 'Loading API docs...',
      loadFailed: 'Failed to load API docs',
      summary: {
        protocols: {
          label: 'Protocols',
          value: '7 protocol pages',
          description: 'Browse common onboarding, OpenAI, Claude, Gemini, Grok, Antigravity, and Vertex / Batch separately.'
        },
        auth: {
          label: 'Auth',
          value: 'Bearer + Google Headers',
          description: 'Shows the exact authentication priority, deprecated query params, and client-specific guidance.'
        },
        sync: {
          label: 'Sync',
          value: 'Code Baseline + Runtime Override',
          description: 'The repository template stays versioned, while admin updates can override it at runtime.'
        }
      }
    },
    usageWindow: {
      fiveHour: '5H',
      daily: '1D',
      weekly: '7D',
      total: 'Total',
      pro: 'Pro',
      flash: 'Flash'
    },
    platformType: {
      oauth: 'OAuth',
      token: 'Token',
      key: 'Key',
      sso: 'SSO',
      aws: 'AWS',
      privacy: 'Privacy',
      fail: 'Fail'
    },
    opsDimensions: {
      platform: 'Platform',
      groupId: 'Group ID',
      region: 'Region'
    }
  }
}
