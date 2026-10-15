import { describe, it, expect } from 'vitest'
import { toUiError } from './errors'

describe('toUiError', () => {
  const cases: Array<[string, string]> = [
    ['folder.relative', 'errors.folder.relative'],
    ['folder.root', 'errors.folder.root'],
    ['folder.missing', 'errors.folder.missing'],
    ['folder.not_dir', 'errors.folder.notDir'],
    ['folder.not_writable', 'errors.folder.notWritable'],
    ['queue.no_output_dir', 'errors.queue.noOutputDir'],
    ['queue.not_ready', 'errors.queue.notReady'],
  ]

  it.each(cases)('maps %s', (code, key) => {
    expect(toUiError(new Error(`[${code}] /some/path`))).toEqual({ key, vars: { detail: '/some/path' } })
  })

  it('maps a code without detail', () => {
    expect(toUiError(new Error('[queue.not_ready]'))).toEqual({
      key: 'errors.queue.notReady',
      vars: { detail: '' },
    })
  })

  it('maps network failures', () => {
    expect(toUiError(new TypeError('Failed to fetch'))).toEqual({ key: 'errors.unreachable' })
    expect(toUiError(new Error('NetworkError when attempting to fetch resource'))).toEqual({ key: 'errors.unreachable' })
  })

  it('maps a 401', () => {
    expect(toUiError(new Error('401 Unauthorized'))).toEqual({ key: 'errors.unauthorized' })
  })

  it('keeps unknown messages as detail of a generic error', () => {
    expect(toUiError(new Error('boom'))).toEqual({ key: 'errors.generic', vars: { detail: 'boom' } })
  })

  it('falls back on an unknown code', () => {
    expect(toUiError(new Error('[nope.unknown] x'))).toEqual({
      key: 'errors.generic',
      vars: { detail: '[nope.unknown] x' },
    })
  })

  it('handles an empty message and non-Error values', () => {
    expect(toUiError(new Error(''))).toEqual({ key: 'errors.generic', vars: { detail: '' } })
    expect(toUiError('plain string')).toEqual({ key: 'errors.generic', vars: { detail: 'plain string' } })
    expect(toUiError(undefined)).toEqual({ key: 'errors.generic', vars: { detail: '' } })
  })
})
