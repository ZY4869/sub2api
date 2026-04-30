import dashboard from './dashboard'
import modules from './modules'
import channels from './channels'
import channelMonitors from './channelMonitors'
import affiliates from './affiliates'
import models from './models'
import billing from './billing'
import dataManagement from './dataManagement'
import users from './users'
import groups from './groups'
import subscriptions from './subscriptions'
import apiDocs from './apiDocs'
import accounts from './accounts'
import scheduledTests from './scheduledTests'
import proxies from './proxies'
import redeem from './redeem'
import announcements from './announcements'
import promo from './promo'
import usage from './usage'
import ops from './ops'
import settings from './settings'
import errorPassthrough from './errorPassthrough'
import requestDetails from './requestDetails'
import tlsFingerprintProfiles from './tlsFingerprintProfiles'

export default {
  admin: {
    dashboard,
    modules,
    channels,
    channelMonitors,
    affiliates,
    models,
    billing,
    dataManagement,
    users,
    groups,
    subscriptions,
    apiDocs,
    accounts,
    scheduledTests,
    proxies,
    redeem,
    announcements,
    promo,
    usage,
    ops,
    settings,
    errorPassthrough,
    requestDetails,
    tlsFingerprintProfiles,
  },
}
