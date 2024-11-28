<script setup lang="ts">
import { useAuthStore } from '@/stores/AuthStore'
import LoginForm from '@/components/LoginForm.vue'
import SelectProject from '@/components/SelectProject.vue'
import PageTitle from '@/components/PageTitle.vue'

const auth = useAuthStore()
</script>

<template>
  <template v-if="auth.isAuthenticated">
    <PageTitle title="Projects" />
    <SelectProject />
  </template>
  <template v-else>
    <LoginForm />
  </template>
</template>
