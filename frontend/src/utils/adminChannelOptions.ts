import channelsAPI, { type Channel } from '@/api/admin/channels'
import type { SelectOption } from '@/types'

const PAGE_SIZE = 100

function formatChannelLabel(channel: Channel): string {
  const suffix = channel.status === 'disabled' ? ' (Disabled)' : ''
  return `${channel.name} (#${channel.id})${suffix}`
}

export async function loadAllAdminChannelOptions(): Promise<SelectOption[]> {
  const items: Channel[] = []
  let page = 1
  let pages = 1

  do {
    const response = await channelsAPI.list(page, PAGE_SIZE)
    items.push(...response.items)
    pages = response.pages || 1
    page += 1
  } while (page <= pages)

  return items.map((channel) => ({
    value: channel.id,
    label: formatChannelLabel(channel),
    status: channel.status,
    description: channel.description ?? '',
  }))
}
