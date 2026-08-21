import { computed, defineComponent, h } from 'vue'
import { useRoute } from 'vue-router'
import { useCampaignStore } from '../stores/campaigns'

const environments = [
  { value: 'staging', label: '预发' },
  { value: 'production', label: '生产' }
]

export default defineComponent({
  name: 'PublicationHeader',
  setup() {
    const route = useRoute()
    const campaigns = useCampaignStore()
    const title = computed(() => String(route.meta.title || '编排台'))

    async function selectEnvironment(value: string) {
      if (campaigns.environment === value) return
      campaigns.environment = value
      await campaigns.load()
    }

    return () => h('header', [
      h('div', [h('p', '季节素材运营'), h('h1', title.value)]),
      h('div', { class: 'header-actions' }, [
        h('div', { class: 'environment-switch', 'aria-label': '发布环境' }, environments.map(item =>
          h('button', {
            type: 'button',
            class: { active: campaigns.environment === item.value },
            onClick: () => { void selectEnvironment(item.value) }
          }, item.label)
        )),
        h('div', { class: 'operator' }, [h('span', '运营编辑'), h('strong', '林屿')])
      ])
    ])
  }
})
