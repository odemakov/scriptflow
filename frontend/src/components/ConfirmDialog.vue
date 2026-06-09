<script setup lang="ts">
import { ref } from "vue";

const dialogRef = ref<HTMLDialogElement | null>(null);
const title = ref("");
const message = ref("");
let resolvePromise: ((value: boolean) => void) | null = null;

const open = (t: string, m: string): Promise<boolean> => {
  title.value = t;
  message.value = m;
  dialogRef.value?.showModal();
  return new Promise((resolve) => {
    resolvePromise = resolve;
  });
};

const confirm = () => {
  dialogRef.value?.close();
  resolvePromise?.(true);
};

const cancel = () => {
  dialogRef.value?.close();
  resolvePromise?.(false);
};

defineExpose({ open });
</script>

<template>
  <dialog ref="dialogRef" class="modal">
    <div class="modal-box">
      <h3 class="font-bold text-lg">{{ title }}</h3>
      <p class="py-4">{{ message }}</p>
      <div class="modal-action">
        <button class="btn" @click="cancel">Cancel</button>
        <button class="btn btn-error" @click="confirm">Confirm</button>
      </div>
    </div>
    <form method="dialog" class="modal-backdrop">
      <button @click="cancel">close</button>
    </form>
  </dialog>
</template>
