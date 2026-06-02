<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" @click.self="$emit('close')">
    <div class="card w-full max-w-lg max-h-[90vh] overflow-y-auto">
      <slot />
    </div>
  </div>
</template>

<script setup>
defineEmits(['close'])
</script>
