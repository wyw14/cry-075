<script setup lang="ts">
import AppHeader from './components/AppHeader'
import AppSidebar from './components/AppSidebar'
</script>

<template>
  <div class="shell">
    <AppSidebar />
    <main>
      <AppHeader />
      <RouterView />
    </main>
  </div>
</template>
