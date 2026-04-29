import { apiClient } from "../client";
import { getLocale } from "@/i18n";

export type AdminModelDebugKeyMode = "saved" | "manual";
export type AdminModelDebugProtocol = "openai" | "anthropic" | "gemini";
export type AdminModelDebugEndpointKind =
  | "responses"
  | "chat_completions"
  | "messages"
  | "generate_content";
export type AdminModelDebugEventType =
  | "start"
  | "request_preview"
  | "response_headers"
  | "content"
  | "final"
  | "error";

export interface AdminModelDebugRunRequest {
  key_mode: AdminModelDebugKeyMode;
  api_key_id?: number;
  manual_api_key?: string;
  protocol: AdminModelDebugProtocol;
  endpoint_kind: AdminModelDebugEndpointKind;
  model: string;
  stream: boolean;
  request_body: Record<string, any>;
}

export interface AdminModelDebugStreamEvent extends Record<string, any> {
  type: AdminModelDebugEventType;
}

export interface AdminModelDebugStreamOptions {
  signal?: AbortSignal;
  onEvent: (event: AdminModelDebugStreamEvent) => void;
}

export async function runModelDebugStream(
  payload: AdminModelDebugRunRequest,
  options: AdminModelDebugStreamOptions,
): Promise<void> {
  const response = await fetch(resolveAdminModelDebugURL(), {
    method: "POST",
    headers: buildHeaders(),
    body: JSON.stringify(payload),
    signal: options.signal,
  });

  if (!response.ok) {
    throw new Error(await readDebugErrorMessage(response));
  }
  if (!response.body) {
    throw new Error("No response body received");
  }

  await readSSEStream(response.body, options.onEvent);
}

function resolveAdminModelDebugURL(): string {
  const baseURL = String(apiClient.defaults.baseURL || "/api/v1").trim();
  const normalizedPath = "/admin/models/debug/run";
  if (/^https?:\/\//i.test(baseURL)) {
    return `${baseURL.replace(/\/+$/g, "")}${normalizedPath}`;
  }
  if (typeof window !== "undefined" && window.location?.origin) {
    return `${window.location.origin.replace(/\/+$/g, "")}/${baseURL.replace(/^\/+|\/+$/g, "")}${normalizedPath}`;
  }
  return `/api/v1${normalizedPath}`;
}

function buildHeaders(): HeadersInit {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    Accept: "text/event-stream",
    "Accept-Language": getLocale(),
  };
  const token = localStorage.getItem("auth_token");
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  return headers;
}

async function readDebugErrorMessage(response: Response): Promise<string> {
  try {
    const contentType = response.headers.get("content-type") || "";
    if (contentType.includes("application/json")) {
      const payload = await response.json();
      const message = String(
        payload?.message ||
          payload?.error ||
          payload?.detail ||
          payload?.msg ||
          "",
      ).trim();
      if (message) {
        return message;
      }
    }
    const text = (await response.text()).trim();
    if (text) {
      return text;
    }
  } catch {
    // Ignore parser failures and fall back to status text.
  }
  return `HTTP ${response.status}`;
}

async function readSSEStream(
  stream: ReadableStream<Uint8Array>,
  onEvent: (event: AdminModelDebugStreamEvent) => void,
) {
  const reader = stream.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }

    buffer += decoder.decode(value, { stream: true });
    const chunks = buffer.split("\n\n");
    buffer = chunks.pop() || "";
    for (const chunk of chunks) {
      const event = parseSSEChunk(chunk);
      if (event) {
        onEvent(event);
      }
    }
  }

  if (buffer.trim()) {
    const event = parseSSEChunk(buffer);
    if (event) {
      onEvent(event);
    }
  }
}

function parseSSEChunk(chunk: string): AdminModelDebugStreamEvent | null {
  let eventType: AdminModelDebugEventType | "" = "";
  const dataLines: string[] = [];

  for (const line of chunk.split(/\r?\n/)) {
    if (line.startsWith("event:")) {
      eventType = line.slice("event:".length).trim() as AdminModelDebugEventType;
      continue;
    }
    if (line.startsWith("data:")) {
      dataLines.push(line.slice("data:".length).trim());
    }
  }

  if (!eventType || dataLines.length === 0) {
    return null;
  }

  const data = dataLines.join("\n");
  try {
    const payload = JSON.parse(data);
    return {
      ...(typeof payload === "object" && payload ? payload : { value: payload }),
      type: eventType,
    };
  } catch {
    return {
      type: eventType,
      raw: data,
    };
  }
}

const modelDebugAPI = {
  runModelDebugStream,
};

export default modelDebugAPI;
