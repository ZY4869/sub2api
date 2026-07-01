const DEFAULT_API_BASE_URL = "/api/v1";

function getConfiguredApiBaseURL(): string {
  return String(import.meta.env.VITE_API_BASE_URL || DEFAULT_API_BASE_URL).trim() || DEFAULT_API_BASE_URL;
}

function trimTrailingSlash(value: string): string {
  return value.replace(/\/+$/g, "");
}

function normalizePath(path: string): string {
  const trimmed = String(path || "").trim();
  return trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
}

function isAbsoluteURL(value: string): boolean {
  return /^https?:\/\//i.test(value);
}

export function buildApiUrl(path: string, baseURL = getConfiguredApiBaseURL()): string {
  const normalizedPath = normalizePath(path);
  const normalizedBase = trimTrailingSlash(String(baseURL || DEFAULT_API_BASE_URL).trim() || DEFAULT_API_BASE_URL);
  return `${normalizedBase}${normalizedPath}`;
}

export function buildBackendRootUrl(path: string, baseURL = getConfiguredApiBaseURL()): string {
  const normalizedPath = normalizePath(path);
  const normalizedBase = String(baseURL || DEFAULT_API_BASE_URL).trim() || DEFAULT_API_BASE_URL;

  if (!isAbsoluteURL(normalizedBase)) {
    return normalizedPath;
  }

  try {
    const parsed = new URL(normalizedBase);
    return `${parsed.origin}${normalizedPath}`;
  } catch {
    return normalizedPath;
  }
}

export function buildAbsoluteApiUrl(path: string, baseURL = getConfiguredApiBaseURL()): string {
  const apiPath = buildApiUrl(path, baseURL);
  if (isAbsoluteURL(apiPath)) {
    return apiPath;
  }
  if (typeof window !== "undefined" && window.location?.origin) {
    return `${trimTrailingSlash(window.location.origin)}/${apiPath.replace(/^\/+/g, "")}`;
  }
  return apiPath;
}

export function buildAbsoluteBackendRootUrl(path: string, baseURL = getConfiguredApiBaseURL()): string {
  const rootPath = buildBackendRootUrl(path, baseURL);
  if (isAbsoluteURL(rootPath)) {
    return rootPath;
  }
  if (typeof window !== "undefined" && window.location?.origin) {
    return `${trimTrailingSlash(window.location.origin)}${normalizePath(rootPath)}`;
  }
  return rootPath;
}

