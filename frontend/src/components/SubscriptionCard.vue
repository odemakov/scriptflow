<script setup lang="ts">
import { computed, ref, watch } from "vue";

import { useSubscriptionStore } from "@/stores/SubscriptionStore";
import { useToastStore } from "@/stores/ToastStore";
import { RunStatusClass } from "@/lib/helpers";
import config from "@/config";

const props = defineProps<{
  task: ITask;
}>();

const useToasts = useToastStore();
const useSubscription = useSubscriptionStore();

const subscriptions = computed(() => useSubscription.getSubscriptions);
const isFolded = ref(
  config.isXS.value || config.isSM.value || config.isMD.value,
);
watch([config.isXS, config.isSM, config.isMD, config.isLG], () => {
  isFolded.value = config.isXS.value || config.isSM.value || config.isMD.value;
});
const toggleFold = () => {
  isFolded.value = !isFolded.value;
};

watch(
  () => props.task.id,
  (newVal) => {
    useSubscription.fetchSubscriptionsForTask(newVal);
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
  <div
    v-if="subscriptions.length > 0"
    class="card card-compact bg-base-100 shadow-xl"
  >
    <div class="card-body">
      <div class="flex justify-between items-center">
        <h2 class="card-title">Task subscriptions</h2>
        <button class="btn btn-xs text-xs" @click="toggleFold">
          <span v-if="isFolded">show</span>
          <span v-else>hide</span>
        </button>
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
