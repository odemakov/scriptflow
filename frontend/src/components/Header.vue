<script setup lang="ts">

import router from '../router';
import { useAuthStore } from '../stores/pocketbase';
import Breadcrumbs from './Breadcrumbs.vue';

const auth = useAuthStore()

const handleLogout = async () => {
  await auth.logout()
  router.push({ name: 'home' })
}
</script>

<template>
  <header class="flex items-center justify-between p-4 bg-neutral-content text-primary">
    <div class="flex items-start space-x-4">
      <span class="text-xl font-bold">
        <a href="/" class="text-primary">
          Scriptflow
        </a>
      </span>
      <Breadcrumbs />
    </div>
    <nav>
      <button v-if="auth.isAuthenticated" @click="handleLogout" class="btn btn-ghost btn-xs">Logout</button>
    </nav>
  </header>
</template>