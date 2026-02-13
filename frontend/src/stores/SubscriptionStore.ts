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
    subscriptions.value = await pb
      .collection(CCollectionName.subscriptions)
      .getFullList<ISubscription>({
        requestKey: taskId,
        expand: "channel",
        sort: "-active,-created",
        filter: pb.filter("task={:taskId}", { taskId: taskId }),
      });
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
