<script setup lang="ts">
import { useRouter } from 'vue-router';

import Command from './Command.vue';
import Identifier from './Identifier.vue';
import IdentifierUrl from './IdentifierUrl.vue';

const props = defineProps<{
  task: ITask,
}>()
const router = useRouter()

const gotoProject = () => {
  router.push({ name: 'project', params: { projectSlug: props.task?.expand?.project?.slug } })
}

</script>

<template>
  <div class="card card-compact bg-base-200 w-96 shadow-xl">
    <div class="card-body">
      <h2 class="card-title">{{ props.task.name }}</h2>
      <Command v-if="props.task.command" :command="props.task.command" />
      <table class="table table-xs">
        <tbody>
          <tr>
            <td>Project</td>
            <td>
              <IdentifierUrl :id="props.task.expand?.project?.name" @click="gotoProject()" />
            </td>
          </tr>
          <tr>
            <td>Id</td>
            <td>
              <Identifier :id="props.task.id" />
            </td>
          </tr>
          <tr>
            <td>Node</td>
            <td>{{ props.task.expand?.node?.host }}</td>
          </tr>
          <tr>
            <td>Schedule</td>
            <td>{{ props.task.schedule }}</td>
          </tr>
          <tr>
            <td>Active</td>
            <td>{{ props.task.active }}</td>
          </tr>
          <tr>
            <td>Prepend datetime</td>
            <td>{{ props.task.prepend_datetime }}</td>
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
</template>