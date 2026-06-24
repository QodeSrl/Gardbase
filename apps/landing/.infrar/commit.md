# Changes

Added a floating support-chat widget to the landing site. Visitors chat with support staff in real time over a WebSocket, and the API persists the conversation so history is restored on reconnect/reload.

## What changed
- `src/lib/chat.ts`: typed client for the chat backend (`/api/chat/*`). Resolves the API base from `VITE_API_BASE_URL` (defaults to `https://api.gardbase.com`), derives the `ws(s)://` URL, defines the `ChatMessage`/`ChatConversation`/`ChatEvent` types, exposes `createConversation`, and persists the visitor's conversation id in `localStorage` so returning visitors keep their history.
- `src/lib/useSupportChat.ts`: React hook managing one visitor conversation — lazily creates the conversation, opens the WebSocket, replaces local state with server-pushed history, de-dupes/append live messages, sends visitor messages, and transparently reconnects with exponential backoff capped at 15s. Cleans up timers/socket on unmount.
- `src/components/SupportChat.tsx`: floating launcher + chat panel using existing theme tokens and `react-icons/lu`. Shows a connection status indicator, renders the message thread (visitor vs staff bubbles), auto-scrolls, and disables sending while offline.
- `src/pages/MainPage.tsx`: mounted `<SupportChat />` at the page root.

## Notes
- Configure `VITE_API_BASE_URL` at build time to point the widget at the API; without it the widget targets the production API.
- Uses the `@/` path alias and `react-icons` (already configured/declared). No new dependencies were added.
- No Node toolchain is available in the sandbox, so this was verified by reading the source (alias, theme tokens, and `react-icons` imports all match existing usage).
