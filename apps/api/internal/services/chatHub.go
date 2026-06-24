package services

import (
	"sync"

	"golang.org/x/net/websocket"
)

// ChatHub keeps track of the live WebSocket connections grouped by
// conversation, so that a message produced by one participant (visitor or
// staff) can be fanned out to every other participant of the same
// conversation in real time.
//
// Persistence is handled by the caller (handler -> storage); the hub only
// deals with the in-memory, ephemeral side of the chat.
type ChatHub struct {
	mu    sync.RWMutex
	rooms map[string]map[*ChatClient]struct{}
}

// ChatClient is a single live WebSocket participant within a conversation.
type ChatClient struct {
	ConversationID string
	Conn           *websocket.Conn
	// send serialises writes to the underlying connection, since a single
	// websocket.Conn must not be written to concurrently.
	send sync.Mutex
}

// NewChatHub creates an empty hub.
func NewChatHub() *ChatHub {
	return &ChatHub{
		rooms: make(map[string]map[*ChatClient]struct{}),
	}
}

// Register adds a client to its conversation room.
func (h *ChatHub) Register(c *ChatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room, ok := h.rooms[c.ConversationID]
	if !ok {
		room = make(map[*ChatClient]struct{})
		h.rooms[c.ConversationID] = room
	}
	room[c] = struct{}{}
}

// Unregister removes a client from its conversation room, cleaning up the room
// when it becomes empty.
func (h *ChatHub) Unregister(c *ChatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room, ok := h.rooms[c.ConversationID]
	if !ok {
		return
	}
	delete(room, c)
	if len(room) == 0 {
		delete(h.rooms, c.ConversationID)
	}
}

// Broadcast sends payload (already-encoded JSON) to every client in the
// conversation except the optional sender. Clients whose write fails are left
// for their own read loop to clean up.
func (h *ChatHub) Broadcast(conversationID string, payload []byte, except *ChatClient) {
	h.mu.RLock()
	room := h.rooms[conversationID]
	clients := make([]*ChatClient, 0, len(room))
	for c := range room {
		if c == except {
			continue
		}
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		c.WriteRaw(payload)
	}
}

// WriteRaw writes a pre-encoded JSON frame to the client, serialising writes.
func (c *ChatClient) WriteRaw(payload []byte) error {
	c.send.Lock()
	defer c.send.Unlock()
	return websocket.Message.Send(c.Conn, string(payload))
}
