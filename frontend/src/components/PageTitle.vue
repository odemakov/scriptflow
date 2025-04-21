<script setup lang="ts">
import TaskIcon from "@/components/icons/TaskIcon.vue";
import NodeIcon from "@/components/icons/NodeIcon.vue";
import ProjectIcon from "@/components/icons/ProjectIcon.vue";
import { CEntityType, EntityType } from "@/types";

const props = defineProps<{
  title: string;
  icon?: EntityType;
}>();

const iconMapping = {
  [CEntityType.task]: TaskIcon,
  [CEntityType.node]: NodeIcon,
  [CEntityType.project]: ProjectIcon,
};
</script>

<template>
  <div class="relative pb-1 md:pb-2">
    <div v-if="props.icon" class="absolute left-0 top-1/2 transform -translate-y-1/2">
      <component :is="iconMapping[props.icon]" />
    </div>
    <div class="text-center">
      <div class="text-xl md:text-2xl text-primary">
        {{ props.title }}
      </div>
    </div>
  </div>
</template>
