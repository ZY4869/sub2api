import common from './common'
import archiveAndSync from './archiveAndSync'
import importExport from './importExport'
import formAndFilters from './formAndFilters'
import bulkAndBatch from './bulkAndBatch'
import platformOpenAI from './platformOpenAI'
import platformGateway from './platformGateway'
import platformBaiduOauth from './platformBaiduOauth'
import platformGeminiGrok from './platformGeminiGrok'
import platformKiro from './platformKiro'
import diagnostics from './diagnostics'
import testing from './testing'
import automation from './automation'
export default {
  ...common,
  ...archiveAndSync,
  ...importExport,
  ...formAndFilters,
  ...bulkAndBatch,
  ...platformOpenAI,
  ...platformGateway,
  ...platformBaiduOauth,
  ...platformGeminiGrok,
  ...platformKiro,
  ...diagnostics,
  ...testing,
  ...automation,
}