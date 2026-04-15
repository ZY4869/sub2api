/**
 * Admin API barrel export
 * Centralized exports for all admin API modules
 */

import dashboardAPI from './dashboard'
import usersAPI from './users'
import groupsAPI from './groups'
import channelsAPI from './channels'
import accountsAPI from './accounts'
import proxiesAPI from './proxies'
import redeemAPI from './redeem'
import promoAPI from './promo'
import announcementsAPI from './announcements'
import settingsAPI from './settings'
import systemAPI from './system'
import subscriptionsAPI from './subscriptions'
import usageAPI from './usage'
import geminiAPI from './gemini'
import antigravityAPI from './antigravity'
import userAttributesAPI from './userAttributes'
import opsAPI from './ops'
import errorPassthroughAPI from './errorPassthrough'
import dataManagementAPI from './dataManagement'
import backupAPI from './backup'
import apiKeysAPI from './apiKeys'
import scheduledTestsAPI from './scheduledTests'
import modelsAPI from './models'
import modelRegistryAPI from './modelRegistry'
import tlsFingerprintProfilesAPI from './tlsFingerprintProfile'

/**
 * Unified admin API object for convenient access
 */
export const adminAPI = {
  dashboard: dashboardAPI,
  users: usersAPI,
  groups: groupsAPI,
  channels: channelsAPI,
  accounts: accountsAPI,
  proxies: proxiesAPI,
  redeem: redeemAPI,
  promo: promoAPI,
  announcements: announcementsAPI,
  settings: settingsAPI,
  system: systemAPI,
  subscriptions: subscriptionsAPI,
  usage: usageAPI,
  gemini: geminiAPI,
  antigravity: antigravityAPI,
  userAttributes: userAttributesAPI,
  ops: opsAPI,
  errorPassthrough: errorPassthroughAPI,
  dataManagement: dataManagementAPI,
  backup: backupAPI,
  apiKeys: apiKeysAPI,
  models: modelsAPI,
  modelRegistry: modelRegistryAPI,
  scheduledTests: scheduledTestsAPI,
  tlsFingerprintProfiles: tlsFingerprintProfilesAPI
}

export {
  dashboardAPI,
  usersAPI,
  groupsAPI,
  channelsAPI,
  accountsAPI,
  proxiesAPI,
  redeemAPI,
  promoAPI,
  announcementsAPI,
  settingsAPI,
  systemAPI,
  subscriptionsAPI,
  usageAPI,
  geminiAPI,
  antigravityAPI,
  userAttributesAPI,
  opsAPI,
  errorPassthroughAPI,
  dataManagementAPI,
  backupAPI,
  apiKeysAPI,
  modelsAPI,
  modelRegistryAPI,
  scheduledTestsAPI,
  tlsFingerprintProfilesAPI
}

export default adminAPI

// Re-export types used by components
export type { BalanceHistoryItem } from './users'
export type { ErrorPassthroughRule, CreateRuleRequest, UpdateRuleRequest } from './errorPassthrough'
export type { BackupAgentHealth, DataManagementConfig } from './dataManagement'
export type { ModelCatalogItem, ModelCatalogDetail, ModelCatalogPricing } from './models'
export type {
  TLSFingerprintProfile,
  CreateTLSFingerprintProfileRequest,
  UpdateTLSFingerprintProfileRequest
} from './tlsFingerprintProfile'
