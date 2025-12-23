# 🎭 Imposter Game

A real-time multiplayer browser game where players try to identify the imposter among them.

**Stack:** Go backend • Vanilla JS frontend • WebSockets  
**Theme:** Purple Neon Cyberpunk 🌆  
**Live:** [imposter.fdecono.com](https://imposter.fdecono.com)

---

## 🎮 How to Play

1. **Create a room** - One player creates a game room
2. **Share the link** - Other players (4-10 total) join via invite link
3. **Roles assigned** - Each round, one player becomes the **IMPOSTER**, others are **VILEKs**
4. **VILEKs** see a secret word, the **Imposter** does not
5. **Word submission** - Players take turns submitting one word each (related to the secret)
6. **Voting** - After all submissions, players have 20 seconds to vote for who they think is the Imposter
7. **Results** - If the Imposter gets the most votes, VILEKs win. Otherwise, Imposter wins!

---

## 🚀 Quick Start

### Prerequisites

- Go 1.22+
- Docker (for deployment)

### Run Locally

```bash
# Clone and enter directory
cd imposter

# Run the server
go run ./cmd/server
```

Open http://localhost:8080 in your browser.

### Testing Multiple Players

1. Open http://localhost:8080 in a normal browser window (create a room)
2. Copy the invite link
3. Open incognito windows or different browsers
4. Paste the invite link to join as different players

---

## 📁 Project Structure

```
imposter/
├── cmd/server/
│   ├── main.go              # Application entry point
│   └── web/                  # Frontend assets (embedded)
│       ├── index.html
│       └── static/
│           ├── css/style.css
│           └── js/app.js
├── internal/
│   ├── domain/               # Pure game logic (no I/O)
│   │   ├── game.go
│   │   ├── player.go
│   │   ├── round.go
│   │   ├── phase.go
│   │   └── events.go
│   ├── app/                  # Application layer (hub, sessions)
│   │   ├── hub.go
│   │   ├── session.go
│   │   └── words.go
│   ├── transport/
│   │   ├── http/             # HTTP handlers
│   │   └── ws/               # WebSocket handlers
│   └── config/               # Configuration loading
├── Dockerfile                # Docker build configuration
├── Makefile                  # Build commands
└── ARCHITECTURE.md           # Detailed design documentation
```

---

## 🧪 Development

```bash
# Build
make build

# Run
make run

# Run tests
make test

# Run with coverage
make test-coverage

# Lint code
make lint
```
