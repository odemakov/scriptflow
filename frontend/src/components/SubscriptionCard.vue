<script setup lang="ts">
import { computed, ref, watch } from "vue";

import { useSubscriptionStore } from "@/stores/SubscriptionStore";
import { useToastStore } from "@/stores/ToastStore";
import { RunStatusClass } from "@/lib/helpers";
import config from "@/config";
import MenuIcon from "./icons/MenuIcon.vue";

const props = defineProps<{
  task: ITask;
}>();

const useToasts = useToastStore();
const useSubscription = useSubscriptionStore();

const subscriptions = computed(() => useSubscription.getSubscriptions);
const loading = ref(true);
const isFolded = ref(config.isXS.value || config.isSM.value || config.isMD.value);
watch([config.isXS, config.isSM, config.isMD, config.isLG], () => {
  isFolded.value = config.isXS.value || config.isSM.value || config.isMD.value;
});
const toggleFold = () => {
  isFolded.value = !isFolded.value;
  closeDropdown();
};

const closeDropdown = () => {
  const elem = document.activeElement;
  if (elem instanceof HTMLElement) {
    elem.blur();
  }
};

watch(
  () => props.task.id,
  async (newVal: string) => {
    loading.value = true;
    try {
      await useSubscription.fetchSubscriptionsForTask(newVal);
    } finally {
      loading.value = false;
    }
  },
);

const toggleSubscriptionActive = async (subscriptionId: string) => {
  const subscription = subscriptions.value.find(
    (s: ISubscription) => s.id === subscriptionId,
  );
  if (subscription) {
    try {
      subscription.active = !subscription.active;
      useSubscription.update(subscription.id, { active: subscription.active });
    } catch (error: unknown) {
      subscription.active = !subscription.active;
      useToasts.addToast((error as Error).message, "error");
    }
  }
};
</script>

<template>
  <div v-if="loading" class="card card-compact bg-base-100 shadow-xl">
    <div class="card-body">
      <div class="flex justify-between items-center mb-2">
        <div class="skeleton h-6 w-48"></div>
        <div class="skeleton h-6 w-12"></div>
      </div>
      <table class="table table-xs">
        <tbody>
          <tr>
            <td><div class="skeleton h-4"></div></td>
          </tr>
          <tr>
            <td><div class="skeleton h-4"></div></td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
  <div
    v-else-if="subscriptions.length > 0"
    class="card card-compact bg-base-100 shadow-xl"
  >
    <div class="card-body">
      <div class="flex justify-between items-center">
        <h2 class="card-title">Task subscriptions</h2>
        <div class="dropdown dropdown-end">
          <div tabindex="0" role="button" class="btn btn-xs">
            <MenuIcon />
          </div>
          <ul
            tabindex="0"
            class="dropdown-content menu bg-base-100 rounded-box z-[1] w-40 p-2 shadow"
          >
            <li>
              <a @click="toggleFold">
                <span v-if="isFolded">Show details</span>
                <span v-else>Hide details</span>
              </a>
            </li>
          </ul>
        </div>
      </div>
      <div :class="{ hidden: isFolded }">
        <table class="table table-xs">
          <tbody>
            <tr v-for="subscription in subscriptions" :key="subscription.id">
              <td>
                <input
                  type="checkbox"
                  class="toggle toggle-sm"
                  :checked="subscription.active"
                  @change="toggleSubscriptionActive(subscription.id)"
                />
              </td>
              <td class="w-1/2">
                {{ subscription.name }}
              </td>
              <td>
                {{ subscription.threshold }}
              </td>
              <td>
                <span
                  v-for="event in subscription.events"
                  :key="event"
                  class="text-sm badge-sm"
                  :class="RunStatusClass(event)"
                >
                  {{ event }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
