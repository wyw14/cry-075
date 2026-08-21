import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useCampaignStore } from './campaigns'

describe('campaign store', () => {
  beforeEach(() => { setActivePinia(createPinia()); vi.restoreAllMocks() })
  it('loads campaigns with the selected environment', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ items: [], page: 1, per_page: 50, total: 0 }), { status: 200 }))
    const store = useCampaignStore()
    await store.load()
    expect(fetchMock.mock.calls[0][0]).toContain('environment=production')
    expect(store.loading).toBe(false)
  })
})

