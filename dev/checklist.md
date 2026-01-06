# Development Checklist

A consolidated checklist for tracking implementation progress. Check items off as you complete and test each feature.

## Phase 1: Auth Foundation

### Setup
- [ ] Go module initialised
- [ ] Directory structure created
- [ ] Dependencies added to go.mod

### Database
- [ ] SQLite connection working
- [ ] WAL mode enabled
- [ ] Migrations system working
- [ ] Users table created
- [ ] Sessions table created

### Auth Logic
- [ ] Password hashing (bcrypt)
- [ ] Password verification
- [ ] Session token generation
- [ ] Session creation
- [ ] Session validation
- [ ] Session deletion

### Middleware
- [ ] RequireAuth middleware
- [ ] RequireAdmin middleware
- [ ] GetUser context helper

### Handlers & UI
- [ ] GET /login renders form
- [ ] POST /auth/login processes login
- [ ] POST /auth/logout clears session
- [ ] Login template styled

### Phase 1 Verification
- [ ] Server starts without error
- [ ] Database file created with correct tables
- [ ] Visiting `/` redirects to `/login`
- [ ] Login page renders
- [ ] Login with wrong credentials shows error
- [ ] Manually inserted user can log in
- [ ] After login, cookie is set
- [ ] After login, visiting `/login` redirects away
- [ ] Logout clears cookie and redirects to `/login`
- [ ] Expired session forces re-login

---

## Phase 2: User Management

### Admin CLI
- [ ] `--create-admin` flag implemented
- [ ] Prompts for credentials
- [ ] Creates admin user in database

### Admin Handlers
- [ ] GET /admin lists users
- [ ] POST /admin/users creates user
- [ ] POST /admin/users/:id updates user
- [ ] POST /admin/users/:id/delete removes user

### Admin UI
- [ ] Create user form
- [ ] User list table
- [ ] Edit functionality
- [ ] Delete with confirmation

### Users Endpoint
- [ ] GET /users returns all users (except self)

### Phase 2 Verification
- [ ] `--create-admin` creates admin user
- [ ] Admin can log in
- [ ] Admin can access `/admin`
- [ ] Non-admin gets 403 on `/admin`
- [ ] Admin can create new user
- [ ] New user appears in list
- [ ] Admin can edit user's display name
- [ ] Admin can change user's password
- [ ] Admin can toggle admin flag
- [ ] Admin can delete user
- [ ] Deleted user's sessions invalidated
- [ ] `/users` returns list for authenticated user

---

## Phase 3: Messaging (REST)

### Database
- [ ] Messages table migration
- [ ] Indexes created

### Message Model
- [ ] Message struct defined
- [ ] CreateMessage repository function
- [ ] GetConversation repository function
- [ ] GetConversationList repository function

### Message Handlers
- [ ] GET /conversations
- [ ] GET /conversations/:userID/messages
- [ ] POST /conversations/:userID/messages

### Chat UI
- [ ] Conversation list sidebar
- [ ] Message display area
- [ ] Message input and send
- [ ] New conversation button
- [ ] Sent/received visual distinction
- [ ] Polling for updates (temporary)

### Phase 3 Verification
- [ ] Messages table created
- [ ] User A can send message to User B
- [ ] User B sees message from User A
- [ ] Messages persist across refresh
- [ ] Conversation list shows users with history
- [ ] Conversation list ordered by most recent
- [ ] Can start new conversation
- [ ] Messages display in correct order
- [ ] Sent vs received visually distinguished
- [ ] Pagination works (50+ messages)
- [ ] Cannot send to non-existent user
- [ ] Cannot send empty message

---

## Phase 4: Real-Time Delivery

### Hub
- [ ] Hub struct with client map
- [ ] Register channel and handler
- [ ] Unregister channel and handler
- [ ] Broadcast channel and handler
- [ ] Hub.Run() goroutine

### Client
- [ ] Client struct
- [ ] ReadPump goroutine
- [ ] WritePump goroutine
- [ ] Ping/pong handling

### WebSocket Handler
- [ ] GET /ws endpoint
- [ ] Auth validation
- [ ] Connection upgrade
- [ ] Client registration

### Integration
- [ ] Message handler calls hub.SendToUser
- [ ] Sender's other devices receive message
- [ ] Recipient's devices receive message

### UI Updates
- [ ] WebSocket connection on page load
- [ ] Message event handling
- [ ] Real-time message append
- [ ] Conversation list updates
- [ ] Reconnection logic
- [ ] Polling code removed

### Phase 4 Verification
- [ ] WebSocket connects after login
- [ ] WebSocket rejects unauthenticated
- [ ] User A sends, User B receives instantly
- [ ] User A's two tabs both show sent message
- [ ] User B's two devices both receive
- [ ] Tab close/reopen reconnects
- [ ] Conversation list updates real-time
- [ ] New conversation appears in sidebar
- [ ] No polling in network tab
- [ ] No goroutine leaks on disconnect

---

## Phase 5: Cleanup and Hardening

### Background Jobs
- [ ] Message expiry (30 days)
- [ ] Session cleanup

### Security
- [ ] CSRF protection (or SameSite reliance documented)
- [ ] Input validation (username, display name, password, message)
- [ ] No SQL injection (parameterised queries)
- [ ] XSS prevention (template escaping)

### Production Config
- [ ] Secure cookie flag
- [ ] HTTPS via Caddy
- [ ] Systemd service file
- [ ] Backup script

### Mobile UI
- [ ] Responsive layout
- [ ] Touch-friendly targets
- [ ] Virtual keyboard handling

### Phase 5 Verification
- [ ] Old messages deleted (backdate to test)
- [ ] Expired sessions cleaned up
- [ ] Invalid input rejected with clear errors
- [ ] XSS attempt escaped
- [ ] Works over HTTPS
- [ ] Secure and HttpOnly cookie flags
- [ ] Usable on mobile phone
- [ ] Sidebar works on mobile
- [ ] Message sending works on mobile
- [ ] No console errors

---

## Deployment

- [ ] Binary builds successfully
- [ ] Systemd service configured
- [ ] Caddy reverse proxy configured
- [ ] HTTPS working
- [ ] Admin user created on production
- [ ] Backup automation set up
- [ ] Health check endpoint working