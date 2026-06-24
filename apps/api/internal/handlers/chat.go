package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/qodesrl/gardbase/apps/api/internal/services"
	"github.com/qodesrl/gardbase/apps/api/internal/storage"
	"go.uber.org/zap"
	"golang.org/x/net/websocket"
)

// ChatHandler powers the support chat widget on the landing site: website
// visitors talk to support staff over a WebSocket connection, and every
// message is persisted via the API so the conversation history survives
// reconnects and page reloads.
type ChatHandler struct {
	Hub    *services.ChatHub
	Dynamo *storage.DynamoClient
	Logger *zap.Logger
	// AllowedOrigins is the set of origins permitted to open a chat
	// WebSocket. An empty slice (or a single "*" entry) allows any origin.
	AllowedOrigins []string
}

// maxMessageBody caps the size of a single chat message body.
const maxChatMessageBody = 8 * 1024

// inboundChatMessage is what a client sends over the socket.
type inboundChatMessage struct {
	Body string `json:"body"`
}

// outboundChatEvent is what the server sends over the socket.
type outboundChatEvent struct {
	Type    string               `json:"type"` // "history" | "message" | "error"
	Message *storage.ChatMessage `json:"message,omitempty"`
	History []storage.ChatMessage `json:"history,omitempty"`
	Error   string               `json:"error,omitempty"`
}

// HandleCreateConversation starts a new support conversation and returns its
// id. Visitors call this once (e.g. when they first open the widget) and then
// connect the WebSocket using the returned id.
func (h *ChatHandler) HandleCreateConversation(c *gin.Context) {
	var req struct {
		VisitorName string `json:"visitorName"`
	}
	// Body is optional.
	_ = c.ShouldBindJSON(&req)

	conv, err := h.Dynamo.CreateChatConversation(c.Request.Context(), "", strings.TrimSpace(req.VisitorName))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, conv)
}

// HandleListMessages returns the persisted message history for a conversation.
// Used by both the visitor widget and the staff console to render history on
// load (the WebSocket also pushes history on connect).
func (h *ChatHandler) HandleListMessages(c *gin.Context) {
	conversationID := c.Param("conversationId")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversationId is required"})
		return
	}
	conv, err := h.Dynamo.GetChatConversation(c.Request.Context(), conversationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load conversation: " + err.Error()})
		return
	}
	if conv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	limit := 0
	if v := c.Query("limit"); v != "" {
		limit, _ = strconv.Atoi(v)
	}
	messages, err := h.Dynamo.ListChatMessages(c.Request.Context(), conversationID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load messages: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"conversation": conv,
		"messages":     messages,
	})
}

// HandleWebSocket upgrades the request to a WebSocket and joins the caller to a
// conversation room. The "role" query parameter selects whether the caller is
// a "visitor" (default) or "staff"; staff connections are gated by the
// staff-token middleware in front of this route.
//
// Wire protocol (JSON text frames):
//   - On connect the server sends {"type":"history","history":[...]}.
//   - Clients send {"body":"..."}; the server persists the message, echoes it
//     back as {"type":"message","message":{...}} and broadcasts the same frame
//     to every other participant in the conversation.
func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
	conversationID := c.Param("conversationId")
	if conversationID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversationId is required"})
		return
	}

	// The sender role is derived from the authenticated route: the staff
	// middleware sets "chatRole" to "staff". Visitors are the default and
	// cannot self-claim the staff role.
	sender := storage.ChatSenderVisitor
	if role, _ := c.Get("chatRole"); role == "staff" {
		sender = storage.ChatSenderStaff
	}

	ctx := c.Request.Context()
	conv, err := h.Dynamo.GetChatConversation(ctx, conversationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load conversation: " + err.Error()})
		return
	}
	if conv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
		return
	}

	server := websocket.Server{
		Handshake: h.checkOrigin,
		Handler:   websocket.Handler(h.serveConn(conversationID, sender)),
	}
	server.ServeHTTP(c.Writer, c.Request)
}

// checkOrigin validates the Origin header against the allow-list during the
// WebSocket handshake.
func (h *ChatHandler) checkOrigin(cfg *websocket.Config, req *http.Request) error {
	if len(h.AllowedOrigins) == 0 {
		return nil
	}
	origin := req.Header.Get("Origin")
	for _, allowed := range h.AllowedOrigins {
		if allowed == "*" || strings.EqualFold(allowed, origin) {
			return nil
		}
	}
	return http.ErrNotSupported
}

// serveConn returns the per-connection handler bound to a conversation/sender.
func (h *ChatHandler) serveConn(conversationID string, sender storage.ChatSender) func(*websocket.Conn) {
	return func(conn *websocket.Conn) {
		client := &services.ChatClient{ConversationID: conversationID, Conn: conn}
		h.Hub.Register(client)
		defer h.Hub.Unregister(client)
		defer conn.Close()

		// Push existing history on connect so reconnecting clients catch up.
		ctx := conn.Request().Context()
		if history, err := h.Dynamo.ListChatMessages(ctx, conversationID, 0); err == nil {
			h.sendEvent(client, outboundChatEvent{Type: "history", History: history})
		} else if h.Logger != nil {
			h.Logger.Warn("chat: failed to load history", zap.String("conversation", conversationID), zap.Error(err))
		}

		for {
			var raw string
			if err := websocket.Message.Receive(conn, &raw); err != nil {
				return // client disconnected
			}

			var in inboundChatMessage
			if err := json.Unmarshal([]byte(raw), &in); err != nil {
				h.sendEvent(client, outboundChatEvent{Type: "error", Error: "invalid message format"})
				continue
			}
			body := strings.TrimSpace(in.Body)
			if body == "" {
				continue
			}
			if len(body) > maxChatMessageBody {
				body = body[:maxChatMessageBody]
			}

			persistCtx := conn.Request().Context()
			msg, err := h.Dynamo.AppendChatMessage(persistCtx, conversationID, sender, body)
			if err != nil {
				if h.Logger != nil {
					h.Logger.Error("chat: failed to persist message", zap.String("conversation", conversationID), zap.Error(err))
				}
				h.sendEvent(client, outboundChatEvent{Type: "error", Error: "failed to deliver message"})
				continue
			}

			payload, err := json.Marshal(outboundChatEvent{Type: "message", Message: msg})
			if err != nil {
				continue
			}
			// Echo to sender (so they get the canonical id/timestamp) and
			// fan out to every other participant.
			_ = client.WriteRaw(payload)
			h.Hub.Broadcast(conversationID, payload, client)
		}
	}
}

func (h *ChatHandler) sendEvent(client *services.ChatClient, event outboundChatEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	_ = client.WriteRaw(payload)
}
