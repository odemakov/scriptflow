<template>
    <div class="fixed bottom-4 right-4 space-y-2">
        <template v-for="toast in toasts" :key="toast.fired">
            <Toast :toast="toast" />
        </template>
    </div>
</template>

<script setup lang="ts">
import { useToastStore } from '../stores/ToastStore';
import { computed } from 'vue';
import Toast from './Toast.vue';
const useToasts = useToastStore()
const toasts = computed(() => useToasts.getToasts);
</script>