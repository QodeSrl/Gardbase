// Client for the Gardbase support-chat backend (see apps/api: /api/chat/*).
//
// The widget talks to two kinds of endpoints:
//   - REST, to create a conversation and load persisted history; and
//   - a WebSocket, for real-time delivery of new messages.
//
// The API base URL is configured at build time via VITE_API_BASE_URL and
// defaults to the production API. The WebSocket URL is derived from it by
// swapping the http(s) scheme for ws(s).

const DEFAULT_API_BASE_URL = "https://api.gardbase.com";

export const apiBaseUrl: string = (
  import.meta.env.VITE_API_BASE_URL || DEFAULT_API_BASE_URL
).replace(/\/+$/, "");

function wsBaseUrl(): string {
  return apiBaseUrl.replace(/^http/i, "ws");
}

export type ChatSender = "visitor" | "staff";

export interface ChatMessage {
  id: string;
  conversationId: string;
  sender: ChatSender;
  body: string;
  createdAt: string;
}

export interface ChatConversation {
  conversationId: string;
  visitorName?: string;
  createdAt: string;
  updatedAt: string;
}

// Server -> client frames pushed over the WebSocket.
export type ChatEvent =
  | { type: "history"; history: ChatMessage[] }
  | { type: "message"; message: ChatMessage }
  | { type: "error"; error: string };

// localStorage key used to remember the visitor's conversation id across page
// reloads so history persists for returning visitors.
const STORAGE_KEY = "gardbase.support.conversationId";

export function loadStoredConversationId(): string | null {
  try {
    return window.localStorage.getItem(STORAGE_KEY);
  } catch {
    return null;
  }
}

export function storeConversationId(id: string): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, id);
  } catch {
    /* ignore (private mode / disabled storage) */
  }
}

export function clearStoredConversationId(): void {
  try {
    window.localStorage.removeItem(STORAGE_KEY);
  } catch {
    /* ignore */
  }
}

// createConversation starts a brand-new support conversation and returns its
// metadata (including the id used to open the WebSocket).
export async function createConversation(visitorName?: string): Promise<ChatConversation> {
  const res = await fetch(`${apiBaseUrl}/api/chat/conversations`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ visitorName: visitorName ?? "" }),
  });
  if (!res.ok) {
    throw new Error(`Failed to start conversation (${res.status})`);
  }
  return (await res.json()) as ChatConversation;
}

// visitorSocketUrl builds the WebSocket URL a visitor uses to join a
// conversation room.
export function visitorSocketUrl(conversationId: string): string {
  return `${wsBaseUrl()}/api/chat/ws/${encodeURIComponent(conversationId)}`;
}
