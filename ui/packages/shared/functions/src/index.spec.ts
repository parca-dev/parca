import { capitalize } from './index'

describe('Functions', () => {
  it('capitalize', () => {
    expect(capitalize('john doe')).toBe('John Doe')
  })
})
