import { defineStore } from 'pinia'
import { request } from '../api'
import type { Campaign, Page } from '../types'

export const useCampaignStore = defineStore('campaigns', {
  state: () => ({ items: [] as Campaign[], loading: false, error: '', environment: 'production', status: '' }),
  getters: { active: state => state.items.filter(item => item.status === 'published') },
  actions: {
    async load() {
      this.loading = true
      this.error = ''
      try {
        const query = new URLSearchParams({ page: '1', per_page: '50', sort: 'updated_at:desc', environment: this.environment })
        if (this.status) query.set('status', this.status)
        this.items = (await request<Page<Campaign>>(`/campaigns?${query}`)).items
      } catch (error) {
        this.error = error instanceof Error ? error.message : '加载失败'
      } finally { this.loading = false }
    }
  }
})

