# API Reference

## Authentication

### GET /login

Renders the login page.

**Response:** HTML login form

**Behaviour:** Redirects to `/` if already authenticated.

---

### POST /auth/login

Authenticates a user and creates a session.

**Content-Type:** `application/x-www-form-urlencoded`

**Body:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| username | string | Yes | User's username |
| password | string | Yes | User's password |

**Success Response:**
- Sets `session` cookie
- Redirects to `/`

**Error Response:**
- Re-renders login page with error message

---

### POST /auth/logout

Ends the current session.

**Authentication:** Required

**Response:** Clears `session` cookie, redirects to `/login`

---

### GET /auth/me

Returns current user information.

**Authentication:** Required

**Response:** `200 OK`
```json
{
  "id": 1,
  "username": "logan",
  "displayName": "Logan",
  "isAdmin": true
}
```

**Error Response:** `401 Unauthorized` if not authenticated

---

## Users

### GET /users

Lists all users except the current user.

**Authentication:** Required

**Response:** `200 OK`
```json
[
  {
    "id": 2,
    "username": "jane",
    "displayName": "Jane"
  },
  {
    "id": 3,
    "username": "alex",
    "displayName": "Alex"
  }
]
```

---

## Admin

All admin endpoints require the user to have `is_admin = true`.

### GET /admin

Renders the admin user management page.

**Authentication:** Admin required

**Response:** HTML page with user list and create form

---

### POST /admin/users

Creates a new user.

**Authentication:** Admin required

**Content-Type:** `application/x-www-form-urlencoded`

**Body:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| username | string | Yes | 3-30 alphanumeric characters |
| display_name | string | Yes | 1-50 characters |
| password | string | Yes | Minimum 8 characters |
| is_admin | string | No | "on" if checked |

**Response:** Redirects to `/admin`

**Validation Errors:** Re-renders form with error message

---

### POST /admin/users/:id

Updates an existing user.

**Authentication:** Admin required

**URL Parameters:**
| Parameter | Description |
|-----------|-------------|
| id | User ID to update |

**Body:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| display_name | string | Yes | 1-50 characters |
| password | string | No | New password (leave empty to keep existing) |
| is_admin | string | No | "on" if checked |

**Response:** Redirects to `/admin`

---

### POST /admin/users/:id/delete

Deletes a user and invalidates their sessions.

**Authentication:** Admin required

**URL Parameters:**
| Parameter | Description |
|-----------|-------------|
| id | User ID to delete |

**Response:** Redirects to `/admin`

---

## Conversations

### GET /conversations

Lists all conversations for the current user.

**Authentication:** Required

**Response:** `200 OK`
```json
[
  {
    "userId": 2,
    "displayName": "Jane",
    "lastMessage": "See you tomorrow!",
    "lastMessageAt": "2025-01-06T14:32:00Z",
    "isLastMessageFromMe": false
  },
  {
    "userId": 3,
    "displayName": "Alex",
    "lastMessage": "Thanks for the help",
    "lastMessageAt": "2025-01-05T09:15:00Z",
    "isLastMessageFromMe": true
  }
]
```

---

### GET /conversations/:userID/messages

Retrieves messages in a conversation.

**Authentication:** Required

**URL Parameters:**
| Parameter | Description |
|-----------|-------------|
| userID | ID of the other user in the conversation |

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| limit | int | 50 | Number of messages to return |
| before | string | - | ISO timestamp, return messages before this time |

**Response:** `200 OK`
```json
[
  {
    "id": 1,
    "senderId": 1,
    "senderDisplayName": "Logan",
    "recipientId": 2,
    "content": "Hey, how are you?",
    "createdAt": "2025-01-06T14:30:00Z"
  },
  {
    "id": 2,
    "senderId": 2,
    "senderDisplayName": "Jane",
    "recipientId": 1,
    "content": "Good! See you tomorrow!",
    "createdAt": "2025-01-06T14:32:00Z"
  }
]
```

**Error Response:** `404 Not Found` if userID doesn't exist

---

### POST /conversations/:userID/messages

Sends a message to another user.

**Authentication:** Required

**URL Parameters:**
| Parameter | Description |
|-----------|-------------|
| userID | ID of the recipient |

**Content-Type:** `application/json` or `application/x-www-form-urlencoded`

**Body:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| content | string | Yes | Message text, 1-4000 characters |

**Response:** `201 Created`
```json
{
  "id": 3,
  "senderId": 1,
  "senderDisplayName": "Logan",
  "recipientId": 2,
  "content": "Hello!",
  "createdAt": "2025-01-06T15:00:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Empty or too long content
- `404 Not Found` - Recipient doesn't exist

---

## WebSocket

### GET /ws

Upgrades connection to WebSocket for real-time message delivery.

**Authentication:** Required (via session cookie)

**Protocol:** WebSocket

**Error Response:** `401 Unauthorized` if not authenticated

### Server → Client Messages

**New message notification:**
```json
{
  "type": "message",
  "id": 3,
  "from": 2,
  "fromName": "Jane",
  "content": "Hello!",
  "timestamp": "2025-01-06T15:00:00Z"
}
```

### Client → Server Messages

No client-to-server messages are used. Message sending is handled via REST API.

### Connection Lifecycle

1. Client connects to `/ws`
2. Server validates session cookie
3. Server registers client in hub
4. Server sends messages as they arrive
5. Client should handle reconnection on disconnect

### Reconnection Strategy

Recommended client behaviour:
- On disconnect, wait 2 seconds before reconnecting
- Use exponential backoff up to 30 seconds
- On successful reconnect, fetch recent messages to catch any missed during disconnect

---

## Health Check

### GET /health

Returns server health status.

**Authentication:** None

**Response:** `200 OK`
```json
{
  "status": "ok"
}
```