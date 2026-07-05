export const COMMON_ERROR_STATUS_CODES = [
  400, 401, 403, 404, 408, 413, 422, 429, 499, 500, 502, 503, 504, 529,
];

export function mapErrorSortKey(key: string): string {
  return key === "status" ? "status_code" : key;
}

export function statusCodeBadgeClass(code: number): string {
  if (code >= 500) {
    return "bg-red-50 text-red-700 ring-red-600/20 dark:bg-red-900/30 dark:text-red-400 dark:ring-red-500/30";
  }
  if (code === 429) {
    return "bg-purple-50 text-purple-700 ring-purple-600/20 dark:bg-purple-900/30 dark:text-purple-400 dark:ring-purple-500/30";
  }
  if (code >= 400) {
    return "bg-amber-50 text-amber-700 ring-amber-600/20 dark:bg-amber-900/30 dark:text-amber-400 dark:ring-amber-500/30";
  }
  return "bg-gray-50 text-gray-700 ring-gray-600/20 dark:bg-gray-900/30 dark:text-gray-400 dark:ring-gray-500/30";
}
