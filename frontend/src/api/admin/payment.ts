/**
 * Admin payment API endpoints.
 */

import { apiClient } from '../client'
import type { PaginatedResponse, PaymentOrder, PaymentRefund } from '@/types'

export interface AdminPaymentOrderFilters {
  status?: string
  provider?: string
  product_type?: string
  user_id?: number
}

export async function listOrders(
  page = 1,
  pageSize = 20,
  filters: AdminPaymentOrderFilters = {}
): Promise<PaginatedResponse<PaymentOrder>> {
  const { data } = await apiClient.get<PaginatedResponse<PaymentOrder>>('/admin/payment/orders', {
    params: {
      page,
      page_size: pageSize,
      ...filters
    }
  })
  return data
}

export async function refundOrder(
  orderNo: string,
  payload: { amount_minor?: number; reason?: string },
  idempotencyKey?: string
): Promise<PaymentRefund> {
  const { data } = await apiClient.post<PaymentRefund>(
    `/admin/payment/orders/${encodeURIComponent(orderNo)}/refund`,
    payload,
    {
      headers: idempotencyKey ? { 'Idempotency-Key': idempotencyKey } : undefined
    }
  )
  return data
}

export const adminPaymentAPI = {
  listOrders,
  refundOrder
}

export default adminPaymentAPI
