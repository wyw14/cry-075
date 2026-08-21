<script setup lang="ts">
import type { Status } from '../types'
const props = defineProps<{ status: Status }>()
const labels: Record<Status, string> = { draft: '草稿', pending_review: '待审核', published: '已发布', paused: '已暂停', expired: '已过期' }
</script>
<template><span class="status" :data-status="props.status">{{ labels[props.status] }}</span></template>

