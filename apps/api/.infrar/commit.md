# Changes

Added a real-time support-chat backend so landing-site visitors can talk to support staff over WebSockets, with every message persisted in DynamoDB so the conversation history survives reconnects and page reloads.

## What changed
- `internal/storage/chat.go`: DynamoDB-backed conversations and messages stored in a dedicated single table (`DYNAMO_CHAT_TABLE`). Layout: conversation meta `pk="CONV#<id>", sk="META"`; messages `pk="CONV#<id>", sk="MSG#<RFC3339Nano>#<msgId>"` so a Query returns them chronologically. Adds `ChatConversation`/`ChatMessage` types and `CreateChatConversation`, `GetChatConversation`, `AppendChatMessage`, `ListChatMessages` (limit defaults 100, max 500). Records carry a `ttl` for retention (default 90 days).
- `internal/storage/dynamo.go`: added `ChatTable`/`ChatRetention` fields to `DynamoClient` and a `chatTable` parameter to `NewDynamoClient`.
- `internal/services/chatHub.go`: in-memory hub that groups live WebSocket clients by conversation and fans a message out to the other participants; per-connection writes are serialised since a `websocket.Conn` must not be written concurrently.
- `internal/handlers/chat.go`: REST endpoints to create a conversation and load history, plus a WebSocket handler that pushes existing history on connect, persists each inbound message, echoes it back to the sender (canonical id/timestamp) and broadcasts it to the room. The handshake enforces an `Origin` allow-list.
- `internal/middleware/staffChat.go`: shared-token auth for the staff side (`STAFF_CHAT_TOKEN`), accepted via `Authorization: Bearer` header or a `token` query param (browsers can't set custom headers on a WS upgrade). Sets `chatRole=staff` so the handler labels staff messages.
- `cmd/server/main.go`: wired the routes under `/api/chat` (visitor: `POST /conversations`, `GET /conversations/:id/messages`, `GET /ws/:id`) and `/api/chat/staff` (token-gated history + WebSocket). The whole feature is gated on `DYNAMO_CHAT_TABLE` being set (logs a warning and skips otherwise). Added a `getEnvAsSlice` helper for `CHAT_ALLOWED_ORIGINS`.

## New environment variables
- `DYNAMO_CHAT_TABLE` (optional): enables the chat feature when set.
- `STAFF_CHAT_TOKEN` (optional): shared bearer token for staff routes.
- `CHAT_ALLOWED_ORIGINS` (optional, default `*`): comma-separated WebSocket origin allow-list.

## Notes
- WebSocket support uses `golang.org/x/net/websocket` (x/net was already a required module; its `// indirect` marker was dropped in go.mod). No new third-party modules were added — `uuid` and `aws-sdk-go-v2` are already in use.
- No Go toolchain is available in the sandbox, so the build was verified by reading the source: imports resolve, `NewDynamoClient`'s single caller was updated, and helpers (`getEnv`, `getEnvAsSlice`) exist.
