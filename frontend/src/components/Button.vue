<script setup lang="ts">
const props = defineProps({
  title: {
    type: String,
    required: true,
  },
})
</script>

<template>
  <button class="bg-gray-600 text-white hover:bg-gray-700 px-4 py-2 rounded-md">{{ props.title }}</button>
</template>