import type {
  APIKeyModelBindingMode,
  EffectiveExternalModelCatalogViewMode,
  ExternalModelCatalogViewMode,
  User,
} from '@/types'

export function normalizeExternalModelCatalogViewMode(
  value?: string | null,
): ExternalModelCatalogViewMode {
  switch (value) {
    case 'group_first':
    case 'model_only':
    case 'follow_key_binding':
      return value
    default:
      return 'follow_key_binding'
  }
}

export function resolveEffectiveExternalModelCatalogViewMode(
  user?: Pick<User, 'external_model_catalog_view_mode' | 'effective_external_model_catalog_view_mode' | 'api_key_model_binding_mode'> | null,
): EffectiveExternalModelCatalogViewMode {
  if (user?.effective_external_model_catalog_view_mode === 'group_first') {
    return 'group_first'
  }
  if (user?.effective_external_model_catalog_view_mode === 'model_only') {
    return 'model_only'
  }

  const mode = normalizeExternalModelCatalogViewMode(user?.external_model_catalog_view_mode)
  if (mode === 'group_first' || mode === 'model_only') {
    return mode
  }
  return normalizeAPIKeyModelBindingMode(user?.api_key_model_binding_mode) === 'group_allowed'
    ? 'group_first'
    : 'model_only'
}

function normalizeAPIKeyModelBindingMode(value?: string | null): APIKeyModelBindingMode {
  return value === 'group_allowed' ? 'group_allowed' : 'model_required'
}
