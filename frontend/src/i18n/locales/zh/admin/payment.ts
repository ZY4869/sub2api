export default {
  title: '支付订单',
  description: '查看站内支付订单并处理退款',
  filters: {
    status: '订单状态',
    productType: '商品类型',
    userId: '用户 ID',
    allStatuses: '全部状态',
    allProducts: '全部商品'
  },
  columns: {
    order: '订单',
    user: '用户',
    product: '商品',
    amount: '金额',
    refund: '退款',
    status: '状态',
    provider: '通道',
    createdAt: '创建时间',
    actions: '操作'
  },
  product: {
    balance_topup: '余额充值',
    subscription: '订阅购买'
  },
  convertedAmount: '折合 {amount}',
  refund: {
    action: '退款',
    title: '发起退款',
    amountMinor: '退款金额（minor）',
    amountHint: '剩余可退金额：{amount}。请输入最小货币单位金额。',
    refundable: '可退',
    invalidAmount: '退款金额必须大于 0，且不能超过剩余可退金额。',
    reason: '退款原因',
    reasonPlaceholder: '客户请求退款',
    submit: '提交退款',
    submitting: '提交中',
    success: '退款已提交',
    failed: '退款提交失败'
  },
  loadFailed: '支付订单加载失败，请刷新重试。',
  refresh: '刷新'
}
