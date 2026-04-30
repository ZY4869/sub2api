import common from './platformGemini/common'
import tierAndAccountType from './platformGemini/tierAndAccountType'
import oauthType from './platformGemini/oauthType'
import setupGuide from './platformGemini/setupGuide'
import quotaPolicy from './platformGemini/quotaPolicy'
import vertex from './platformGemini/vertex'

export default {
  ...common,
  ...tierAndAccountType,
  ...oauthType,
  ...setupGuide,
  ...quotaPolicy,
  ...vertex,
}
