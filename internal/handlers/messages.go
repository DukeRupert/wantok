package handlers

import (
	"context"
	"encoding/json"
	"html"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/dukerupert/wantok/internal/auth"
	"github.com/dukerupert/wantok/internal/realtime"
	"github.com/dukerupert/wantok/internal/render"
	"github.com/dukerupert/wantok/internal/store"
	"github.com/dukerupert/wantok/internal/validate"
)

// ConversationListItem represents a conversation in the sidebar.
type ConversationListItem struct {
	UserID          int64  `json:"user_id"`
	DisplayName     string `json:"display_name"`
	LastMessage     string `json:"last_message"`
	LastMessageTime string `json:"last_message_time"`
}

// MessageItem represents a single message in a conversation.
type MessageItem struct {
	ID         int64  `json:"id"`
	Content    string `json:"content"`
	SenderID   int64  `json:"sender_id"`
	SenderName string `json:"sender_name"`
	CreatedAt  string `json:"created_at"`
	IsSent     bool   `json:"is_sent"`
}

// ChatPageData holds data for the chat template.
type ChatPageData struct {
	Conversations   []ConversationListItem
	ActiveUserID    int64
	ActiveUserName  string
	Messages        []MessageItem
	CurrentUserID   int64
	CurrentUserName string
	IsAdmin         bool
}

// HandleChatPage renders the main chat interface.
func HandleChatPage(queries *store.Queries, renderer *render.Renderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := auth.GetUser(ctx)

		// Fetch conversations list
		conversations := getConversationsList(queries, ctx, user.ID)

		// Check for ?user= query param to load specific conversation
		data := ChatPageData{
			Conversations:   conversations,
			CurrentUserID:   user.ID,
			CurrentUserName: user.DisplayName,
			IsAdmin:         user.IsAdmin,
		}

		userIDParam := r.URL.Query().Get("user")
		if userIDParam != "" {
			otherUserID, err := strconv.ParseInt(userIDParam, 10, 64)
			if err == nil && otherUserID != user.ID {
				// Load messages for this conversation
				otherUser, err := queries.GetUserByID(ctx, otherUserID)
				if err == nil {
					data.ActiveUserID = otherUserID
					data.ActiveUserName = otherUser.DisplayName

					// Fetch messages
					msgs, err := queries.GetConversationMessages(ctx, store.GetConversationMessagesParams{
						SenderID:      user.ID,
						RecipientID:   otherUserID,
						SenderID_2:    otherUserID,
						RecipientID_2: user.ID,
						Limit:         50,
						Offset:        0,
					})
					if err == nil {
						data.Messages = make([]MessageItem, len(msgs))
						for i, m := range msgs {
							data.Messages[i] = MessageItem{
								ID:         m.ID,
								Content:    m.Content,
								SenderID:   m.SenderID,
								SenderName: m.SenderDisplayName,
								CreatedAt:  m.CreatedAt,
								IsSent:     m.SenderID == user.ID,
							}
						}
					}
				}
			}
		}

		if err := renderer.Render(w, "chat", data); err != nil {
			slog.Error("failed to render chat page", "type", "request", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

// HandleGetConversations returns the conversation list as JSON.
func HandleGetConversations(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := auth.GetUser(ctx)

		conversations := getConversationsList(queries, ctx, user.ID)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(conversations); err != nil {
			slog.Error("failed to encode conversations", "type", "request", "error", err)
		}
	}
}

// HandleGetMessages returns messages for a conversation as JSON.
func HandleGetMessages(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := auth.GetUser(ctx)

		// Parse userID from path
		otherUserID, err := strconv.ParseInt(r.PathValue("userID"), 10, 64)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Prevent messaging self
		if otherUserID == user.ID {
			http.Error(w, "Cannot message yourself", http.StatusBadRequest)
			return
		}

		// Parse pagination params
		limit := int64(50)
		offset := int64(0)
		if l := r.URL.Query().Get("limit"); l != "" {
			if parsed, err := strconv.ParseInt(l, 10, 64); err == nil && parsed > 0 && parsed <= 100 {
				limit = parsed
			}
		}
		if o := r.URL.Query().Get("offset"); o != "" {
			if parsed, err := strconv.ParseInt(o, 10, 64); err == nil && parsed >= 0 {
				offset = parsed
			}
		}

		// Fetch messages
		msgs, err := queries.GetConversationMessages(ctx, store.GetConversationMessagesParams{
			SenderID:      user.ID,
			RecipientID:   otherUserID,
			SenderID_2:    otherUserID,
			RecipientID_2: user.ID,
			Limit:         limit,
			Offset:        offset,
		})
		if err != nil {
			slog.Error("failed to get messages", "type", "request", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Transform to MessageItem slice
		messages := make([]MessageItem, len(msgs))
		for i, m := range msgs {
			messages[i] = MessageItem{
				ID:         m.ID,
				Content:    m.Content,
				SenderID:   m.SenderID,
				SenderName: m.SenderDisplayName,
				CreatedAt:  m.CreatedAt,
				IsSent:     m.SenderID == user.ID,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(messages); err != nil {
			slog.Error("failed to encode messages", "type", "request", "error", err)
		}
	}
}

// HandleSendMessage creates a new message in a conversation.
func HandleSendMessage(queries *store.Queries, hub *realtime.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := auth.GetUser(ctx)

		// Parse userID from path
		recipientID, err := strconv.ParseInt(r.PathValue("userID"), 10, 64)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Prevent messaging self
		if recipientID == user.ID {
			http.Error(w, "Cannot message yourself", http.StatusBadRequest)
			return
		}

		// Parse form
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		content := strings.TrimSpace(r.FormValue("content"))
		if err := validate.Message(content); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Verify recipient exists
		recipient, err := queries.GetUserByID(ctx, recipientID)
		if err != nil {
			http.Error(w, "Recipient not found", http.StatusNotFound)
			return
		}

		// Create message
		msg, err := queries.CreateMessage(ctx, store.CreateMessageParams{
			SenderID:    user.ID,
			RecipientID: recipientID,
			Content:     content,
		})
		if err != nil {
			slog.Error("failed to create message", "type", "request", "error", err)
			http.Error(w, "Failed to send message", http.StatusInternalServerError)
			return
		}

		slog.Info("message sent", "type", "request", "from", user.ID, "to", recipientID, "message_id", msg.ID)

		// Broadcast via WebSocket to sender's other devices and recipient
		wsMsg := &realtime.Message{
			Type: "message",
			Payload: MessageItem{
				ID:         msg.ID,
				Content:    msg.Content,
				SenderID:   msg.SenderID,
				SenderName: user.DisplayName,
				CreatedAt:  msg.CreatedAt,
				IsSent:     false, // Will be determined by recipient
			},
		}
		hub.SendToUser(recipientID, wsMsg)
		// Also send to sender's other devices (mark as sent)
		wsMsg.Payload = MessageItem{
			ID:         msg.ID,
			Content:    msg.Content,
			SenderID:   msg.SenderID,
			SenderName: user.DisplayName,
			CreatedAt:  msg.CreatedAt,
			IsSent:     true,
		}
		hub.SendToUser(user.ID, wsMsg)

		// Check if HTMX request - return HTML fragment
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusCreated)
			// Return message HTML that matches the template structure (escape content for XSS)
			escapedContent := html.EscapeString(msg.Content)
			htmlResp := `<div class="flex justify-end" data-message-id="` + strconv.FormatInt(msg.ID, 10) + `">
				<div class="max-w-xs lg:max-w-md px-4 py-2 rounded-lg bg-emerald-600 text-white">
					<p>` + escapedContent + `</p>
					<p class="text-xs mt-1 text-emerald-100">` + msg.CreatedAt + `</p>
				</div>
			</div>`
			w.Write([]byte(htmlResp))
			return
		}

		// Return created message as JSON for API clients
		response := MessageItem{
			ID:         msg.ID,
			Content:    msg.Content,
			SenderID:   msg.SenderID,
			SenderName: user.DisplayName,
			CreatedAt:  msg.CreatedAt,
			IsSent:     true,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			slog.Error("failed to encode message", "type", "request", "error", err)
		}

		_ = recipient // recipient info used above
	}
}

// getConversationsList fetches and deduplicates conversations for a user.
func getConversationsList(queries *store.Queries, ctx context.Context, userID int64) []ConversationListItem {
	rows, err := queries.GetRecentMessagePerUser(ctx, store.GetRecentMessagePerUserParams{
		SenderID:    userID,
		SenderID_2:  userID,
		RecipientID: userID,
	})
	if err != nil {
		slog.Error("failed to get conversations", "type", "request", "error", err)
		return []ConversationListItem{}
	}

	// Deduplicate by other user ID, keeping most recent (already sorted by created_at DESC)
	seen := make(map[int64]bool)
	var conversations []ConversationListItem

	for _, row := range rows {
		// Determine the other user's ID
		otherUserID := row.RecipientID
		if row.SenderID != userID {
			otherUserID = row.SenderID
		}

		if seen[otherUserID] {
			continue
		}
		seen[otherUserID] = true

		// Truncate message preview
		preview := row.Content
		if len(preview) > 50 {
			preview = preview[:47] + "..."
		}

		conversations = append(conversations, ConversationListItem{
			UserID:          otherUserID,
			DisplayName:     row.OtherUserDisplayName,
			LastMessage:     preview,
			LastMessageTime: row.CreatedAt,
		})
	}

	return conversations
}
