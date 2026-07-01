export default {
  title: 'Payment Orders',
  description: 'Review built-in payment orders and process refunds',
  filters: {
    status: 'Order status',
    productType: 'Product type',
    userId: 'User ID',
    allStatuses: 'All statuses',
    allProducts: 'All products'
  },
  columns: {
    order: 'Order',
    user: 'User',
    product: 'Product',
    amount: 'Amount',
    refund: 'Refund',
    status: 'Status',
    provider: 'Provider',
    createdAt: 'Created',
    actions: 'Actions'
  },
  product: {
    balance_topup: 'Balance top-up',
    subscription: 'Subscription'
  },
  convertedAmount: 'Converted {amount}',
  refund: {
    action: 'Refund',
    title: 'Create refund',
    amountMinor: 'Refund amount (minor)',
    amountHint: 'Remaining refundable amount: {amount}. Enter the amount in the smallest currency unit.',
    refundable: 'Refundable',
    invalidAmount: 'Refund amount must be greater than 0 and cannot exceed the remaining refundable amount.',
    reason: 'Refund reason',
    reasonPlaceholder: 'Customer requested refund',
    submit: 'Submit refund',
    submitting: 'Submitting',
    success: 'Refund submitted',
    failed: 'Refund failed'
  },
  loadFailed: 'Failed to load payment orders. Refresh and try again.',
  refresh: 'Refresh'
}
