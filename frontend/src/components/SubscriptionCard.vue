<script setup lang="ts">
import { computed, watch } from 'vue';

import { useSubscriptionStore } from '@/stores/SubscriptionStore';
import { useToastStore } from '@/stores/ToastStore';

const props = defineProps<{
  task: ITask,
}>()

const useToasts = useToastStore()
const useSubscription = useSubscriptionStore()

const subscriptions = computed(() => useSubscription.getSubscriptions)

watch(() => props.task.id, (newVal) => {
  useSubscription.fetchSubscriptionsForTask(newVal)
})

const toggleTaskActive = async (subscriptionId: string) => {
  const subscription = subscriptions.value.find((s: ISubscription) => s.id === subscriptionId)
  if (subscription) {
    try {
      subscription.active = !subscription.active
      useSubscription.update(subscription.id, { active: subscription.active })
    } catch (error: unknown) {
      subscription.active = !subscription.active
      useToasts.addToast(
        (error as Error).message,
        'error',
      )
    }
  }
}
</script>

<template>
  <div class="card card-compact bg-base-100 shadow-xl">
    <div class="card-body">
      <h2 class="card-title">Task subscriptions</h2>
      <table class="table table-xs">
        <tbody>
          <tr v-for="subscription in subscriptions" :key="subscription.id">
            <td>
              <input type="checkbox" class="toggle toggle-sm" :checked="subscription.active"
                @change="toggleTaskActive(subscription.id)" />
            </td>
            <td class="w-1/2">
              {{ subscription.name }}
            </td>
            <td>
              <li v-for="event in subscription.events" :key="event">
                {{ event }}
              </li>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
