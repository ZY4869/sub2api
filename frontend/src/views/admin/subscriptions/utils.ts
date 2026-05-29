import type { ComposerTranslation } from "vue-i18n";

export function getSubscriptionDaysRemaining(
  expiresAt: string,
): number | null {
  const now = new Date();
  const expires = new Date(expiresAt);
  const diff = expires.getTime() - now.getTime();
  if (diff < 0) return null;
  return Math.ceil(diff / (1000 * 60 * 60 * 24));
}

export function isSubscriptionExpiringSoon(expiresAt: string): boolean {
  const days = getSubscriptionDaysRemaining(expiresAt);
  return days !== null && days <= 7;
}

export function getSubscriptionProgressWidth(
  used: number | null | undefined,
  limit: number | null,
): string {
  if (!limit || limit === 0) return "0%";
  const usedValue = used ?? 0;
  const percentage = Math.min((usedValue / limit) * 100, 100);
  return `${percentage}%`;
}

export function getSubscriptionProgressClass(
  used: number | null | undefined,
  limit: number | null,
): string {
  if (!limit || limit === 0) return "bg-gray-400";
  const usedValue = used ?? 0;
  const percentage = (usedValue / limit) * 100;
  if (percentage >= 90) return "bg-red-500";
  if (percentage >= 70) return "bg-orange-500";
  return "bg-green-500";
}

export function formatSubscriptionResetTime(
  windowStart: string,
  period: "daily" | "weekly" | "monthly",
  t: ComposerTranslation,
): string {
  if (!windowStart) return t("admin.subscriptions.windowNotActive");

  const start = new Date(windowStart);
  const now = new Date();
  let resetTime: Date;
  switch (period) {
    case "daily":
      resetTime = new Date(start.getTime() + 24 * 60 * 60 * 1000);
      break;
    case "weekly":
      resetTime = new Date(start.getTime() + 7 * 24 * 60 * 60 * 1000);
      break;
    case "monthly":
      resetTime = new Date(start.getTime() + 30 * 24 * 60 * 60 * 1000);
      break;
  }

  const diffMs = resetTime.getTime() - now.getTime();
  if (diffMs <= 0) return t("admin.subscriptions.windowNotActive");

  const diffSeconds = Math.floor(diffMs / 1000);
  const days = Math.floor(diffSeconds / 86400);
  const hours = Math.floor((diffSeconds % 86400) / 3600);
  const minutes = Math.floor((diffSeconds % 3600) / 60);

  if (days > 0) {
    return t("admin.subscriptions.resetInDaysHours", { days, hours });
  }
  if (hours > 0) {
    return t("admin.subscriptions.resetInHoursMinutes", { hours, minutes });
  }
  return t("admin.subscriptions.resetInMinutes", { minutes });
}
