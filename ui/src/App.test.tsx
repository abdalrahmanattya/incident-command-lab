import { describe, expect, it } from 'vitest'
import { config } from './auth'

describe('operator runtime configuration', () => {
  it('defaults to a credential-free local mode', () => {
    delete window.__INCIDENTLAB_CONFIG__
    expect(config().authMode).toBe('disabled')
  })
  it('accepts an Entra cloud configuration', () => {
    window.__INCIDENTLAB_CONFIG__ = { authMode: 'entra', clientId: 'client', tenantId: 'tenant', operatorGroupId: 'group', apiScope: 'api://client/access_as_user' }
    expect(config().operatorGroupId).toBe('group')
  })
})
