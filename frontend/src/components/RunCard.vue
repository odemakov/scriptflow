<script setup lang="ts">
import { useRouter } from 'vue-router';

import Command from './Command.vue';
import Identifier from './Identifier.vue';
import RunStatus from './RunStatus.vue';
const props = defineProps<{
  run: IRun,
}>()
const router = useRouter()

const gotoTask = () => {
  router.push({ name: 'task', params: { id: props.run?.expand?.task?.id } })
}

</script>

<template>
  <div class="card card-compact bg-base-200 w-96 shadow-xl">
    <div class="card-body">
      <h2 class="card-title">{{ props.run?.expand?.task?.name }}</h2>
      <Command v-if="props.run.command" :command="props.run.command" />
      <table class="table table-xs">
        <tbody>
          <tr>
            <td>Task</td>
            <td>
              <Identifier :id="props.run?.expand?.task?.id" @click="gotoTask()" />
            </td>
          </tr>
          <tr>
            <td>Id</td>
            <td>
              <Identifier :id="props.run.id" />
            </td>
          </tr>
          <tr>
            <td>status</td>
            <td>
              <RunStatus :run="props.run" />
            </td>
          </tr>
          <tr>
            <td>Host</td>
            <td>{{ props.run.host }}</td>
          </tr>
          <tr>
            <td>Exit code</td>
            <td>{{ props.run.exit_code }}</td>
          </tr>
          <tr>
            <td>Connection error</td>
            <td>{{ props.run.connection_error }}</td>
          </tr>
          <tr>
            <td>Created</td>
            <td>{{ props.run.created }}</td>
          </tr>
          <tr>
            <td>Updated</td>
            <td>{{ props.run.updated }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>