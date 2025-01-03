<script setup lang="ts">
import { onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import { ICrumb } from "@/types";

const router = useRouter();
const props = defineProps<{
  crumbs: ICrumb[];
}>();

const handleKeyPress = (event: KeyboardEvent) => {
  if (event.key === "b" || event.key === "B") {
    // follow the one before last crumb
    if (props.crumbs.length > 1) {
      props.crumbs[props.crumbs.length - 2].to();
    } else {
      // go home
      router.push({ name: "home" });
    }
  }
};

onMounted(() => {
  window.addEventListener("keydown", handleKeyPress);
});

onUnmounted(() => {
  window.removeEventListener("keydown", handleKeyPress);
});
</script>
<template>
  <div class="text-center pb-4 flex justify-center items-center space-x-4">
    <div class="breadcrumbs text-md">
      <ul>
        <li>
          <a @click="() => router.push({ name: 'home' })">Home</a>
        </li>
        <li v-for="(crumb, index) in props.crumbs" :key="index">
          <a v-if="crumb.to" @click="crumb.to">{{ crumb.label }}</a>
          <span v-else>{{ crumb.label }}</span>
        </li>
      </ul>
    </div>
  </div>
</template>
