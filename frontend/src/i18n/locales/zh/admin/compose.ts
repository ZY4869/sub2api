type AnyRecord = Record<string, any>

const mergeLocaleBranch = (base: AnyRecord = {}, override: AnyRecord = {}) => ({
  ...base,
  ...override,
})

export default function composeZhAdmin(baseLocale: AnyRecord, overrides: AnyRecord): { admin: AnyRecord } {
  const baseAdmin = baseLocale?.admin ?? {}
  const overrideAdmin = overrides?.admin ?? {}

  const baseModels = baseAdmin.models ?? {}
  const overrideModels = overrideAdmin.models ?? {}

  const basePages = baseModels.pages ?? {}
  const overridePages = overrideModels.pages ?? {}

  const baseAllModelsPage = basePages.all ?? {}
  const overrideAllModelsPage = overridePages.all ?? {}

  return {
    admin: {
      ...baseAdmin,
      ...overrideAdmin,
      models: {
        ...baseModels,
        ...overrideModels,
        pages: {
          ...mergeLocaleBranch(basePages, overridePages),
          available: {
            ...basePages.available,
            ...overridePages.available,
          },
          all: mergeLocaleBranch(baseAllModelsPage, {
            ...overrideAllModelsPage,
            viewModes: mergeLocaleBranch(baseAllModelsPage.viewModes, overrideAllModelsPage.viewModes),
            categories: mergeLocaleBranch(baseAllModelsPage.categories, overrideAllModelsPage.categories),
            bulk: mergeLocaleBranch(baseAllModelsPage.bulk, overrideAllModelsPage.bulk),
          }),
          pricing: {
            ...basePages.pricing,
            ...overridePages.pricing,
          },
          billing: {
            ...basePages.billing,
            ...overridePages.billing,
          },
          official: {
            ...basePages.official,
          },
          sale: {
            ...basePages.sale,
          },
        },
        registry: {
          ...baseModels.registry,
          ...overrideModels.registry,
          fields: {
            ...baseModels.registry?.fields,
            ...overrideModels.registry?.fields,
          },
          actions: {
            ...baseModels.registry?.actions,
            ...overrideModels.registry?.actions,
          },
        },
        available: {
          ...baseModels.available,
          ...overrideModels.available,
          activateDialog: {
            ...baseModels.available?.activateDialog,
            ...overrideModels.available?.activateDialog,
          },
        },
      },
    },
  }
}

