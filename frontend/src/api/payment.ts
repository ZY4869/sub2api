/**
 * User payment API endpoints.
 */

import { apiClient } from './client'
import type {
  PaymentCreateOrderRequest,
  PaymentCreateOrderResponse,
  PaymentOrder,
  PaymentResumeOrderResponse
} from '@/types'

export async function createPaymentOrder(
  payload: PaymentCreateOrderRequest,
  idempotencyKey?: string
): Promise<PaymentCreateOrderResponse> {
  const { data } = await apiClient.post<PaymentCreateOrderResponse>('/payment/orders', payload, {
    headers: idempotencyKey ? { 'Idempotency-Key': idempotencyKey } : undefined
  })
  return data
}

export async function getPaymentOrder(orderNo: string): Promise<PaymentOrder> {
  const { data } = await apiClient.get<PaymentOrder>(`/payment/orders/${encodeURIComponent(orderNo)}`)
  return data
}

export async function resumePaymentOrder(resumeToken: string): Promise<PaymentResumeOrderResponse> {
  const { data } = await apiClient.get<PaymentResumeOrderResponse>(
    `/payment/resume/${encodeURIComponent(resumeToken)}`
  )
  return data
}

export async function resumePaymentOrderByOrderNo(orderNo: string): Promise<PaymentResumeOrderResponse> {
  const { data } = await apiClient.get<PaymentResumeOrderResponse>(
    `/payment/orders/${encodeURIComponent(orderNo)}/resume`
  )
  return data
}

export async function cancelPaymentOrder(orderNo: string): Promise<{ order_no: string }> {
  const { data } = await apiClient.post<{ order_no: string }>(
    `/payment/orders/${encodeURIComponent(orderNo)}/cancel`
  )
  return data
}

export const paymentAPI = {
  createPaymentOrder,
  getPaymentOrder,
  resumePaymentOrder,
  resumePaymentOrderByOrderNo,
  cancelPaymentOrder
}

export default paymentAPI
