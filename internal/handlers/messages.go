package handlers

import (
	"net/http"

	"github.com/dukerupert/wantok/internal/render"
	"github.com/dukerupert/wantok/internal/store"
)

// ConversationListItem represents a conversation in the sidebar.
type ConversationListItem struct {
	UserID          int64  // The other user's ID
	DisplayName     string // The other user's display name
	LastMessage     string // Preview of the last message
	LastMessageTime string // Formatted timestamp
	IsUnread        bool   // TODO: Future enhancement
}

// MessageItem represents a single message in a conversation.
type MessageItem struct {
	ID          int64
	Content     string
	SenderID    int64
	SenderName  string
	CreatedAt   string
	IsSent      bool // true if current user sent this message
}

// ChatPageData holds data for the chat template.
type ChatPageData struct {
	Conversations    []ConversationListItem
	ActiveUserID     int64       // The user we're chatting with (0 if none selected)
	ActiveUserName   string      // Display name of active conversation
	Messages         []MessageItem
	CurrentUserID    int64
	CurrentUserName  string
}

// HandleChatPage renders the main chat interface.
// Route: GET /
// Notes:
//   - Fetch conversation list for sidebar using GetRecentMessagePerUser
//   - Group by other_user_id and take most recent message per user
//   - If conversationID query param provided, load that conversation's messages
//   - Otherwise show empty message area with prompt to select conversation
func HandleChatPage(queries *store.Queries, renderer *render.Renderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement
		// 1. Get current user from context
		// 2. Fetch conversations list (deduplicated by other user)
		// 3. Check for ?user= query param to load specific conversation
		// 4. If user param present, fetch messages for that conversation
		// 5. Render chat template with data
		http.Error(w, "Not implemented", http.StatusNotImplemented)
	}
}

// HandleGetConversations returns the conversation list as JSON.
// Route: GET /conversations
// Notes:
//   - Returns list of users the current user has messaged
//   - Includes last message preview and timestamp
//   - Ordered by most recent message first
//   - Used for HTMX partial updates of sidebar
func HandleGetConversations(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement
		// 1. Get current user from context
		// 2. Query GetRecentMessagePerUser
		// 3. Deduplicate by other_user_id, keeping most recent
		// 4. Return JSON array of ConversationListItem
		http.Error(w, "Not implemented", http.StatusNotImplemented)
	}
}

// HandleGetMessages returns messages for a conversation as JSON.
// Route: GET /conversations/{userID}/messages
// Notes:
//   - userID is the other participant in the conversation
//   - Supports pagination via ?limit= and ?offset= query params
//   - Default limit: 50, max limit: 100
//   - Returns messages in reverse chronological order (newest first)
//   - Client should reverse for display or use CSS flex-direction
func HandleGetMessages(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement
		// 1. Get current user from context
		// 2. Parse userID from path using r.PathValue("userID")
		// 3. Parse limit/offset from query params (with defaults)
		// 4. Validate userID exists and is not current user
		// 5. Query GetConversationMessages
		// 6. Transform to MessageItem slice (set IsSent based on sender_id)
		// 7. Return JSON array
		http.Error(w, "Not implemented", http.StatusNotImplemented)
	}
}

// HandleSendMessage creates a new message in a conversation.
// Route: POST /conversations/{userID}/messages
// Notes:
//   - userID is the recipient
//   - Request body: form-encoded with "content" field
//   - Validates: content not empty, recipient exists, not messaging self
//   - Returns created message as JSON (for optimistic UI update)
//   - Phase 4: Will trigger WebSocket broadcast to recipient
func HandleSendMessage(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement
		// 1. Get current user from context
		// 2. Parse userID from path
		// 3. Parse form to get content
		// 4. Validate: content not empty, not self-messaging
		// 5. Verify recipient exists (GetUserByID)
		// 6. Create message using CreateMessage query
		// 7. Return created message as JSON
		// 8. (Phase 4) Broadcast via WebSocket hub
		http.Error(w, "Not implemented", http.StatusNotImplemented)
	}
}
