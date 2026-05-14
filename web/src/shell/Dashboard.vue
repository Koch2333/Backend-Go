<script setup lang="ts">
import { MODULES } from './modules'
</script>

<template>
  <div>
    <h1 class="text-lg font-semibold text-gray-800">后台总览</h1>
    <p class="mt-1 text-xs text-gray-500">
      已装载 {{ MODULES.length }} 个模块。在左侧选一个进入。
    </p>

    <div class="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      <router-link
        v-for="m in MODULES"
        :key="m.name"
        :to="(m.nav && m.nav[0]?.to) || `/m/${m.name}/login`"
        class="m3-card block rounded-3xl bg-white p-4 transition hover:shadow-md"
      >
        <div class="text-base font-semibold text-gray-800">{{ m.title }}</div>
        <div class="text-xs text-gray-400">{{ m.name }}　·　{{ m.apiPrefix }}</div>
        <p v-if="m.description" class="mt-2 text-sm text-gray-600">{{ m.description }}</p>
      </router-link>
    </div>

    <div v-if="MODULES.length === 0" class="m3-card mt-8 rounded-3xl border border-dashed border-gray-300 bg-white p-6 text-center text-sm text-gray-400">
      还没有模块。复制 <code>src/modules/_template/</code> 过来是最快的起手方式。
    </div>
  </div>
</template>

<style scoped>
.m3-card {
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
}
</style>
