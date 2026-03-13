import home from './home'
import keyUsage from './keyUsage'
import setup from './setup'
import common from './common'
import nav from './nav'
import auth from './auth'
import dashboard from './dashboard'
import groups from './groups'
import keys from './keys'
import usage from './usage'
import redeem from './redeem'
import profile from './profile'
import empty from './empty'
import table from './table'
import pagination from './pagination'
import errors from './errors'
import dates from './dates'
import admin from './admin'
import adminModelOverrides from './adminModelOverrides'
import subscriptionProgress from './subscriptionProgress'
import version from './version'
import purchase from './purchase'
import customPage from './customPage'
import announcements from './announcements'
import userSubscriptions from './userSubscriptions'
import onboarding from './onboarding'
import sora from './sora'

const zhAdmin = admin as any
const zhAdminOverrides = adminModelOverrides as any
const zhAdminModels = zhAdmin.admin?.models ?? {}
const zhAdminOverrideModels = zhAdminOverrides.admin?.models ?? {}

export default {
  ...home,
  ...keyUsage,
  ...setup,
  ...common,
  ...nav,
  ...auth,
  ...dashboard,
  ...groups,
  ...keys,
  ...usage,
  ...redeem,
  ...profile,
  ...empty,
  ...table,
  ...pagination,
  ...errors,
  ...dates,
  ...admin,
  admin: {
    ...zhAdmin.admin,
    ...zhAdminOverrides.admin,
    models: {
      ...zhAdminModels,
      ...zhAdminOverrideModels,
      pages: {
        available: {
          ...zhAdminModels.pages?.available,
          ...zhAdminOverrideModels.pages?.available
        },
        all: {
          ...zhAdminModels.pages?.all,
          ...zhAdminOverrideModels.pages?.all
        },
        pricing: {
          ...zhAdminModels.pages?.pricing,
          ...zhAdminOverrideModels.pages?.pricing
        },
        official: {
          ...zhAdminModels.pages?.official
        },
        sale: {
          ...zhAdminModels.pages?.sale
        }
      },
      registry: {
        ...zhAdminModels.registry,
        ...zhAdminOverrideModels.registry,
        fields: {
          ...zhAdminModels.registry?.fields,
          ...zhAdminOverrideModels.registry?.fields
        },
        actions: {
          ...zhAdminModels.registry?.actions,
          ...zhAdminOverrideModels.registry?.actions
        }
      },
      available: {
        ...zhAdminModels.available,
        ...zhAdminOverrideModels.available,
        activateDialog: {
          ...zhAdminModels.available?.activateDialog,
          ...zhAdminOverrideModels.available?.activateDialog
        }
      }
    }
  },
  ...subscriptionProgress,
  ...version,
  ...purchase,
  ...customPage,
  ...announcements,
  ...userSubscriptions,
  ...onboarding,
  ...sora,
}
