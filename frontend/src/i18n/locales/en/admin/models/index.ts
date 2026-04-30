import common from './common'
import pages from './pages'
import registry from './registry'
import available from './available'
import catalog from './catalog'
import pricing from './pricing'

export default {
  ...common,
  pages,
  registry,
  available,
  ...catalog,
  ...pricing,
}
