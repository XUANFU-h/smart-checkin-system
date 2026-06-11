import { defineStore } from 'pinia'
import { ref } from 'vue'
import request from '@/api/request.ts'

export const usePermStore = defineStore('perm', () => {
  const menus = ref<string[]>([])
  const actions = ref<string[]>([])
  const trackIds = ref<number[]>([])
  const isSuperAdmin = ref(false)
  const loaded = ref(false)

  async function loadPermissions() {
    if (loaded.value) return
    try {
      const res: any = await request.get('/admin/my-permissions')
      const data = res.data ?? res
      menus.value = data.menus ?? []
      actions.value = data.actions ?? []
      trackIds.value = data.trackIds ?? []
      isSuperAdmin.value = data.isSuperAdmin ?? false
      loaded.value = true
    } catch {
      menus.value = []
      actions.value = []
    }
  }

  function hasMenu(key: string): boolean {
    if (isSuperAdmin.value) return true
    return menus.value.includes(key)
  }

  function hasAction(key: string): boolean {
    if (isSuperAdmin.value) return true
    return actions.value.includes(key)
  }

  function reset() {
    menus.value = []
    actions.value = []
    trackIds.value = []
    isSuperAdmin.value = false
    loaded.value = false
  }

  return { menus, actions, trackIds, isSuperAdmin, loaded, loadPermissions, hasMenu, hasAction, reset }
})
