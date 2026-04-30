import common from './common'
import telegram from './telegram'
import registration from './registration'
import affiliate from './affiliate'
import turnstile from './turnstile'
import linuxdo from './linuxdo'
import defaults from './defaults'
import claudeCode from './claudeCode'
import gateway from './gateway'
import site from './site'
import email from './email'
import opsMonitoring from './opsMonitoring'
import adminApiKey from './adminApiKey'
import googleBatch from './googleBatch'

export default {
  ...common,
  ...telegram,
  ...registration,
  ...affiliate,
  ...turnstile,
  ...linuxdo,
  ...defaults,
  ...claudeCode,
  ...gateway,
  ...site,
  ...email,
  ...opsMonitoring,
  ...adminApiKey,
  ...googleBatch,
}
