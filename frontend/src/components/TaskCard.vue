<script setup lang="ts">
import { computed, watch, ref } from "vue";
import { useRouter } from "vue-router";

import Command from "./Command.vue";
import Identifier from "./Identifier.vue";
import IdentifierUrl from "./IdentifierUrl.vue";
import TrueFalse from "./TrueFalse.vue";
import MenuIcon from "./icons/MenuIcon.vue";
import config from "@/config";
import { useRunStore } from "@/stores/RunStore";
import { useTaskStore } from "@/stores/TaskStore";
import { useToastStore } from "@/stores/ToastStore";

const props = defineProps<{
  task: ITask;
}>();

const router = useRouter();
const useToasts = useToastStore();
const useRuns = useRunStore();
const useTask = useTaskStore();

const lastRuns = computed(() => useRuns.getLastRuns[props.task.id]);
const loading = ref(true);
const lastRunStarted = computed(() => {
  if (lastRuns.value && lastRuns.value.length > 0) {
    return lastRuns.value[0].status === CRunStatus.started;
  } else {
    return false;
  }
});
// this variable is used to disable the run button when a run is in progress or the task is inactive
// we can't fully rely on the last run status because it's updated with small delay
const runTaskButtonDisabled = ref(false);

// fold it on medium screens and below
const isFolded = ref(config.isXS.value || config.isSM.value || config.isMD.value);
watch([config.isXS, config.isSM, config.isMD, config.isLG], () => {
  isFolded.value = config.isXS.value || config.isSM.value || config.isMD.value;
});
const closeDropdown = () => {
  const elem = document.activeElement;
  if (elem instanceof HTMLElement) {
    elem.blur();
  }
};

const toggleFold = () => {
  isFolded.value = !isFolded.value;
  closeDropdown();
};

watch(
  () => props.task,
  async () => {
    loading.value = true;
    runTaskButtonDisabled.value = lastRunStarted.value || !props.task.active;
    try {
      await useRuns.fetchLastRuns(props.task.id);
      useRuns.subscribe();
    } catch (error: unknown) {
      // useToasts.addToast(
      //   (error as Error).message,
      //   'error',
      // )
    } finally {
      loading.value = false;
    }
  },
);

const gotoEntity = (entityType: string, entityId?: string) => {
  if (!entityId) return;

  router.push({
    name: entityType,
    params: { [`${entityType}Id`]: entityId },
  });
};

const toggleTaskActive = async () => {
  closeDropdown();
  try {
    await useTask.toggleTaskActive(props.task.id);
  } catch (error: unknown) {
    useToasts.addToast((error as Error).message, "error");
  }
};

const runTask = async () => {
  closeDropdown();
  runTaskButtonDisabled.value = true;

  const oldSchedule = props.task.schedule;
  try {
    await useTask.updateTask(props.task.id, {
      ...props.task,
      schedule: "@every 1s",
    });
    runTaskButtonDisabled.value = false;
  } catch (error: unknown) {
    useToasts.addToast((error as Error).message, "error");
  }

  // wait 2 seconds and set oldSchedule back
  setTimeout(async () => {
    try {
      await useTask.updateTask(props.task.id, {
        ...props.task,
        schedule: oldSchedule,
      });
    } catch (error: unknown) {
      useToasts.addToast((error as Error).message, "error");
    }
  }, 2000);
};
</script>

<template>
  <div class="card card-compact bg-base-100 shadow-xl max-m-[600px] lg:max-w-[400px]">
    <div v-if="loading" class="card-body">
      <div class="skeleton h-8 w-3/4 mb-4"></div>
      <div class="skeleton h-12 w-full mb-2"></div>
      <table class="table table-xs">
        <tbody>
          <tr>
            <td>Id</td>
            <td><div class="skeleton h-4 w-20"></div></td>
          </tr>
          <tr>
            <td>Project</td>
            <td><div class="skeleton h-4 w-32"></div></td>
          </tr>
          <tr>
            <td>Node</td>
            <td><div class="skeleton h-4 w-24"></div></td>
          </tr>
          <tr>
            <td>Schedule</td>
            <td><div class="skeleton h-4 w-28"></div></td>
          </tr>
          <tr>
            <td>Active</td>
            <td><div class="skeleton h-4 w-16"></div></td>
          </tr>
          <tr>
            <td>Prepend datetime</td>
            <td><div class="skeleton h-4 w-16"></div></td>
          </tr>
          <tr>
            <td>Created</td>
            <td><div class="skeleton h-4 w-36"></div></td>
          </tr>
          <tr>
            <td>Updated</td>
            <td><div class="skeleton h-4 w-36"></div></td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else class="card-body">
      <div class="flex justify-between items-center">
        <h2 class="card-title">{{ props.task.name }}</h2>
        <div class="dropdown dropdown-end">
          <div tabindex="0" role="button" class="btn btn-xs">
            <MenuIcon />
          </div>
          <ul
            tabindex="0"
            class="dropdown-content menu bg-base-100 rounded-box z-[1] w-40 p-2 shadow"
          >
            <li>
              <a @click="toggleTaskActive">
                <span v-if="props.task.active">Turn off</span>
                <span v-else>Turn on</span>
              </a>
            </li>
            <li>
              <a
                v-if="!(runTaskButtonDisabled || lastRunStarted || !props.task.active)"
                @click="runTask()"
              >
                Run once
              </a>
              <span
                v-else
                class="text-base-300 px-3 py-2 block disabled-menu-item"
              >
                Run once
              </span>
            </li>
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
        <Command v-if="props.task.command" :command="props.task.command" />
        <table class="table table-xs">
          <tbody>
            <tr>
              <td>Id</td>
              <td>
                <Identifier :id="props.task.id" />
              </td>
            </tr>
            <tr>
              <td>Project</td>
              <td>
                <IdentifierUrl
                  :id="props.task.expand?.project?.name"
                  @click="gotoEntity('project', props.task?.expand?.project?.id)"
                />
              </td>
            </tr>
            <tr>
              <td>Node</td>
              <td>
                <IdentifierUrl
                  :id="props.task.expand?.node?.id"
                  @click="gotoEntity('node', props.task?.expand?.node?.id)"
                />
              </td>
            </tr>
            <tr>
              <td>Schedule</td>
              <td>{{ props.task.schedule }}</td>
            </tr>
            <tr>
              <td>Active</td>
              <td>
                <TrueFalse
                  v-if="props.task.active !== undefined"
                  :status="props.task.active"
                />
              </td>
            </tr>

            <tr>
              <td>Prepend datetime</td>
              <td>
                <TrueFalse
                  v-if="props.task.prepend_datetime !== undefined"
                  :status="props.task.prepend_datetime"
                />
              </td>
            </tr>
            <tr>
              <td>Created</td>
              <td>{{ props.task.created }}</td>
            </tr>
            <tr>
              <td>Updated</td>
              <td>{{ props.task.updated }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.disabled-menu-item {
  cursor: not-allowed !important;
}
</style>
