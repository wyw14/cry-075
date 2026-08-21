<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { GripVertical, Plus, Search, SlidersHorizontal } from 'lucide-vue-next'
import { useCampaignStore } from '../stores/campaigns'
import StatusBadge from '../components/StatusBadge.vue'
const store = useCampaignStore()
const selected = ref('')
onMounted(() => store.load())
</script>
<template>
  <section class="workspace">
    <div class="toolbar"><label class="search"><Search :size="17"/><input placeholder="搜索专题或季节" /></label><select v-model="store.status" @change="store.load"><option value="">全部状态</option><option value="draft">草稿</option><option value="pending_review">待审核</option><option value="published">已发布</option></select><button class="icon-button" title="筛选"><SlidersHorizontal :size="18"/></button><button class="primary"><Plus :size="18"/>新建专题</button></div>
    <div v-if="store.error" class="notice error">{{ store.error }}</div>
    <div class="split">
      <div class="table-panel"><div class="table-head"><span>专题</span><span>展示时间</span><span>状态</span><span>版本</span></div><button v-for="campaign in store.items" :key="campaign.id" class="table-row" :class="{selected:selected===campaign.id}" @click="selected=campaign.id"><span><strong>{{ campaign.name }}</strong><small>{{ campaign.season }} · {{ campaign.environment }}</small></span><span>{{ new Date(campaign.window.starts_at).toLocaleDateString() }} - {{ new Date(campaign.window.ends_at).toLocaleDateString() }}</span><StatusBadge :status="campaign.status"/><span>v{{ campaign.version }}</span></button></div>
      <aside class="inspector"><div class="panel-title"><div><small>编排画布</small><h2>{{ selected ? '专题素材包' : '选择一个专题' }}</h2></div><button class="icon-button" title="添加素材"><Plus :size="18"/></button></div><div class="asset-list"><div v-for="index in 4" :key="index" class="asset-row"><GripVertical :size="17"/><div class="thumb" :data-tone="index"/><span><strong>{{ ['主视觉海报','限时活动横幅','新品短片','会员权益图'][index-1] }}</strong><small>推荐位顺序 {{ index }}</small></span><span v-if="index===1" class="pin">置顶</span></div></div><div class="inspector-footer"><span>4 个素材 · 1 个替补</span><button>保存草稿</button></div></aside>
    </div>
  </section>
</template>

