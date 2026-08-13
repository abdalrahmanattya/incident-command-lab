import { PublicClientApplication, type AccountInfo, type AuthenticationResult } from '@azure/msal-browser'
import type { RuntimeConfig } from './vite-env'
import { setAccessToken } from './api'

export function config(): RuntimeConfig {
  return { authMode: 'disabled', clientId: '', tenantId: '', operatorGroupId: '', apiScope: '', ...window.__INCIDENTLAB_CONFIG__ }
}

export class OperatorAuth {
  private readonly runtime = config()
  private readonly msal = this.runtime.authMode === 'entra' && this.runtime.clientId && this.runtime.tenantId
    ? new PublicClientApplication({ auth: { clientId: this.runtime.clientId, authority: `https://login.microsoftonline.com/${this.runtime.tenantId}`, redirectUri: window.location.origin } })
    : null
  account: AccountInfo | null = this.msal?.getAllAccounts()[0] ?? null
  get enabled() { return this.runtime.authMode === 'entra' }
  get allowed() {
    if (!this.enabled) return true
    const groups = (this.account?.idTokenClaims?.groups as string[] | undefined) ?? []
    return Boolean(this.account && this.runtime.operatorGroupId && groups.includes(this.runtime.operatorGroupId))
  }
  async initialise() {
    if (!this.msal) return
    await this.msal.initialize()
    const result = await this.msal.handleRedirectPromise()
    this.account = result?.account ?? this.msal.getAllAccounts()[0] ?? null
    if (result?.accessToken) setAccessToken(result.accessToken)
  }
  async signIn() {
    if (!this.msal) return
    await this.msal.loginRedirect({ scopes: this.runtime.apiScope ? [this.runtime.apiScope] : ['openid', 'profile'] })
  }
  async token() {
    if (!this.msal || !this.account) return
    const result: AuthenticationResult = await this.msal.acquireTokenSilent({ account: this.account, scopes: this.runtime.apiScope ? [this.runtime.apiScope] : ['openid', 'profile'] })
    setAccessToken(result.accessToken)
  }
}
