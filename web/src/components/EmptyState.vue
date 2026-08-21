<script setup lang="ts">import { Inbox } from 'lucide-vue-next';defineProps<{ title: string; detail: string }>()</script>
<template><div class="empty"><Inbox :size="30"/><strong>{{ title }}</strong><span>{{ detail }}</span></div></template>

