<script setup lang="ts">
const props = defineProps<{
    runs: IRun[],
}>()
console.log(props.runs)
</script>