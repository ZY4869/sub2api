import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

const revisionState = ref(0)

export function invalidateModelInventory(): number {
  revisionState.value = Date.now()
  return revisionState.value
}

export const useModelInventoryStore = defineStore('modelInventory', () => {
  const revision = computed(() => revisionState.value)

  function invalidate() {
    return invalidateModelInventory()
  }

  return {
    revision,
    invalidate
  }
})
