import { useCallback, useEffect, useRef, useState } from "react";
import {
  ChatEvent,
  ChatMessage,
  createConversation,
  loadStoredConversationId,
  storeConversationId,
  visitorSocketUrl,
} from "@/lib/chat";

export type ConnectionStatus = "idle" | "connecting" | "online" | "offline";

export interface SupportChatState {
  messages: ChatMessage[];
  status: ConnectionStatus;
  error: string | null;
  /** Open (and lazily create) the conversation and connect the socket. */
  connect: () => void;
  /** Send a visitor message over the socket. */
  send: (body: string) => void;
}

// useSupportChat manages a single visitor support conversation: it lazily
// creates the conversation, opens the WebSocket, surfaces persisted history
// pushed by the server on connect, appends live messages, and transparently
// reconnects with a capped backoff.
export function useSupportChat(): SupportChatState {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [status, setStatus] = useState<ConnectionStatus>("idle");
  const [error, setError] = useState<string | null>(null);

  const socketRef = useRef<WebSocket | null>(null);
  const conversationIdRef = useRef<string | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimerRef = useRef<number | null>(null);
  const startedRef = useRef(false);
  const closedByUserRef = useRef(false);

  const upsertMessage = useCallback((incoming: ChatMessage) => {
    setMessages((prev) => {
      if (prev.some((m) => m.id === incoming.id)) {
        return prev;
      }
      return [...prev, incoming];
    });
  }, []);

  const openSocket = useCallback(
    (conversationId: string) => {
      // Guard against duplicate sockets.
      if (
        socketRef.current &&
        (socketRef.current.readyState === WebSocket.OPEN ||
          socketRef.current.readyState === WebSocket.CONNECTING)
      ) {
        return;
      }

      setStatus("connecting");
      const socket = new WebSocket(visitorSocketUrl(conversationId));
      socketRef.current = socket;

      socket.onopen = () => {
        reconnectAttemptsRef.current = 0;
        setStatus("online");
        setError(null);
      };

      socket.onmessage = (ev) => {
        let event: ChatEvent;
        try {
          event = JSON.parse(ev.data as string) as ChatEvent;
        } catch {
          return;
        }
        switch (event.type) {
          case "history":
            // Server is the source of truth for history; replace local state.
            setMessages(event.history ?? []);
            break;
          case "message":
            if (event.message) upsertMessage(event.message);
            break;
          case "error":
            setError(event.error || "Something went wrong");
            break;
        }
      };

      socket.onerror = () => {
        setStatus("offline");
      };

      socket.onclose = () => {
        socketRef.current = null;
        if (closedByUserRef.current) {
          setStatus("idle");
          return;
        }
        setStatus("offline");
        // Reconnect with exponential backoff capped at 15s.
        const attempt = reconnectAttemptsRef.current++;
        const delay = Math.min(1000 * 2 ** attempt, 15000);
        reconnectTimerRef.current = window.setTimeout(() => {
          if (conversationIdRef.current) {
            openSocket(conversationIdRef.current);
          }
        }, delay);
      };
    },
    [upsertMessage]
  );

  const connect = useCallback(() => {
    if (startedRef.current) {
      // Already started; just ensure the socket is alive.
      if (conversationIdRef.current && !socketRef.current) {
        openSocket(conversationIdRef.current);
      }
      return;
    }
    startedRef.current = true;
    closedByUserRef.current = false;
    setStatus("connecting");

    const existing = loadStoredConversationId();
    if (existing) {
      conversationIdRef.current = existing;
      openSocket(existing);
      return;
    }

    createConversation()
      .then((conv) => {
        conversationIdRef.current = conv.conversationId;
        storeConversationId(conv.conversationId);
        openSocket(conv.conversationId);
      })
      .catch((err: unknown) => {
        startedRef.current = false;
        setStatus("offline");
        setError(err instanceof Error ? err.message : "Failed to start chat");
      });
  }, [openSocket]);

  const send = useCallback((body: string) => {
    const trimmed = body.trim();
    const socket = socketRef.current;
    if (!trimmed || !socket || socket.readyState !== WebSocket.OPEN) {
      return;
    }
    socket.send(JSON.stringify({ body: trimmed }));
  }, []);

  // Tidy up timers and sockets on unmount.
  useEffect(() => {
    return () => {
      closedByUserRef.current = true;
      if (reconnectTimerRef.current !== null) {
        window.clearTimeout(reconnectTimerRef.current);
      }
      socketRef.current?.close();
    };
  }, []);

  return { messages, status, error, connect, send };
}
