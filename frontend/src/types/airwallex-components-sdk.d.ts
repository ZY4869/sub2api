declare module '@airwallex/components-sdk' {
  export interface AirwallexInitOptions {
    env?: 'demo' | 'prod' | string
    locale?: string
    clientId?: string
    enabledElements?: Array<'payments' | 'payouts' | 'onboarding' | 'risk' | string>
    [key: string]: unknown
  }

  export interface AirwallexElement {
    mount: (selectorOrElement: string | HTMLElement) => void
    confirm?: (options: Record<string, unknown>) => Promise<unknown>
    unmount?: () => void
    destroy?: () => void
    on?: (event: string, handler: (...args: unknown[]) => void) => void
  }

  export function init(options: AirwallexInitOptions): Promise<unknown> | unknown
  export function createElement(
    type: string,
    options?: Record<string, unknown>
  ): Promise<AirwallexElement | null> | AirwallexElement | null
}
