<script setup lang="ts">
import { useRouter } from 'vue-router'
const router = useRouter()
</script>

<template>
  <div class="not-found">
    <div class="m3-empty-icon big-icon">
      <md-icon>search_off</md-icon>
    </div>
    <h1 class="m3-headline-medium text-on-surface mt-4">这页不存在</h1>
    <p class="m3-body-medium text-on-surface-variant mt-2">
      可能模块未启用，或路由未声明。
    </p>
    <md-filled-button class="mt-6" @click="router.replace('/')">回总览</md-filled-button>
  </div>
</template>

<style scoped>
.not-found {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  background: var(--md-sys-color-background);
  color: var(--md-sys-color-on-background);
  padding: 24px;
}
.big-icon {
  width: 96px;
  height: 96px;
  border-radius: 48px;
}
.big-icon md-icon {
  --md-icon-size: 48px;
  font-size: 48px;
}
</style>
