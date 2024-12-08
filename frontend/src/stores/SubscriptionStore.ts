import { ref, computed } from "vue";
import { defineStore } from "pinia";

import { getPocketBaseInstance } from "./AuthStore";

export const useSubscriptionStore = defineStore("subscriptions", () => {
  const pb = getPocketBaseInstance();
  const subscriptions = ref([] as ISubscription[]);

  // getters
  const getSubscriptions = computed(() => subscriptions.value);

  // methods
  async function fetchSubscriptionsForTask(taskId: string) {
    const records = await pb
      .collection(CCollectionName.subscriptions)
      .getList<ISubscription>(1, 100, {
        expand: "channel",
        sort: "-active,-created",
        filter: pb.filter("task={:taskId}", { taskId: taskId }),
      });
    subscriptions.value = records.items;
  }

  async function update(subscriptionId: string, updatedData: Object) {
    await pb
      .collection(CCollectionName.subscriptions)
      .update(subscriptionId, updatedData);
  }

  return {
    getSubscriptions,
    fetchSubscriptionsForTask,
    update,
  };
});
