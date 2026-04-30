import overview from './overview'
import diagnosis from './diagnosis'
import errors from './errors'
import requestDetails from './requestDetails'
import alerts from './alerts'
import runtime from './runtime'
import notifications from './notifications'
import settings from './settings'

export default {
  ...overview,
  ...diagnosis,
  ...errors,
  ...requestDetails,
  ...alerts,
  ...runtime,
  ...notifications,
  ...settings,
}
