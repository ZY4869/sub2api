export default {
  title: 'Image batch generation',
  hint: 'Allows this Gemini group to serve user-submitted image batch jobs when the global switch is enabled.',
  allowedProviders: 'Allowed providers',
  allowedProvidersHint: 'Comma separated. Leave empty to allow the default provider policy.',
  allowedModels: 'Allowed models',
  allowedModelsPlaceholder: 'gemini-2.5-flash-image',
  allowedModelsHint: 'One display model ID per line. Leave empty to use the group model policy only.',
  maxItems: 'Max items',
  maxDownloadMB: 'Max ZIP MB',
  downloadConcurrency: 'Download concurrency',
}
