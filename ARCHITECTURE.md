# Imposter Game - Architecture Design Document

> **Status**: Design Phase (No Code Yet)  
> **Stack**: Go Backend + Browser Clients (Desktop/Mobile)  
> **Theme**: Purple Neon Cyberpunk  

---

## Table of Contents

1. [Project Structure](#1-project-structure)
2. [Domain Model](#2-domain-model)
3. [WebSocket Protocol](#3-websocket-protocol)
4. [HTTP API Endpoints](#4-http-api-endpoints)
5. [Concurrency & State Management](#5-concurrency--state-management)
6. [Configuration & Environment](#6-configuration--environment)
7. [Deployment](#7-deployment)
8. [Local Development](#8-local-development)

---

## 1. Project Structure

```
imposter/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
│
├── internal/
│   ├── domain/                     # Pure game logic (no I/O dependencies)
│   │   ├── game.go                 # Game/GameSession entity
│   │   ├── player.go               # Player entity
│   │   ├── round.go                # Round management
│   │   ├── role.go                 # Role enum (Imposter, Vilek)
│   │   ├── phase.go                # Phase enum & state machine
│   │   ├── vote.go                 # Vote entity
│   │   ├── submission.go           # Word submission entity
│   │   ├── errors.go               # Domain-specific errors
│   │   └── events.go               # Domain events (for broadcasting)
│   │
│   ├── app/                        # Application/Service layer
│   │   ├── hub.go                  # GameHub - manages all active games
│   │   ├── session.go              # GameSession wrapper with concurrency
│   │   ├── broadcaster.go          # Handles broadcasting to players
│   │   └── words.go                # Secret word list/generator
│   │
│   ├── transport/
│   │   ├── http/
│   │   │   ├── server.go           # HTTP server setup
│   │   │   ├── handlers.go         # REST handlers
│   │   │   ├── middleware.go       # Logging, CORS, etc.
│   │   │   └── responses.go        # JSON response helpers
│   │   │
│   │   └── ws/
│   │       ├── handler.go          # WebSocket upgrade & connection
│   │       ├── client.go           # Per-player WebSocket connection
│   │       ├── messages.go         # Message type definitions
│   │       └── router.go           # Routes incoming WS messages to handlers
│   │
│   └── config/
│       └── config.go               # Configuration loading
│
├── web/                            # Frontend assets
│   ├── static/
│   │   ├── css/
│   │   │   └── style.css           # Purple neon cyberpunk styles
│   │   ├── js/
│   │   │   ├── app.js              # Main application logic
│   │   │   ├── websocket.js        # WebSocket client wrapper
│   │   │   └── ui.js               # UI manipulation helpers
│   │   └── assets/
│   │       └── fonts/              # Custom fonts (if any)
│   │
│   └── index.html                  # Single-page application shell
│
├── scripts/
│   ├── deploy.sh                   # Deployment script
│   └── dev.sh                      # Local development runner
│
├── deployments/
│   ├── systemd/
│   │   └── imposter.service        # systemd unit file
│   └── nginx/
│       └── imposter.conf           # Nginx reverse proxy config
│
├── .env.example                    # Example environment variables
├── go.mod
├── go.sum
├── Makefile                        # Build, test, run commands
└── README.md
```

---

## 2. Domain Model

### 2.1 Enums

```go
// internal/domain/role.go
type Role string

const (
    RoleImposter Role = "IMPOSTER"
    RoleVilek    Role = "VILEK"
)
```

```go
// internal/domain/phase.go
type Phase string

const (
    PhaseLobby         Phase = "LOBBY"           // Waiting for players
    PhaseRoleAssignment Phase = "ROLE_ASSIGNMENT" // Roles just assigned, showing to players
    PhaseSubmission    Phase = "SUBMISSION"       // Players submitting words one by one
    PhaseVoting        Phase = "VOTING"           // 20s countdown, everyone votes
    PhaseResults       Phase = "RESULTS"          // Show votes & winner
)
```

```go
// internal/domain/player.go
type ConnectionStatus string

const (
    StatusConnected    ConnectionStatus = "CONNECTED"
    StatusDisconnected ConnectionStatus = "DISCONNECTED"
)
```

### 2.2 Core Entities

```go
// internal/domain/player.go
type Player struct {
    ID           string           // Unique player ID (UUID)
    Nickname     string           // Display name
    Role         Role             // Assigned role for current round
    HasVoted     bool             // Whether player has voted this round
    HasSubmitted bool             // Whether player has submitted this round
    Status       ConnectionStatus // Connection status
    JoinedAt     time.Time
}
```

```go
// internal/domain/submission.go
type Submission struct {
    PlayerID  string
    Word      string
    Timestamp time.Time
    Order     int       // 1-based order in submission sequence
}
```

```go
// internal/domain/vote.go
type Vote struct {
    VoterID   string    // Who cast the vote
    TargetID  string    // Who they voted for
    Timestamp time.Time
}
```

```go
// internal/domain/round.go
type Round struct {
    Number           int
    SecretWord       string        // The word VILEKs see
    ImposterID       string        // Player ID of the Imposter
    Submissions      []Submission  // Ordered list of submissions
    Votes            []Vote        // All votes cast
    CurrentPlayerIdx int           // Index in player order for submissions
    PlayerOrder      []string      // Order of player IDs for submission phase
    Winner           Role          // Set after voting phase ends
    StartedAt        time.Time
    EndedAt          time.Time
}
```

```go
// internal/domain/game.go
type Game struct {
    ID           string              // Room code (e.g., "NEON42")
    HostID       string              // Player ID of the host
    Players      map[string]*Player  // playerID -> Player
    CurrentRound *Round
    RoundHistory []*Round
    Phase        Phase
    CreatedAt    time.Time
    Settings     GameSettings
}

type GameSettings struct {
    MinPlayers     int           // Default: 4
    MaxPlayers     int           // Default: 10
    VotingDuration time.Duration // Default: 20s
    RoleRevealTime time.Duration // Default: 5s (time to show role before submissions)
}
```

### 2.3 Domain Events

```go
// internal/domain/events.go
type EventType string

const (
    EventPlayerJoined      EventType = "PLAYER_JOINED"
    EventPlayerLeft        EventType = "PLAYER_LEFT"
    EventPlayerReconnected EventType = "PLAYER_RECONNECTED"
    EventGameStarted       EventType = "GAME_STARTED"
    EventRolesAssigned     EventType = "ROLES_ASSIGNED"
    EventSubmissionMade    EventType = "SUBMISSION_MADE"
    EventVotingStarted     EventType = "VOTING_STARTED"
    EventVoteCast          EventType = "VOTE_CAST"
    EventRoundEnded        EventType = "ROUND_ENDED"
    EventGameEnded         EventType = "GAME_ENDED"
)

type GameEvent struct {
    Type      EventType
    GameID    string
    Payload   interface{}
    Timestamp time.Time
}
```

### 2.4 Domain Errors

```go
// internal/domain/errors.go
var (
    ErrGameNotFound       = errors.New("game not found")
    ErrGameFull           = errors.New("game is full")
    ErrGameAlreadyStarted = errors.New("game already started")
    ErrNotEnoughPlayers   = errors.New("not enough players to start")
    ErrNotYourTurn        = errors.New("not your turn to submit")
    ErrAlreadySubmitted   = errors.New("already submitted this round")
    ErrAlreadyVoted       = errors.New("already voted this round")
    ErrInvalidPhase       = errors.New("invalid action for current phase")
    ErrPlayerNotFound     = errors.New("player not found")
    ErrNotHost            = errors.New("only host can perform this action")
    ErrCannotVoteSelf     = errors.New("cannot vote for yourself")
)
```

---

## 3. WebSocket Protocol

All WebSocket messages use JSON with a consistent envelope:

### 3.1 Message Envelope

```typescript
// Client → Server
interface ClientMessage {
    type: string;
    payload?: any;
}

// Server → Client
interface ServerMessage {
    type: string;
    payload?: any;
    timestamp: string; // ISO 8601
}
```

### 3.2 Client → Server Messages

| Type | Payload | Description |
|------|---------|-------------|
| `join_lobby` | `{ nickname: string }` | Join game lobby with nickname |
| `start_game` | `{}` | Host starts the game |
| `submit_word` | `{ word: string }` | Submit a word during submission phase |
| `cast_vote` | `{ targetPlayerId: string }` | Vote for a player |
| `request_new_round` | `{}` | Host requests another round |
| `ping` | `{}` | Keepalive ping |

### 3.3 Server → Client Messages

| Type | Payload | Description |
|------|---------|-------------|
| `connected` | `{ playerId, gameId, gameState }` | Connection confirmed |
| `error` | `{ code, message }` | Error response |
| `lobby_update` | `{ players[], hostId, canStart }` | Lobby state changed |
| `game_started` | `{}` | Game has started |
| `role_assigned` | `{ role, secretWord? }` | Your role (and word if VILEK) |
| `submission_phase` | `{ currentPlayerId, playerOrder, submissions[] }` | Submission phase state |
| `submission_update` | `{ submissions[], currentPlayerId, isComplete }` | New submission made |
| `voting_phase` | `{ remainingSeconds, players[] }` | Voting started |
| `voting_countdown` | `{ remainingSeconds }` | Countdown tick |
| `vote_update` | `{ votedCount, totalPlayers }` | Vote progress (no reveal who) |
| `round_results` | `{ votes[], imposterId, winner, secretWord }` | Round finished |
| `player_disconnected` | `{ playerId, nickname }` | Player disconnected |
| `player_reconnected` | `{ playerId, nickname }` | Player reconnected |
| `pong` | `{}` | Keepalive response |

### 3.4 Example Message Flows

#### Join Game Flow
```
Client                          Server
   |                               |
   |---[WS Connect]--------------->|
   |                               |
   |<--[connected]-----------------|  {playerId, gameId, gameState}
   |                               |
   |---[join_lobby]--------------->|  {nickname: "CyberNinja"}
   |                               |
   |<--[lobby_update]--------------|  {players: [...], canStart: true}
   |                               |  (broadcast to all)
```

#### Submission Flow
```
Client (P1)                     Server                      Client (P2)
   |                               |                            |
   |---[submit_word]-------------->|                            |
   |   {word: "laser"}             |                            |
   |                               |                            |
   |<--[submission_update]---------|---[submission_update]----->|
   |   {currentPlayerId: "P2"}     |   {currentPlayerId: "P2"}  |
```

#### Voting Flow
```
All Clients                     Server
   |                               |
   |<--[voting_phase]--------------|  {remainingSeconds: 20, players: [...]}
   |                               |
   |<--[voting_countdown]----------|  (every second)
   |                               |
   |---[cast_vote]---------------->|
   |                               |
   |<--[vote_update]---------------|  {votedCount: 3, totalPlayers: 5}
   |                               |
   |<--[round_results]-------------|  (after 20s or all voted)
```

---

## 4. HTTP API Endpoints

### 4.1 REST Endpoints

| Method | Path | Description | Request Body | Response |
|--------|------|-------------|--------------|----------|
| `GET` | `/` | Serve index.html | - | HTML |
| `GET` | `/static/*` | Serve static assets | - | File |
| `POST` | `/api/rooms` | Create new room | `{}` | `{ roomCode, inviteLink }` |
| `GET` | `/api/rooms/:roomCode` | Get room info | - | `{ roomCode, playerCount, phase, canJoin }` |
| `GET` | `/api/rooms/:roomCode/exists` | Check if room exists | - | `{ exists: bool }` |
| `GET` | `/api/health` | Health check | - | `{ status: "ok" }` |

### 4.2 WebSocket Endpoint

| Path | Query Params | Description |
|------|--------------|-------------|
| `/ws` | `roomCode`, `playerId?` | WebSocket upgrade for game connection |

**Connection Logic:**
- If `playerId` is provided and valid → attempt reconnection
- If `playerId` is missing → new player connection
- If `roomCode` invalid → close with error

### 4.3 Response Formats

```json
// Success
{
    "success": true,
    "data": { ... }
}

// Error
{
    "success": false,
    "error": {
        "code": "ROOM_NOT_FOUND",
        "message": "Room with code XYZ123 does not exist"
    }
}
```

### 4.4 Invite Link Format

```
https://imposter.yourdomain.com/join/{roomCode}

Examples:
- https://imposter.yourdomain.com/join/NEON42
- https://imposter.yourdomain.com/join/CYBER99
```

**Route handling:** `/join/:roomCode` serves `index.html`; the frontend reads the room code from URL and initiates join flow.

---

## 5. Concurrency & State Management

### 5.1 GameHub (Central Manager)

```go
// internal/app/hub.go
type GameHub struct {
    games    map[string]*GameSession  // roomCode -> session
    mu       sync.RWMutex
    
    // Configuration
    roomCodeLength int
    maxGamesPerHub int
}

// Key methods:
// - CreateGame() (*GameSession, error)
// - GetGame(roomCode string) (*GameSession, error)
// - DeleteGame(roomCode string)
// - CleanupStaleGames() // Called periodically
```

### 5.2 GameSession (Per-Game Wrapper)

```go
// internal/app/session.go
type GameSession struct {
    game        *domain.Game
    mu          sync.RWMutex
    
    clients     map[string]*ws.Client  // playerID -> WebSocket client
    clientsMu   sync.RWMutex
    
    broadcast   chan GameEvent
    done        chan struct{}
    
    timers      *SessionTimers
}

type SessionTimers struct {
    votingTimer    *time.Timer
    countdownTick  *time.Ticker
}
```

### 5.3 Concurrency Patterns

1. **Read-heavy operations** (get game state): Use `RLock()`
2. **Write operations** (submit, vote): Use `Lock()`
3. **Broadcasts**: Send to channel, dedicated goroutine handles fan-out
4. **Timers**: Use `time.AfterFunc` for voting countdown

```go
// Example: Submission flow
func (s *GameSession) SubmitWord(playerID, word string) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    // 1. Validate (domain logic)
    err := s.game.SubmitWord(playerID, word)
    if err != nil {
        return err
    }
    
    // 2. Check if phase transition needed
    if s.game.AllPlayersSubmitted() {
        s.game.TransitionTo(domain.PhaseVoting)
        s.startVotingTimer()
    }
    
    // 3. Queue broadcast (non-blocking)
    s.broadcast <- s.buildSubmissionUpdate()
    
    return nil
}
```

### 5.4 Player Connection Tracking

```go
// internal/transport/ws/client.go
type Client struct {
    conn      *websocket.Conn
    playerID  string
    gameID    string
    session   *app.GameSession
    
    send      chan []byte    // Outbound messages
    done      chan struct{}  // Close signal
}
```

**Reconnection Logic:**
1. Player disconnects → mark as `DISCONNECTED`, keep state
2. Same `playerId` connects within grace period (e.g., 2 min) → restore session
3. Grace period expires → remove player from game

---

## 6. Configuration & Environment

### 6.1 Environment Variables

```bash
# .env.example

# Server
PORT=8080
HOST=0.0.0.0
ENV=development  # development | production

# Game Settings
MIN_PLAYERS=4
MAX_PLAYERS=10
VOTING_DURATION_SECONDS=20
RECONNECT_GRACE_PERIOD_SECONDS=120

# Security
ROOM_CODE_LENGTH=6

# Logging
LOG_LEVEL=info  # debug | info | warn | error
LOG_FORMAT=json  # json | text

# Optional: Future database
# DATABASE_URL=postgres://user:pass@host:5432/imposter
```

### 6.2 Config Struct

```go
// internal/config/config.go
type Config struct {
    Server   ServerConfig
    Game     GameConfig
    Security SecurityConfig
    Logging  LoggingConfig
}

type ServerConfig struct {
    Port string
    Host string
    Env  string // "development" or "production"
}

type GameConfig struct {
    MinPlayers            int
    MaxPlayers            int
    VotingDurationSeconds int
    ReconnectGracePeriod  time.Duration
}

type SecurityConfig struct {
    RoomCodeLength int
}

type LoggingConfig struct {
    Level  string
    Format string
}

func Load() (*Config, error) {
    // Load from environment, with defaults
}
```

---

## 7. Deployment

### 7.1 systemd Service

```ini
# deployments/systemd/imposter.service
[Unit]
Description=Imposter Game Server
After=network.target

[Service]
Type=simple
User=imposter
Group=imposter
WorkingDirectory=/opt/imposter
ExecStart=/opt/imposter/bin/server
Restart=on-failure
RestartSec=5
EnvironmentFile=/opt/imposter/.env

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/imposter

[Install]
WantedBy=multi-user.target
```

### 7.2 Nginx Configuration

```nginx
# deployments/nginx/imposter.conf
upstream imposter_backend {
    server 127.0.0.1:8080;
    keepalive 32;
}

server {
    listen 80;
    server_name imposter.yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name imposter.yourdomain.com;

    # SSL (Let's Encrypt via certbot)
    ssl_certificate /etc/letsencrypt/live/imposter.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/imposter.yourdomain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers off;

    # Static assets (optional: serve directly from nginx for perf)
    location /static/ {
        alias /opt/imposter/web/static/;
        expires 1d;
        add_header Cache-Control "public, immutable";
    }

    # WebSocket endpoint
    location /ws {
        proxy_pass http://imposter_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400;  # 24h for long-lived connections
    }

    # API and everything else
    location / {
        proxy_pass http://imposter_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 7.3 Alternative: Caddy (Simpler)

```caddyfile
# Caddyfile
imposter.yourdomain.com {
    # Automatic HTTPS via Let's Encrypt
    
    # Static files
    handle /static/* {
        root * /opt/imposter/web
        file_server
    }
    
    # Reverse proxy to Go backend
    handle {
        reverse_proxy localhost:8080
    }
}
```

### 7.4 Build & Deploy Script

```bash
#!/bin/bash
# scripts/deploy.sh

set -e

SERVER="your-lightsail-ip"
DEPLOY_PATH="/opt/imposter"

# Build
echo "Building..."
GOOS=linux GOARCH=amd64 go build -o bin/server ./cmd/server

# Deploy
echo "Deploying to $SERVER..."
rsync -avz --delete \
    bin/server \
    web/ \
    .env.production \
    $SERVER:$DEPLOY_PATH/

# Restart
ssh $SERVER "sudo systemctl restart imposter"

echo "Deployed!"
```

---

## 8. Local Development

### 8.1 Running Locally

```bash
# Option 1: Direct run
go run ./cmd/server

# Option 2: With hot reload (using air)
air

# Option 3: Make command
make run
```

### 8.2 Makefile

```makefile
# Makefile

.PHONY: build run test clean

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

clean:
	rm -rf bin/

lint:
	golangci-lint run

# Development with hot reload
dev:
	air
```

### 8.3 Testing Multiple Players

```bash
# Terminal 1: Start server
make run

# Browser windows:
# 1. Normal window      → http://localhost:8080 (create room)
# 2. Incognito window   → http://localhost:8080/join/ROOMCODE
# 3. Different browser  → http://localhost:8080/join/ROOMCODE
# 4. Another incognito  → http://localhost:8080/join/ROOMCODE
```

### 8.4 Test Structure

```
internal/
├── domain/
│   ├── game_test.go         # Unit tests for game logic
│   ├── round_test.go        # Role assignment, winner calculation
│   └── phase_test.go        # State machine transitions
│
├── app/
│   ├── hub_test.go          # Game creation, cleanup
│   └── session_test.go      # Concurrency tests
│
└── transport/
    └── ws/
        └── integration_test.go  # WebSocket client simulation
```

### 8.5 Example Domain Tests

```go
// internal/domain/game_test.go

func TestRoleAssignment(t *testing.T) {
    game := NewGame("TEST01")
    
    // Add 5 players
    for i := 0; i < 5; i++ {
        game.AddPlayer(fmt.Sprintf("player%d", i), fmt.Sprintf("Nick%d", i))
    }
    
    // Start round
    err := game.StartRound("banana")
    require.NoError(t, err)
    
    // Verify exactly 1 imposter
    imposterCount := 0
    for _, p := range game.Players {
        if p.Role == RoleImposter {
            imposterCount++
        }
    }
    assert.Equal(t, 1, imposterCount)
}

func TestWinnerCalculation(t *testing.T) {
    tests := []struct {
        name       string
        votes      map[string]string  // voterID -> targetID
        imposterID string
        expected   Role
    }{
        {
            name: "vileks win - imposter caught",
            votes: map[string]string{
                "p1": "p3",  // imposter
                "p2": "p3",
                "p3": "p1",
                "p4": "p3",
            },
            imposterID: "p3",
            expected:   RoleVilek,
        },
        {
            name: "imposter wins - not caught",
            votes: map[string]string{
                "p1": "p2",
                "p2": "p1",
                "p3": "p4",
                "p4": "p1",
            },
            imposterID: "p3",
            expected:   RoleImposter,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            winner := CalculateWinner(tt.votes, tt.imposterID)
            assert.Equal(t, tt.expected, winner)
        })
    }
}
```

---

## Appendix A: Secret Word List

Store a curated list of words that work well for the game:

```go
// internal/app/words.go
var secretWords = []string{
    // Animals
    "dragon", "phoenix", "unicorn", "kraken",
    // Tech
    "hacker", "cyborg", "android", "hologram",
    // Cyberpunk themed
    "neon", "chrome", "synth", "glitch", "matrix",
    // Objects
    "laser", "plasma", "quantum", "binary",
    // Places
    "arcade", "casino", "subway", "rooftop",
    // ... 100+ words
}

func GetRandomWord() string {
    return secretWords[rand.Intn(len(secretWords))]
}
```

---

## Appendix B: Room Code Generation

```go
// Safe characters (no ambiguous chars like 0/O, 1/I/l)
const roomCodeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func GenerateRoomCode(length int) string {
    b := make([]byte, length)
    for i := range b {
        b[i] = roomCodeChars[rand.Intn(len(roomCodeChars))]
    }
    return string(b)
}
```

---

## Appendix C: Frontend UI Components (Conceptual)

```
Screens:
├── HomeScreen
│   ├── Logo + Title (neon glow effect)
│   ├── [Create Room] button
│   └── [Join Room] input + button
│
├── LobbyScreen
│   ├── Room Code display (big, copyable)
│   ├── Invite Link + [Copy] button
│   ├── Player list (avatars with neon borders)
│   └── [Start Game] button (host only)
│
├── RoleRevealScreen
│   ├── Full-screen role reveal
│   ├── "You are VILEK" + secret word (glitch animation)
│   └── OR "You are IMPOSTER" (red neon, dramatic)
│
├── SubmissionScreen
│   ├── Current player highlight
│   ├── Word input (when your turn)
│   ├── Submitted words list (updates live)
│   └── Waiting indicator (when not your turn)
│
├── VotingScreen
│   ├── Countdown timer (20s, neon digits)
│   ├── Player cards (clickable to vote)
│   └── Vote confirmation
│
└── ResultsScreen
    ├── Vote breakdown (who voted for whom)
    ├── Imposter reveal (dramatic animation)
    ├── Winner announcement (VILEKS WIN / IMPOSTER WINS)
    └── [Play Again] button
```

---

## Summary Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                          BROWSER                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │   HTML/CSS/JS (Purple Neon Cyberpunk UI)                │    │
│  │   ┌─────────────────┐  ┌─────────────────────────────┐  │    │
│  │   │  HTTP Client    │  │  WebSocket Client           │  │    │
│  │   │  (fetch API)    │  │  (native WebSocket)         │  │    │
│  │   └────────┬────────┘  └────────────┬────────────────┘  │    │
│  └────────────┼─────────────────────────┼──────────────────┘    │
└───────────────┼─────────────────────────┼───────────────────────┘
                │                         │
                │ HTTPS                   │ WSS
                │                         │
┌───────────────┼─────────────────────────┼───────────────────────┐
│               │      NGINX / CADDY      │                       │
│               │        (TLS + Proxy)    │                       │
└───────────────┼─────────────────────────┼───────────────────────┘
                │                         │
                ▼                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                      GO BACKEND (Single Binary)                  │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    TRANSPORT LAYER                       │    │
│  │  ┌──────────────────┐    ┌────────────────────────────┐ │    │
│  │  │  HTTP Handlers   │    │  WebSocket Handler          │ │    │
│  │  │  /api/rooms      │    │  /ws                        │ │    │
│  │  │  /static/*       │    │  Message Router             │ │    │
│  │  └────────┬─────────┘    └─────────────┬──────────────┘ │    │
│  └───────────┼────────────────────────────┼────────────────┘    │
│              │                            │                      │
│              ▼                            ▼                      │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                    APPLICATION LAYER                     │    │
│  │  ┌──────────────────────────────────────────────────┐   │    │
│  │  │  GameHub                                          │   │    │
│  │  │  ├── games map[roomCode]*GameSession             │   │    │
│  │  │  └── CreateGame / GetGame / DeleteGame           │   │    │
│  │  └──────────────────────────────────────────────────┘   │    │
│  │  ┌──────────────────────────────────────────────────┐   │    │
│  │  │  GameSession                                      │   │    │
│  │  │  ├── Game state + mutex                          │   │    │
│  │  │  ├── WebSocket clients                           │   │    │
│  │  │  └── Broadcast channel                           │   │    │
│  │  └──────────────────────────────────────────────────┘   │    │
│  └─────────────────────────────────────────────────────────┘    │
│              │                                                   │
│              ▼                                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                     DOMAIN LAYER                         │    │
│  │  ┌────────┐ ┌────────┐ ┌───────┐ ┌──────┐ ┌──────────┐  │    │
│  │  │  Game  │ │ Player │ │ Round │ │ Vote │ │Submission│  │    │
│  │  └────────┘ └────────┘ └───────┘ └──────┘ └──────────┘  │    │
│  │  Pure Go logic • No I/O • No HTTP/WS knowledge          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌───────────────────┐
                    │  AWS Lightsail    │
                    │  Ubuntu Instance  │
                    │  + systemd        │
                    └───────────────────┘
```

---

**Next Steps (Implementation Order):**

1. ✅ Architecture document (this file)
2. 🔜 Set up Go module and project structure
3. 🔜 Implement domain layer (pure game logic)
4. 🔜 Implement application layer (hub, sessions)
5. 🔜 Implement transport layer (HTTP + WebSocket)
6. 🔜 Build frontend (HTML/CSS/JS with cyberpunk theme)
7. 🔜 Local testing with multiple browser windows
8. 🔜 Deploy to Lightsail

