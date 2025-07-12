<div align="center">
  <p><strong>ThrowBackChat: an IRC-style chat application with modern web technologies</strong></p>

</div>

*This project was built for a vibe coding presentation entirely with Claude Code*

ThrowBackChat is a real-time chat application that brings back the simplicity and charm of IRC
with modern web technologies. Built with a Go backend and SolidJS frontend, it features
channel-based conversations, operator permissions, and real-time messaging via WebSockets.

The application supports classic IRC commands like `/kick`, `/topic`, `/join`, `/leave`, `/me`, 
and `/nick`, while providing a clean web interface. Users can take possession of nicknames 
without signup, join channels, and engage in real-time conversations with markdown support.

**Features:**

* Real-time messaging via WebSockets
* Channel-based conversations with operator permissions  
* Classic IRC-style commands (`/kick`, `/topic`, `/join`, etc.)
* Session management with heartbeat monitoring
* Markdown support in messages
* No signup required - take possession of any available nickname
* Automatic channel creation (first joiner becomes operator)
* Clean, responsive web interface built with SolidJS

## Quick Start

**Development:**

```bash
# Install dependencies
make install

# Start development environment (both backend and frontend)
make dev

# View logs
make tail-log
```

**Building:**

```bash
# Build production binary
make build

# Format code
make format

# Run static analysis and type checking
make check
```

## Architecture

**Backend (Go):**
- Chi router with WebSocket support
- SQLite database with migrations
- Real-time messaging via Gorilla WebSocket
- Session management with heartbeat monitoring

**Frontend (SolidJS):**
- TypeScript + TailwindCSS v4
- Vite development server with API proxy
- Markdown rendering with `marked`
- Responsive design

**Key Components:**
- `cmd/server/main.go` - Application entry point
- `internal/chat/` - Chat management and business logic
- `internal/web/` - HTTP handlers and WebSocket management
- `internal/db/` - Database operations and migrations
- `web/` - SolidJS frontend application

## Configuration

Environment variables (create a `.env` file):

```bash
TBCHAT_PORT=8080          # Server port (default: 8080)
TBCHAT_HOST=0.0.0.0       # Server host (default: 0.0.0.0)
TBCHAT_DB=chat.db         # SQLite database path (default: chat.db)
```

## IRC Commands

ThrowBackChat supports classic IRC commands:

- `/join #channel` - Join a channel
- `/leave` - Leave current channel
- `/nick newname` - Change nickname
- `/me action` - Send action message
- `/topic new topic` - Set channel topic (operators only)
- `/kick username` - Kick user from channel (operators only)

## Technology Stack

- **Backend**: Go, Chi router, Gorilla WebSocket, SQLite
- **Frontend**: SolidJS, TypeScript, TailwindCSS, Vite
- **Database**: SQLite with migration system
