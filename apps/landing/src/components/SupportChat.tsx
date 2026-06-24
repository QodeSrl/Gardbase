import React, { useEffect, useRef, useState } from "react";
import { LuMessageCircle, LuX, LuSend } from "react-icons/lu";
import { useSupportChat } from "@/lib/useSupportChat";

// SupportChat renders a floating support-chat widget for the landing site.
// Visitors chat with support staff in real time over a WebSocket, and the
// conversation history is persisted by the API so it survives reloads.
const SupportChat: React.FC = () => {
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState("");
  const { messages, status, error, connect, send } = useSupportChat();
  const scrollRef = useRef<HTMLDivElement | null>(null);

  // Lazily connect the first time the panel is opened.
  useEffect(() => {
    if (open) connect();
  }, [open, connect]);

  // Keep the newest message in view.
  useEffect(() => {
    if (open && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages, open]);

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!draft.trim()) return;
    send(draft);
    setDraft("");
  };

  const statusLabel =
    status === "online"
      ? "Online"
      : status === "connecting"
        ? "Connecting…"
        : status === "offline"
          ? "Reconnecting…"
          : "Support";

  const statusDotClass =
    status === "online"
      ? "bg-emerald-400"
      : status === "connecting"
        ? "bg-amber-400"
        : status === "offline"
          ? "bg-rose-400"
          : "bg-subtle";

  return (
    <div className="fixed bottom-5 right-5 z-50 flex flex-col items-end">
      {open && (
        <div className="mb-3 flex h-[28rem] w-[22rem] max-w-[calc(100vw-2.5rem)] flex-col overflow-hidden rounded-2xl border border-line bg-card shadow-2xl shadow-black/40">
          {/* Header */}
          <div className="flex items-center justify-between border-b border-line bg-brand px-4 py-3">
            <div className="flex items-center gap-2">
              <span className={`h-2.5 w-2.5 rounded-full ${statusDotClass}`} aria-hidden />
              <div className="leading-tight">
                <p className="text-sm font-semibold text-white">Gardbase Support</p>
                <p className="text-xs text-slate-300">{statusLabel}</p>
              </div>
            </div>
            <button
              onClick={() => setOpen(false)}
              aria-label="Close support chat"
              className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-slate-300 transition-colors hover:bg-white/10 hover:text-white"
            >
              <LuX className="h-4 w-4" />
            </button>
          </div>

          {/* Messages */}
          <div ref={scrollRef} className="flex-1 space-y-3 overflow-y-auto px-4 py-4">
            {messages.length === 0 && (
              <p className="mt-6 text-center text-sm text-subtle">
                Hi! 👋 Ask us anything about Gardbase and a team member will reply here.
              </p>
            )}
            {messages.map((m) => {
              const fromVisitor = m.sender === "visitor";
              return (
                <div
                  key={m.id}
                  className={`flex ${fromVisitor ? "justify-end" : "justify-start"}`}
                >
                  <div
                    className={`max-w-[80%] whitespace-pre-wrap break-words rounded-2xl px-3.5 py-2 text-sm ${
                      fromVisitor
                        ? "rounded-br-sm bg-accent text-white"
                        : "rounded-bl-sm bg-card2 text-fg"
                    }`}
                  >
                    {m.body}
                  </div>
                </div>
              );
            })}
            {error && (
              <p className="text-center text-xs text-rose-400" role="alert">
                {error}
              </p>
            )}
          </div>

          {/* Composer */}
          <form onSubmit={onSubmit} className="flex items-center gap-2 border-t border-line p-3">
            <input
              type="text"
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              placeholder="Type a message…"
              aria-label="Message"
              className="min-w-0 flex-1 rounded-xl border border-line bg-bg2 px-3 py-2 text-sm text-fg outline-none placeholder:text-subtle focus:border-accent"
            />
            <button
              type="submit"
              disabled={!draft.trim() || status !== "online"}
              aria-label="Send message"
              className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-accent text-white transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40"
            >
              <LuSend className="h-4 w-4" />
            </button>
          </form>
        </div>
      )}

      {/* Launcher */}
      <button
        onClick={() => setOpen((v) => !v)}
        aria-label={open ? "Minimize support chat" : "Open support chat"}
        aria-expanded={open}
        className="inline-flex h-14 w-14 items-center justify-center rounded-full bg-accent text-white shadow-lg shadow-accent/30 transition-transform hover:scale-105"
      >
        {open ? <LuX className="h-6 w-6" /> : <LuMessageCircle className="h-6 w-6" />}
      </button>
    </div>
  );
};

export default SupportChat;
