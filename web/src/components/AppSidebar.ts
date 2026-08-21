import { computed, defineComponent, h } from 'vue'
import { RouterLink } from 'vue-router'
import { CalendarClock, Clapperboard, Eye, History, OctagonAlert, ShieldCheck } from 'lucide-vue-next'
import { useCampaignStore } from '../stores/campaigns'

const destinations = [
  { to: '/campaigns', label: '专题编排', icon: Clapperboard },
  { to: '/timeline', label: '发布时间轴', icon: CalendarClock },
  { to: '/preview', label: '前台预览', icon: Eye },
  { to: '/reviews', label: '审核中心', icon: ShieldCheck },
  { to: '/invalidations', label: '失效中心', icon: OctagonAlert },
  { to: '/versions', label: '版本与历史', icon: History }
]

export default defineComponent({
  name: 'PublicationSidebar',
  setup() {
    const campaigns = useCampaignStore()
    const environmentLabel = computed(() => campaigns.environment === 'staging' ? '预发' : '生产')

    return () => h('aside', { class: 'sidebar' }, [
      h('div', { class: 'brand' }, [
        h('span', { class: 'brand-mark' }, '季'),
        h('div', [h('strong', '素材编排台'), h('small', 'Seasonal Desk')])
      ]),
      h('nav', destinations.map(item =>
        h(RouterLink, { key: item.to, to: item.to }, {
          default: () => [h(item.icon, { size: 18 }), item.label]
        })
      )),
      h('div', { class: 'environment' }, [
        h('span', { class: 'pulse' }), `${environmentLabel.value}环境 · 本地`
      ])
    ])
  }
})
