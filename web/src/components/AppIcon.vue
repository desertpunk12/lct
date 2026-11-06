<script setup lang="ts">
const paths = {
  plus: 'M12 5v14M5 12h14',
  close: 'm6 6 12 12M6 18 18 6',
  check: 'm5 12 4 4L19 6',
  chevron: 'm7 10 5 5 5-5',
  refresh: 'M20 7v5h-5M4 17v-5h5M6.1 7a7 7 0 0 1 11.55-1.6L20 8M4 16l2.35 2.6A7 7 0 0 0 17.9 17',
  warning: 'M12 8v5m0 3h.01M10.3 4.8 2.5 18.2A1.2 1.2 0 0 0 3.5 20h17a1.2 1.2 0 0 0 1-1.8L13.7 4.8a2 2 0 0 0-3.4 0Z',
  ledger: 'M6 3h13v18H6a3 3 0 0 1-3-3V6a3 3 0 0 1 3-3ZM7 3v18M11 8h4m-4 4h4m-4 4h2',
  car: 'm5 8 1.5-4h11L19 8M3 16V9l2-1h14l2 1v7H3Zm2 0v3m14-3v3M6 12h2m8 0h2',
} as const

defineProps<{ name: keyof typeof paths }>()
</script>

<template>
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">
    <path :d="paths[name]" />
  </svg>
</template>
