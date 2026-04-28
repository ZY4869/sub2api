export type AdminModuleIcon =
  | 'badge'
  | 'bell'
  | 'chart'
  | 'cog'
  | 'creditCard'
  | 'gift'
  | 'globe'
  | 'home'
  | 'mail'
  | 'server'
  | 'shield'
  | 'userPlus'
  | 'users'

export type AdminModuleAccent = 'emerald' | 'sky' | 'violet' | 'amber' | 'rose' | 'cyan'

export interface AdminModuleCard {
  id: string
  titleKey: string
  descriptionKey: string
  to: string
  icon: AdminModuleIcon
  accent: AdminModuleAccent
  hideInSimpleMode?: boolean
}

export interface AdminModuleSection {
  id: string
  titleKey: string
  descriptionKey: string
  cards: AdminModuleCard[]
}

export const adminModuleSections: AdminModuleSection[] = [
  {
    id: 'growth',
    titleKey: 'admin.modules.sections.growth.title',
    descriptionKey: 'admin.modules.sections.growth.description',
    cards: [
      {
        id: 'promo-codes',
        titleKey: 'admin.modules.cards.promoCodes.title',
        descriptionKey: 'admin.modules.cards.promoCodes.description',
        to: '/admin/promo-codes',
        icon: 'gift',
        accent: 'emerald',
        hideInSimpleMode: true
      },
      {
        id: 'redeem-codes',
        titleKey: 'admin.modules.cards.redeemCodes.title',
        descriptionKey: 'admin.modules.cards.redeemCodes.description',
        to: '/admin/redeem',
        icon: 'badge',
        accent: 'sky',
        hideInSimpleMode: true
      },
      {
        id: 'registration',
        titleKey: 'admin.modules.cards.registration.title',
        descriptionKey: 'admin.modules.cards.registration.description',
        to: '/admin/settings?tab=security',
        icon: 'shield',
        accent: 'violet'
      },
      {
        id: 'invitation',
        titleKey: 'admin.modules.cards.invitation.title',
        descriptionKey: 'admin.modules.cards.invitation.description',
        to: '/admin/affiliates',
        icon: 'userPlus',
        accent: 'amber',
        hideInSimpleMode: true
      }
    ]
  },
  {
    id: 'channels',
    titleKey: 'admin.modules.sections.channels.title',
    descriptionKey: 'admin.modules.sections.channels.description',
    cards: [
      {
        id: 'proxies',
        titleKey: 'admin.modules.cards.proxies.title',
        descriptionKey: 'admin.modules.cards.proxies.description',
        to: '/admin/proxies',
        icon: 'server',
        accent: 'cyan'
      },
      {
        id: 'subscriptions',
        titleKey: 'admin.modules.cards.subscriptions.title',
        descriptionKey: 'admin.modules.cards.subscriptions.description',
        to: '/admin/subscriptions',
        icon: 'creditCard',
        accent: 'emerald',
        hideInSimpleMode: true
      },
      {
        id: 'channel-monitors',
        titleKey: 'admin.modules.cards.channelMonitors.title',
        descriptionKey: 'admin.modules.cards.channelMonitors.description',
        to: '/admin/channel-monitors',
        icon: 'chart',
        accent: 'rose',
        hideInSimpleMode: true
      },
      {
        id: 'channels',
        titleKey: 'admin.modules.cards.channels.title',
        descriptionKey: 'admin.modules.cards.channels.description',
        to: '/admin/channels',
        icon: 'globe',
        accent: 'sky',
        hideInSimpleMode: true
      }
    ]
  },
  {
    id: 'settings',
    titleKey: 'admin.modules.sections.settings.title',
    descriptionKey: 'admin.modules.sections.settings.description',
    cards: [
      {
        id: 'settings-general',
        titleKey: 'admin.modules.cards.settingsGeneral.title',
        descriptionKey: 'admin.modules.cards.settingsGeneral.description',
        to: '/admin/settings?tab=general',
        icon: 'home',
        accent: 'emerald'
      },
      {
        id: 'settings-security',
        titleKey: 'admin.modules.cards.settingsSecurity.title',
        descriptionKey: 'admin.modules.cards.settingsSecurity.description',
        to: '/admin/settings?tab=security',
        icon: 'shield',
        accent: 'violet'
      },
      {
        id: 'settings-users',
        titleKey: 'admin.modules.cards.settingsUsers.title',
        descriptionKey: 'admin.modules.cards.settingsUsers.description',
        to: '/admin/settings?tab=users',
        icon: 'users',
        accent: 'sky'
      },
      {
        id: 'settings-gateway',
        titleKey: 'admin.modules.cards.settingsGateway.title',
        descriptionKey: 'admin.modules.cards.settingsGateway.description',
        to: '/admin/settings?tab=gateway',
        icon: 'server',
        accent: 'cyan'
      },
      {
        id: 'settings-notification',
        titleKey: 'admin.modules.cards.settingsNotification.title',
        descriptionKey: 'admin.modules.cards.settingsNotification.description',
        to: '/admin/settings?tab=notification',
        icon: 'bell',
        accent: 'amber'
      },
      {
        id: 'settings-email',
        titleKey: 'admin.modules.cards.settingsEmail.title',
        descriptionKey: 'admin.modules.cards.settingsEmail.description',
        to: '/admin/settings?tab=email',
        icon: 'mail',
        accent: 'rose'
      }
    ]
  }
]
