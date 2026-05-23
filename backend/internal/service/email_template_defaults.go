package service

import "time"

func emailTemplateDefVerifyCode() EmailTemplateDefinition {
	return EmailTemplateDefinition{
		Key:       EmailTemplateVerifyCode,
		Name:      "Verification code",
		Variables: []string{"SiteName", "Code"},
		BuiltIn: map[string]EmailTemplate{
			"en": builtInTemplate(EmailTemplateVerifyCode, "en", "[{{.SiteName}}] Email Verification Code", `
<h2>{{.SiteName}}</h2>
<p>Your verification code is:</p>
<p style="font-size:32px;font-weight:700;letter-spacing:6px">{{.Code}}</p>
<p>This code will expire in 15 minutes.</p>
<p>If you did not request this code, please ignore this email.</p>`),
			"zh": builtInTemplate(EmailTemplateVerifyCode, "zh", "[{{.SiteName}}] 邮箱验证码", `
<h2>{{.SiteName}}</h2>
<p>您的邮箱验证码是：</p>
<p style="font-size:32px;font-weight:700;letter-spacing:6px">{{.Code}}</p>
<p>验证码将在 15 分钟后过期。</p>
<p>如果这不是您本人操作，请忽略此邮件。</p>`),
		},
	}
}

func emailTemplateDefPasswordReset() EmailTemplateDefinition {
	return EmailTemplateDefinition{
		Key:       EmailTemplatePasswordReset,
		Name:      "Password reset",
		Variables: []string{"SiteName", "ResetURL"},
		BuiltIn: map[string]EmailTemplate{
			"en": builtInTemplate(EmailTemplatePasswordReset, "en", "[{{.SiteName}}] Password Reset Request", `
<h2>{{.SiteName}}</h2>
<p>You requested a password reset. Open the link below to set a new password:</p>
<p><a href="{{.ResetURL}}">Reset password</a></p>
<p>This link will expire in 30 minutes. If you did not request this, ignore this email.</p>
<p style="word-break:break-all">{{.ResetURL}}</p>`),
			"zh": builtInTemplate(EmailTemplatePasswordReset, "zh", "[{{.SiteName}}] 密码重置请求", `
<h2>{{.SiteName}}</h2>
<p>您已请求重置密码。请点击下方链接设置新密码：</p>
<p><a href="{{.ResetURL}}">重置密码</a></p>
<p>此链接将在 30 分钟后失效。如果您没有请求重置密码，请忽略此邮件。</p>
<p style="word-break:break-all">{{.ResetURL}}</p>`),
		},
	}
}

func emailTemplateDefPaymentSuccess() EmailTemplateDefinition {
	return EmailTemplateDefinition{
		Key:       EmailTemplatePaymentSuccess,
		Name:      "Payment success",
		Variables: []string{"SiteName", "OrderNo", "ProductType", "Amount", "Currency"},
		BuiltIn: map[string]EmailTemplate{
			"en": builtInTemplate(EmailTemplatePaymentSuccess, "en", "[{{.SiteName}}] Payment Successful", `
<h2>Payment successful</h2>
<p>Your order {{.OrderNo}} has been paid successfully.</p>
<p>Product: {{.ProductType}}</p>
<p>Amount: {{.Amount}} {{.Currency}}</p>`),
			"zh": builtInTemplate(EmailTemplatePaymentSuccess, "zh", "[{{.SiteName}}] 支付成功", `
<h2>支付成功</h2>
<p>您的订单 {{.OrderNo}} 已支付成功。</p>
<p>商品：{{.ProductType}}</p>
<p>金额：{{.Amount}} {{.Currency}}</p>`),
		},
	}
}

func emailTemplateDefBalanceLow() EmailTemplateDefinition {
	return EmailTemplateDefinition{
		Key:       EmailTemplateBalanceLow,
		Name:      "Balance low",
		Variables: []string{"SiteName", "Balance", "Currency", "Threshold"},
		BuiltIn: map[string]EmailTemplate{
			"en": builtInTemplate(EmailTemplateBalanceLow, "en", "[{{.SiteName}}] Balance Reminder", `
<h2>Balance reminder</h2>
<p>Your current balance is {{.Balance}} {{.Currency}}, below the reminder threshold {{.Threshold}}.</p>
<p>Please top up in time to avoid service interruption.</p>`),
			"zh": builtInTemplate(EmailTemplateBalanceLow, "zh", "[{{.SiteName}}] 余额提醒", `
<h2>余额提醒</h2>
<p>您的当前余额为 {{.Balance}} {{.Currency}}，已低于提醒阈值 {{.Threshold}}。</p>
<p>请及时充值，避免服务中断。</p>`),
		},
	}
}

func emailTemplateDefSubscriptionExpiring() EmailTemplateDefinition {
	return EmailTemplateDefinition{
		Key:       EmailTemplateSubscriptionExpiring,
		Name:      "Subscription expiring",
		Variables: []string{"SiteName", "GroupName", "ExpiresAt", "DaysLeft"},
		BuiltIn: map[string]EmailTemplate{
			"en": builtInTemplate(EmailTemplateSubscriptionExpiring, "en", "[{{.SiteName}}] Subscription Expiring", `
<h2>Subscription expiring</h2>
<p>Your subscription for {{.GroupName}} will expire at {{.ExpiresAt}}.</p>
<p>Days left: {{.DaysLeft}}</p>`),
			"zh": builtInTemplate(EmailTemplateSubscriptionExpiring, "zh", "[{{.SiteName}}] 订阅即将到期", `
<h2>订阅即将到期</h2>
<p>您的 {{.GroupName}} 订阅将在 {{.ExpiresAt}} 到期。</p>
<p>剩余天数：{{.DaysLeft}}</p>`),
		},
	}
}

func emailTemplateDefScheduledTestResult() EmailTemplateDefinition {
	return EmailTemplateDefinition{
		Key:       EmailTemplateScheduledTestResult,
		Name:      "Scheduled test result",
		Variables: []string{"SiteName", "AccountName", "AccountID", "Platform", "PlanID", "ResultID", "Model", "Status", "LatencyMs", "Error", "ConsecutiveFailures", "CompletedAt", "NextRun"},
		BuiltIn: map[string]EmailTemplate{
			"en": builtInTemplate(EmailTemplateScheduledTestResult, "en", "[{{.SiteName}}] Scheduled Test {{.Status}}", `
<h2>Scheduled test result</h2>
<p>Account: {{.AccountName}} (#{{.AccountID}})</p>
<p>Platform: {{.Platform}}</p>
<p>Plan: {{.PlanID}} / Result: {{.ResultID}}</p>
<p>Model: {{.Model}}</p>
<p>Status: {{.Status}}</p>
<p>Latency: {{.LatencyMs}} ms</p>
<p>Error: {{.Error}}</p>
<p>Consecutive failures: {{.ConsecutiveFailures}}</p>
<p>Completed at: {{.CompletedAt}}</p>
<p>Next run: {{.NextRun}}</p>`),
			"zh": builtInTemplate(EmailTemplateScheduledTestResult, "zh", "[{{.SiteName}}] 定时测试{{.Status}}", `
<h2>定时测试结果</h2>
<p>账号：{{.AccountName}} (#{{.AccountID}})</p>
<p>平台：{{.Platform}}</p>
<p>计划：{{.PlanID}} / 结果：{{.ResultID}}</p>
<p>模型：{{.Model}}</p>
<p>状态：{{.Status}}</p>
<p>耗时：{{.LatencyMs}} ms</p>
<p>错误：{{.Error}}</p>
<p>连续失败：{{.ConsecutiveFailures}}</p>
<p>完成时间：{{.CompletedAt}}</p>
<p>下次运行：{{.NextRun}}</p>`),
		},
	}
}

func builtInTemplate(key, locale, subject, body string) EmailTemplate {
	now := time.Unix(0, 0).UTC()
	return EmailTemplate{
		TemplateKey: key,
		Locale:      locale,
		Subject:     subject,
		Body:        body,
		Enabled:     true,
		IsCustom:    false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
