export const ADMIN_STEP_UP_TOTP_HEADER = 'X-Sub2API-Step-Up-TOTP'

export interface AdminStepUpOptions {
  stepUpTotp?: string
}

export function stepUpHeaders(options?: AdminStepUpOptions) {
  const code = options?.stepUpTotp?.trim()
  return code ? { [ADMIN_STEP_UP_TOTP_HEADER]: code } : undefined
}
