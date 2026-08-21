export type Status = 'draft' | 'pending_review' | 'published' | 'paused' | 'expired'
export interface TimeWindow { starts_at: string; ends_at: string }
export interface Campaign { id: string; name: string; season: string; status: Status; package_id: string; placement_id: string; audience_id: string; environment: string; window: TimeWindow; version: number; updated_at: string }
export interface Page<T> { items: T[]; page: number; per_page: number; total: number }
export interface PreviewItem { asset_id: string; name: string; storage_key: string; position: number; fallback: boolean }
export interface PreviewResult { campaign_id?: string; package_id?: string; placement_id: string; items: PreviewItem[]; empty: boolean; reason?: string; generated_at: string }
export interface Impact { campaign_id: string; affected_assets: string[]; conflicting_campaign_ids: string[]; fallback_package_id?: string; audience_estimate: number; warnings: string[] }

