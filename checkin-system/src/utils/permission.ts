import { usePermStore } from '@/store/permStore'

export function hasAction(key: string): boolean {
  const store = usePermStore()
  return store.hasAction(key)
}
