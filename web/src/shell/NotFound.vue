<script setup lang="ts">
import { useRouter } from 'vue-router'
const router = useRouter()
</script>

<template>
  <div class="flex min-h-screen flex-col items-center justify-center text-center">
    <div class="text-5xl">🤔</div>
    <h1 class="mt-3 text-lg font-semibold text-gray-800">这页不存在</h1>
    <p class="mt-1 text-xs text-gray-400">可能模块未启用，或路由未声明。</p>
    <van-button class="mt-4" round size="small" @click="router.replace('/')">回总览</van-button>
  </div>
</template>
