// ==================== Common Types ====================

export interface SelectOption {
  value: string | number | boolean | null;
  label: string;
  [key: string]: any; // Support extra properties for custom templates
}

export interface BasePaginationResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  pages: number;
}

export interface FetchOptions {
  signal?: AbortSignal;
}

// ==================== API Response Types ====================

export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
}

export interface ApiError {
  detail: string;
  code?: string;
  field?: string;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  pages: number;
}

// ==================== UI State Types ====================

export type ToastType = "success" | "error" | "info" | "warning";
export type ToastDetailTone = "success" | "error" | "info" | "warning";

export interface ToastDetailItem {
  text: string;
  tone?: ToastDetailTone;
}

export type ToastDetailInput = string | ToastDetailItem;

export interface ToastOptions {
  title?: string;
  details?: ToastDetailInput[];
  copyText?: string;
  persistent?: boolean;
  duration?: number;
}

export interface Toast {
  id: string;
  type: ToastType;
  message: string;
  title?: string;
  details?: ToastDetailItem[];
  copyText?: string;
  persistent?: boolean;
  duration?: number;
  startTime?: number; // timestamp when toast was created, for progress bar
}

export interface AppState {
  sidebarCollapsed: boolean;
  loading: boolean;
  toasts: Toast[];
}

// ==================== Validation Types ====================

export interface ValidationError {
  field: string;
  message: string;
}

// ==================== Table/List Types ====================

export interface SortConfig {
  key: string;
  order: "asc" | "desc";
}

export interface FilterConfig {
  [key: string]: string | number | boolean | null | undefined;
}

export interface PaginationConfig {
  page: number;
  page_size: number;
}
