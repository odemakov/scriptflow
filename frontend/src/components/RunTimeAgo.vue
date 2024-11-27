<script setup lang="ts">
import { TimeAgo } from '@/lib/helpers';

const props = defineProps<{
  datetime: string,
}>()

</script>

<template>
  <span>
    <span class="text-brown">{{ TimeAgo(props.datetime) }}</span> ago
  </span>
</template>