export interface AccountPoolModeState {
  enabled: boolean
  retryCount: number
}

export interface AccountCustomErrorCodesState {
  enabled: boolean
  selectedCodes: number[]
  input: number | null
}

export const createDefaultAccountPoolModeState = (
  defaultRetryCount: number
): AccountPoolModeState => ({
  enabled: false,
  retryCount: defaultRetryCount
})

export const createDefaultAccountCustomErrorCodesState =
  (): AccountCustomErrorCodesState => ({
    enabled: false,
    selectedCodes: [],
    input: null
  })
