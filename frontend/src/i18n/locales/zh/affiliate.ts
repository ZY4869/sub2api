export default {
  affiliate: {
    title: '邀请返利',
    description: '邀请好友注册并获得返利，返利可转入余额。',

    disabled: {
      title: '邀请返利未开启',
      desc: '当前站点未开启邀请返利功能，请联系管理员。'
    },

    stats: {
      invitees: '邀请人数',
      available: '可转余额',
      frozen: '冻结中',
      lifetime: '累计返利'
    },

    myCode: '我的返利码',
    myCodeHint: '分享给好友注册时填写。',
    inviteLink: '邀请链接',
    inviteLinkHint: '可直接分享链接，打开注册页会自动填充返利码。',

    copied: {
      code: '已复制返利码',
      link: '已复制邀请链接'
    },

    transfer: {
      button: '转入余额',
      disabled: '转入功能已关闭，请联系管理员。',
      noBalance: '当前没有可转入的返利余额。',
      success: '已转入余额：{amount}',
      none: '本次无可转入金额。'
    },

    rulesTitle: '规则',
    rules: {
      rate: '返利比例',
      freeze: '冻结期',
      duration: '有效期',
      durationForever: '永久',
      cap: '单人上限',
      capUnlimited: '不限',
      switches: '返利触发：消耗 {usage} · 入金 {topup}'
    }
  }
}

