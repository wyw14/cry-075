import { createRouter, createWebHistory } from 'vue-router'
import CampaignComposer from './views/CampaignComposer.vue'
import TimelineView from './views/TimelineView.vue'
import FrontPreview from './views/FrontPreview.vue'
import ReviewQueue from './views/ReviewQueue.vue'
import InvalidationCenter from './views/InvalidationCenter.vue'
import VersionHistory from './views/VersionHistory.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/campaigns' },
    { path: '/campaigns', component: CampaignComposer, meta: { title: '专题编排' } },
    { path: '/timeline', component: TimelineView, meta: { title: '发布时间轴' } },
    { path: '/preview', component: FrontPreview, meta: { title: '前台预览' } },
    { path: '/reviews', component: ReviewQueue, meta: { title: '审核中心' } },
    { path: '/invalidations', component: InvalidationCenter, meta: { title: '失效中心' } },
    { path: '/versions', component: VersionHistory, meta: { title: '版本与历史' } }
  ]
})

