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
import ui from './ui'

const zhAdmin = admin as any
const zhAdminOverrides = adminModelOverrides as any
const zhAdminModels = zhAdmin.admin?.models ?? {}
const zhAdminOverrideModels = zhAdminOverrides.admin?.models ?? {}
const zhAdminPages = zhAdminModels.pages ?? {}
const zhAdminOverridePages = zhAdminOverrideModels.pages ?? {}
const zhAdminAllModelsPage = zhAdminPages.all ?? {}
const zhAdminOverrideAllModelsPage = zhAdminOverridePages.all ?? {}

const mergeLocaleBranch = (base: Record<string, any> = {}, override: Record<string, any> = {}) => ({
  ...base,
  ...override
})

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
          ...zhAdminPages.available,
          ...zhAdminOverridePages.available
        },
        all: mergeLocaleBranch(zhAdminAllModelsPage, {
          ...zhAdminOverrideAllModelsPage,
          viewModes: mergeLocaleBranch(zhAdminAllModelsPage.viewModes, zhAdminOverrideAllModelsPage.viewModes),
          categories: mergeLocaleBranch(zhAdminAllModelsPage.categories, zhAdminOverrideAllModelsPage.categories),
          bulk: mergeLocaleBranch(zhAdminAllModelsPage.bulk, zhAdminOverrideAllModelsPage.bulk)
        }),
        pricing: {
          ...zhAdminPages.pricing,
          ...zhAdminOverridePages.pricing
        },
        billing: {
          ...zhAdminPages.billing,
          ...zhAdminOverridePages.billing
        },
        official: {
          ...zhAdminPages.official
        },
        sale: {
          ...zhAdminPages.sale
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
  ...ui,
}
