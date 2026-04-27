import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/views/admin/UsersView.vue'),
  'utf8',
)

describe('admin UsersView request details access', () => {
  it('does not expose reviewer grant or revoke actions', () => {
    expect(source).not.toContain('handleToggleRequestDetailsReview')
    expect(source).not.toContain('grantRequestDetailsReview')
    expect(source).not.toContain('revokeRequestDetailsReview')
    expect(source).not.toContain('requestDetailsReviewBadge')
  })
})
